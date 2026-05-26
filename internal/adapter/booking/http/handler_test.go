package bookinghttp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

var testJWTSecret = []byte("handler-test-secret")

func mintToken(t *testing.T, userID int, login string, role domain.Role) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   itoa(userID),
		"login": login,
		"role":  string(role),
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(testJWTSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func itoa(i int) string {
	return strings.TrimSpace(strings.Replace(jsonNumber(i), "\"", "", -1))
}

// jsonNumber returns an int as the same string format json/Marshal would emit.
func jsonNumber(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

type stubUseCase struct {
	listRoomsFn       func(ctx context.Context) ([]domain.Room, error)
	getRoomScheduleFn func(ctx context.Context, roomID int) ([]domain.ScheduleItem, error)
	listMyBookingsFn  func(ctx context.Context, caller service.Caller) ([]domain.Booking, error)
	createBookingFn   func(ctx context.Context, input service.CreateBookingInput) (domain.Booking, error)
	cancelBookingFn   func(ctx context.Context, bookingID int, caller service.Caller) error
}

func (s stubUseCase) ListRooms(ctx context.Context) ([]domain.Room, error) {
	return s.listRoomsFn(ctx)
}
func (s stubUseCase) GetRoomSchedule(ctx context.Context, roomID int) ([]domain.ScheduleItem, error) {
	return s.getRoomScheduleFn(ctx, roomID)
}
func (s stubUseCase) ListMyBookings(ctx context.Context, caller service.Caller) ([]domain.Booking, error) {
	return s.listMyBookingsFn(ctx, caller)
}
func (s stubUseCase) CreateBooking(ctx context.Context, input service.CreateBookingInput) (domain.Booking, error) {
	return s.createBookingFn(ctx, input)
}
func (s stubUseCase) CancelBooking(ctx context.Context, bookingID int, caller service.Caller) error {
	return s.cancelBookingFn(ctx, bookingID, caller)
}

func newServer(t *testing.T, uc service.UseCase) *httptest.Server {
	t.Helper()
	router := httpx.Chain(
		bookingserver.Handler(bookinghttp.NewHandler(uc, nil)),
		httpx.ParseToken(testJWTSecret),
	)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url, token string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp, payload
}

func TestGetRooms(t *testing.T) {
	t.Parallel()

	t.Run("200 returns mapped rooms (anonymous OK)", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			listRoomsFn: func(ctx context.Context) ([]domain.Room, error) {
				return []domain.Room{{ID: 7, Name: "A101", Capacity: 12, Building: "North"}}, nil
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodGet, srv.URL+"/rooms", "", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
		}
		var rooms []bookingserver.Room
		if err := json.Unmarshal(body, &rooms); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(rooms) != 1 || rooms[0].Id == nil || *rooms[0].Id != 7 {
			t.Fatalf("rooms = %+v", rooms)
		}
	})

	t.Run("500 on unknown use-case error", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			listRoomsFn: func(ctx context.Context) ([]domain.Room, error) {
				return nil, errors.New("boom")
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodGet, srv.URL+"/rooms", "", nil)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500; body=%s", resp.StatusCode, body)
		}
		assertErrorBody(t, body, "boom")
	})
}

func TestGetRoomsId_404OnRoomNotFound(t *testing.T) {
	t.Parallel()

	uc := stubUseCase{
		getRoomScheduleFn: func(ctx context.Context, roomID int) ([]domain.ScheduleItem, error) {
			return nil, domain.ErrRoomNotFound
		},
	}
	srv := newServer(t, uc)

	resp, body := do(t, http.MethodGet, srv.URL+"/rooms/42", "", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
	}
	assertErrorBody(t, body, domain.ErrRoomNotFound.Error())
}

func TestPostBooking(t *testing.T) {
	t.Parallel()

	validBody := func() io.Reader {
		req := bookingserver.CreateBookingRequest{
			RoomId:    1,
			StartTime: time.Date(2026, time.April, 17, 9, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, time.April, 17, 10, 0, 0, 0, time.UTC),
		}
		b, _ := json.Marshal(req)
		return bytes.NewReader(b)
	}

	t.Run("401 without token", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				t.Fatal("should not be called")
				return domain.Booking{}, nil
			},
		}
		srv := newServer(t, uc)
		resp, _ := do(t, http.MethodPost, srv.URL+"/booking", "", validBody())
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("200 returns created booking", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, input service.CreateBookingInput) (domain.Booking, error) {
				if input.Caller.UserID != 5 || input.Caller.Role != domain.RoleStudentB {
					t.Errorf("caller = %+v", input.Caller)
				}
				return domain.Booking{
					ID:          99,
					RoomID:      input.RoomID,
					UserID:      input.Caller.UserID,
					CreatorRole: input.Caller.Role,
					StartTime:   input.StartTime,
					EndTime:     input.EndTime,
				}, nil
			},
		}
		srv := newServer(t, uc)

		token := mintToken(t, 5, "alice", domain.RoleStudentB)
		resp, body := do(t, http.MethodPost, srv.URL+"/booking", token, validBody())
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				t.Fatal("should not be called")
				return domain.Booking{}, nil
			},
		}
		srv := newServer(t, uc)

		token := mintToken(t, 1, "x", domain.RoleStudentB)
		resp, _ := do(t, http.MethodPost, srv.URL+"/booking", token, strings.NewReader("{not json"))
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
	})

	t.Run("400 on ErrInvalidTimeRange", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, domain.ErrInvalidTimeRange
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "x", domain.RoleStudentB)

		resp, body := do(t, http.MethodPost, srv.URL+"/booking", token, validBody())
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
		}
	})

	t.Run("404 on ErrRoomNotFound", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, domain.ErrRoomNotFound
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "x", domain.RoleStudentB)

		resp, _ := do(t, http.MethodPost, srv.URL+"/booking", token, validBody())
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("409 on ErrScheduleConflict", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, domain.ErrScheduleConflict
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "x", domain.RoleStudentB)

		resp, _ := do(t, http.MethodPost, srv.URL+"/booking", token, validBody())
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("500 on unknown error", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, errors.New("kaboom")
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "x", domain.RoleStudentB)

		resp, _ := do(t, http.MethodPost, srv.URL+"/booking", token, validBody())
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})
}

func TestGetBookingMy(t *testing.T) {
	t.Parallel()

	t.Run("401 without token", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			listMyBookingsFn: func(ctx context.Context, caller service.Caller) ([]domain.Booking, error) {
				t.Fatal("should not be called")
				return nil, nil
			},
		}
		srv := newServer(t, uc)
		resp, _ := do(t, http.MethodGet, srv.URL+"/booking/my", "", nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("200 returns caller's bookings", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			listMyBookingsFn: func(_ context.Context, caller service.Caller) ([]domain.Booking, error) {
				if caller.UserID != 7 || caller.Role != domain.RoleStudentB {
					t.Errorf("caller = %+v", caller)
				}
				return []domain.Booking{
					{ID: 1, RoomID: 2, UserID: 7, CreatorRole: domain.RoleStudentB,
						StartTime: time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC),
						EndTime:   time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)},
				}, nil
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 7, "alice", domain.RoleStudentB)

		resp, body := do(t, http.MethodGet, srv.URL+"/booking/my", token, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
		}
		var bookings []bookingserver.Booking
		if err := json.Unmarshal(body, &bookings); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(bookings) != 1 || bookings[0].Id == nil || *bookings[0].Id != 1 {
			t.Fatalf("bookings = %+v", bookings)
		}
	})

	t.Run("200 with empty array when caller has none", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			listMyBookingsFn: func(_ context.Context, _ service.Caller) ([]domain.Booking, error) {
				return nil, nil
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 7, "alice", domain.RoleStudentB)

		resp, body := do(t, http.MethodGet, srv.URL+"/booking/my", token, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, body=%s", resp.StatusCode, body)
		}
		if strings.TrimSpace(string(body)) != "[]" {
			t.Fatalf("body = %q, want []", body)
		}
	})
}

func TestDeleteBookingId(t *testing.T) {
	t.Parallel()

	t.Run("401 without token", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			cancelBookingFn: func(_ context.Context, _ int, _ service.Caller) error {
				t.Fatal("should not be called")
				return nil
			},
		}
		srv := newServer(t, uc)
		resp, _ := do(t, http.MethodDelete, srv.URL+"/booking/5", "", nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("200 on success", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			cancelBookingFn: func(_ context.Context, id int, caller service.Caller) error {
				if id != 5 || caller.UserID != 1 {
					t.Errorf("id=%d caller=%+v", id, caller)
				}
				return nil
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "alice", domain.RoleStudentB)

		resp, _ := do(t, http.MethodDelete, srv.URL+"/booking/5", token, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})

	t.Run("403 on ErrForbidden", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			cancelBookingFn: func(_ context.Context, _ int, _ service.Caller) error {
				return domain.ErrForbidden
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "alice", domain.RoleStudentB)

		resp, body := do(t, http.MethodDelete, srv.URL+"/booking/5", token, nil)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d; body=%s", resp.StatusCode, body)
		}
	})

	t.Run("404 on ErrBookingNotFound", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			cancelBookingFn: func(_ context.Context, _ int, _ service.Caller) error {
				return domain.ErrBookingNotFound
			},
		}
		srv := newServer(t, uc)
		token := mintToken(t, 1, "alice", domain.RoleStudentB)

		resp, _ := do(t, http.MethodDelete, srv.URL+"/booking/5", token, nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	})
}

func assertErrorBody(t *testing.T, payload []byte, wantMessage string) {
	t.Helper()
	var envelope struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("error body not JSON: %v; raw=%s", err, payload)
	}
	if envelope.Error != wantMessage {
		t.Fatalf("error = %q, want %q", envelope.Error, wantMessage)
	}
}
