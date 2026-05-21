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
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	bookingpostgres "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/postgres"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

var migrationsDir = filepath.Join("..", "..", "migrations", "booking")

// e2eServer is the booking-service HTTP wiring under test, plus the URL to hit it on.
type e2eServer struct {
	URL    string
	cancel context.CancelFunc
	pool   *pgxpool.Pool
}

// bootServer brings up a postgres container, applies migrations, wires the same
// stack as cmd/booking/main.go, and starts the HTTP listener on a random port.
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

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	store := bookingpostgres.NewStoreFromPool(pool)
	useCase := service.New(store, store)
	handler := bookinghttp.NewHandler(useCase, logger)

	router := httpx.Chain(
		bookingserver.Handler(handler),
		httpx.RequestID,
		httpx.RecoverPanic(logger),
		httpx.AccessLog(logger),
	)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &e2eServer{URL: srv.URL, cancel: func() {}, pool: pool}
}

func doJSON(t *testing.T, method, url string, body any) (*http.Response, []byte) {
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

func TestE2E_GetRooms(t *testing.T) {
	srv := bootServer(t)

	resp, body := doJSON(t, http.MethodGet, srv.URL+"/rooms", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
	}
	var rooms []bookingserver.Room
	if err := json.Unmarshal(body, &rooms); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(rooms) != 3 {
		t.Fatalf("got %d rooms, want 3 seeded", len(rooms))
	}
}

func TestE2E_BookingLifecycle(t *testing.T) {
	srv := bootServer(t)

	base := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
	create := func(roomID int, start, end time.Time) (*http.Response, []byte) {
		return doJSON(t, http.MethodPost, srv.URL+"/booking", map[string]any{
			"room_id":    roomID,
			"start_time": start.Format(time.RFC3339),
			"end_time":   end.Format(time.RFC3339),
		})
	}

	resp, body := create(1, base, base.Add(time.Hour))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first create status = %d; body=%s", resp.StatusCode, body)
	}
	var booking bookingserver.Booking
	if err := json.Unmarshal(body, &booking); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if booking.Id == nil || *booking.Id == 0 {
		t.Fatalf("missing id: %+v", booking)
	}
	bookingID := *booking.Id

	// Overlap → 409
	resp, body = create(1, base.Add(30*time.Minute), base.Add(90*time.Minute))
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("overlap status = %d; body=%s", resp.StatusCode, body)
	}

	// Boundary-touch → 200
	resp, body = create(1, base.Add(time.Hour), base.Add(2*time.Hour))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("boundary status = %d; body=%s", resp.StatusCode, body)
	}

	// Invalid range → 400
	resp, body = create(1, base.Add(3*time.Hour), base.Add(2*time.Hour))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid range status = %d; body=%s", resp.StatusCode, body)
	}

	// Missing room → 404
	resp, body = create(999, base, base.Add(time.Hour))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing room status = %d; body=%s", resp.StatusCode, body)
	}

	// Cancel → 200, second cancel → 404
	resp, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("%s/booking/%d", srv.URL, bookingID), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first delete status = %d", resp.StatusCode)
	}
	resp, _ = doJSON(t, http.MethodDelete, fmt.Sprintf("%s/booking/%d", srv.URL, bookingID), nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("second delete status = %d", resp.StatusCode)
	}
}

func TestE2E_RaceThroughHTTP(t *testing.T) {
	srv := bootServer(t)

	start := time.Date(2026, time.December, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	payload := map[string]any{
		"room_id":    2,
		"start_time": start.Format(time.RFC3339),
		"end_time":   end.Format(time.RFC3339),
	}
	raw, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 50}}

	const attempts = 25
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

// Compile-time guard so the linter doesn't drop the net import if unused elsewhere.
var _ = net.SplitHostPort
