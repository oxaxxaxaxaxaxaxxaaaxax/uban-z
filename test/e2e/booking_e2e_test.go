//go:build integration

package e2e_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

var (
	migrationsDir = filepath.Join("..", "..", "migrations", "booking")
	jwtSecret     = []byte("e2e-test-secret")
)

type e2eServer struct {
	URL string
}

func bootServer(t *testing.T) *e2eServer {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		_ = db.Close()
		t.Fatalf("goose up: %v", err)
	}
	_ = db.Close()

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("pgxpool.ParseConfig: %v", err)
	}
	// Default MaxConns is max(4, num_cpus). The race test fires 10 parallel
	// requests; with the default size most of them queue behind 4 connections
	// and time out under load when multiple integration suites run back-to-back.
	poolCfg.MaxConns = 25
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatalf("pgxpool.NewWithConfig: %v", err)
	}
	t.Cleanup(pool.Close)
	seedE2ERooms(t, pool)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := bookingpostgres.NewStoreFromPool(pool)
	useCase := service.New(store, store, nil)
	handler := bookinghttp.NewHandler(useCase, logger)

	router := httpx.Chain(
		bookingserver.Handler(handler),
		httpx.ParseToken(jwtSecret),
		httpx.RequestID,
		httpx.RecoverPanic(logger),
		httpx.AccessLog(logger),
	)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &e2eServer{URL: srv.URL}
}

func seedE2ERooms(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(context.Background(), `
		INSERT INTO rooms (name, capacity, building) VALUES
			('A101', 30, 'НГУ'),
			('B204', 30, 'НГУ'),
			('C305', 30, 'НГУ')
		ON CONFLICT (building, name) DO UPDATE
		SET capacity = EXCLUDED.capacity
	`)
	if err != nil {
		t.Fatalf("seed e2e rooms: %v", err)
	}
}

func mintToken(t *testing.T, userID int, login string, role domain.Role) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   strconv.Itoa(userID),
		"login": login,
		"role":  string(role),
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func doJSON(t *testing.T, method, url, token string, body any) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return resp, payload
}

func TestE2E_GetRoomsAnonymous(t *testing.T) {
	srv := bootServer(t)

	resp, body := doJSON(t, http.MethodGet, srv.URL+"/rooms", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
	}
	var rooms []bookingserver.Room
	if err := json.Unmarshal(body, &rooms); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rooms) != 3 {
		t.Fatalf("got %d rooms, want 3 test-seeded", len(rooms))
	}
}

func TestE2E_WriteEndpointsRequireToken(t *testing.T) {
	srv := bootServer(t)

	base := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
	createBody := map[string]any{
		"room_id":    1,
		"start_time": base.Format(time.RFC3339),
		"end_time":   base.Add(time.Hour).Format(time.RFC3339),
	}

	resp, _ := doJSON(t, http.MethodPost, srv.URL+"/booking", "", createBody)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST without token = %d, want 401", resp.StatusCode)
	}

	resp, _ = doJSON(t, http.MethodDelete, srv.URL+"/booking/1", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("DELETE without token = %d, want 401", resp.StatusCode)
	}

	resp, _ = doJSON(t, http.MethodGet, srv.URL+"/booking/my", "", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /booking/my without token = %d, want 401", resp.StatusCode)
	}
}

func TestE2E_GetBookingMy(t *testing.T) {
	srv := bootServer(t)

	alice := mintToken(t, 30, "alice", domain.RoleStudentB)
	bob := mintToken(t, 31, "bob", domain.RoleStudentB)
	base := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)

	create := func(token string, roomID int, start, end time.Time) {
		t.Helper()
		resp, body := doJSON(t, http.MethodPost, srv.URL+"/booking", token, map[string]any{
			"room_id":    roomID,
			"start_time": start.Format(time.RFC3339),
			"end_time":   end.Format(time.RFC3339),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create status = %d; body=%s", resp.StatusCode, body)
		}
	}

	create(alice, 1, base, base.Add(time.Hour))
	create(alice, 2, base.Add(2*time.Hour), base.Add(3*time.Hour))
	create(bob, 1, base.Add(4*time.Hour), base.Add(5*time.Hour))

	resp, body := doJSON(t, http.MethodGet, srv.URL+"/booking/my", alice, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /booking/my = %d; body=%s", resp.StatusCode, body)
	}
	var bookings []bookingserver.Booking
	if err := json.Unmarshal(body, &bookings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(bookings) != 2 {
		t.Fatalf("alice has %d bookings, want 2", len(bookings))
	}

	resp, body = doJSON(t, http.MethodGet, srv.URL+"/booking/my", bob, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /booking/my (bob) = %d; body=%s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, &bookings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(bookings) != 1 {
		t.Fatalf("bob has %d bookings, want 1", len(bookings))
	}
}

func TestE2E_BookingLifecycle(t *testing.T) {
	srv := bootServer(t)

	studentB := mintToken(t, 10, "alice", domain.RoleStudentB)
	base := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)

	create := func(token string, roomID int, start, end time.Time) (*http.Response, []byte) {
		return doJSON(t, http.MethodPost, srv.URL+"/booking", token, map[string]any{
			"room_id":    roomID,
			"start_time": start.Format(time.RFC3339),
			"end_time":   end.Format(time.RFC3339),
		})
	}

	resp, body := create(studentB, 1, base, base.Add(time.Hour))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first create status = %d; body=%s", resp.StatusCode, body)
	}
	var booking bookingserver.Booking
	if err := json.Unmarshal(body, &booking); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	bookingID := *booking.Id

	// Overlap → 409
	resp, body = create(studentB, 1, base.Add(30*time.Minute), base.Add(90*time.Minute))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("overlap status = %d; body=%s", resp.StatusCode, body)
	}

	// Boundary-touch → 200
	resp, body = create(studentB, 1, base.Add(time.Hour), base.Add(2*time.Hour))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("boundary status = %d; body=%s", resp.StatusCode, body)
	}

	// Invalid range → 400
	resp, body = create(studentB, 1, base.Add(3*time.Hour), base.Add(2*time.Hour))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid range status = %d; body=%s", resp.StatusCode, body)
	}

	// Missing room → 404
	resp, body = create(studentB, 999, base, base.Add(time.Hour))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing room status = %d; body=%s", resp.StatusCode, body)
	}

	// Cancel as owner → 200, second cancel → 404
	resp, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("%s/booking/%d", srv.URL, bookingID), studentB, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first delete status = %d", resp.StatusCode)
	}
	resp, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("%s/booking/%d", srv.URL, bookingID), studentB, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete status = %d", resp.StatusCode)
	}
}

func TestE2E_RankBasedCancel(t *testing.T) {
	srv := bootServer(t)

	studentB := mintToken(t, 10, "alice", domain.RoleStudentB)
	studentM := mintToken(t, 11, "bob", domain.RoleStudentM)
	teacher := mintToken(t, 20, "prof", domain.RoleTeacher)
	admin := mintToken(t, 99, "root", domain.RoleAdmin)

	type slot struct {
		start time.Time
		end   time.Time
	}

	mkSlot := func(hour int) slot {
		start := time.Date(2026, time.September, 5, hour, 0, 0, 0, time.UTC)
		return slot{start, start.Add(time.Hour)}
	}

	createAs := func(token string, room int, s slot) int {
		t.Helper()
		resp, body := doJSON(t, http.MethodPost, srv.URL+"/booking", token, map[string]any{
			"room_id":    room,
			"start_time": s.start.Format(time.RFC3339),
			"end_time":   s.end.Format(time.RFC3339),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create as %s: status %d, body=%s", token[:10], resp.StatusCode, body)
		}
		var b bookingserver.Booking
		if err := json.Unmarshal(body, &b); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return *b.Id
	}

	deleteAs := func(token string, id int) int {
		resp, _ := doJSON(t, http.MethodDelete, fmt.Sprintf("%s/booking/%d", srv.URL, id), token, nil)
		return resp.StatusCode
	}

	t.Run("equal-rank different-user cannot cancel", func(t *testing.T) {
		id := createAs(studentB, 1, mkSlot(8))
		studentB2 := mintToken(t, 12, "carol", domain.RoleStudentB)
		if status := deleteAs(studentB2, id); status != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", status)
		}
	})

	t.Run("higher rank can cancel lower", func(t *testing.T) {
		id := createAs(studentB, 1, mkSlot(9))
		if status := deleteAs(studentM, id); status != http.StatusOK {
			t.Fatalf("studentM cancelling studentB: status = %d, want 200", status)
		}
	})

	t.Run("lower rank cannot cancel higher", func(t *testing.T) {
		id := createAs(teacher, 2, mkSlot(10))
		if status := deleteAs(studentM, id); status != http.StatusForbidden {
			t.Fatalf("studentM cancelling teacher: status = %d, want 403", status)
		}
	})

	t.Run("admin overrides any role", func(t *testing.T) {
		id := createAs(teacher, 3, mkSlot(11))
		if status := deleteAs(admin, id); status != http.StatusOK {
			t.Fatalf("admin cancelling teacher: status = %d, want 200", status)
		}
	})
}

func TestE2E_RaceThroughHTTP(t *testing.T) {
	srv := bootServer(t)

	token := mintToken(t, 10, "racer", domain.RoleStudentB)
	start := time.Date(2026, time.December, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	payload := map[string]any{
		"room_id":    2,
		"start_time": start.Format(time.RFC3339),
		"end_time":   end.Format(time.RFC3339),
	}
	raw, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 50}}

	const attempts = 10
	var (
		wg        sync.WaitGroup
		got200    int64
		got409    int64
		gotOther  int64
		mu        sync.Mutex
		otherMsgs []string
	)

	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/booking", bytes.NewReader(raw))
			if err != nil {
				t.Errorf("request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil {
				t.Errorf("do: %v", err)
				return
			}
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				atomic.AddInt64(&got200, 1)
			case http.StatusConflict:
				atomic.AddInt64(&got409, 1)
			default:
				atomic.AddInt64(&gotOther, 1)
				mu.Lock()
				otherMsgs = append(otherMsgs, fmt.Sprintf("%d %s", resp.StatusCode, strings.TrimSpace(string(body))))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if got200 != 1 {
		t.Fatalf("got200 = %d, want 1", got200)
	}
	if got409 != attempts-1 {
		t.Fatalf("got409 = %d, want %d", got409, attempts-1)
	}
	if gotOther > 0 {
		t.Fatalf("got %d unexpected statuses: %v", gotOther, otherMsgs)
	}
}
