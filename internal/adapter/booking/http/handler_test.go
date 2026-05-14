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

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	bookinghttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/http"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
)

type stubUseCase struct {
	listRoomsFn       func(ctx context.Context) ([]domain.Room, error)
	getRoomScheduleFn func(ctx context.Context, roomID int) ([]domain.ScheduleItem, error)
	createBookingFn   func(ctx context.Context, input service.CreateBookingInput) (domain.Booking, error)
	cancelBookingFn   func(ctx context.Context, bookingID int) error
}

func (s stubUseCase) ListRooms(ctx context.Context) ([]domain.Room, error) {
	return s.listRoomsFn(ctx)
}
func (s stubUseCase) GetRoomSchedule(ctx context.Context, roomID int) ([]domain.ScheduleItem, error) {
	return s.getRoomScheduleFn(ctx, roomID)
}
func (s stubUseCase) CreateBooking(ctx context.Context, input service.CreateBookingInput) (domain.Booking, error) {
	return s.createBookingFn(ctx, input)
}
func (s stubUseCase) CancelBooking(ctx context.Context, bookingID int) error {
	return s.cancelBookingFn(ctx, bookingID)
}

func newServer(t *testing.T, uc service.UseCase) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(bookingserver.Handler(bookinghttp.NewHandler(uc, nil)))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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

	t.Run("200 returns mapped rooms", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			listRoomsFn: func(ctx context.Context) ([]domain.Room, error) {
				return []domain.Room{{ID: 7, Name: "A101", Capacity: 12, Building: "North"}}, nil
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodGet, srv.URL+"/rooms", nil)
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

		resp, body := do(t, http.MethodGet, srv.URL+"/rooms", nil)
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

	resp, body := do(t, http.MethodGet, srv.URL+"/rooms/42", nil)
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

	t.Run("200 returns created booking", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, input service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{
					ID:        99,
					RoomID:    input.RoomID,
					StartTime: input.StartTime,
					EndTime:   input.EndTime,
				}, nil
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodPost, srv.URL+"/booking", validBody())
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
		}
		var booking bookingserver.Booking
		if err := json.Unmarshal(body, &booking); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if booking.Id == nil || *booking.Id != 99 {
			t.Fatalf("id = %+v", booking.Id)
		}
		if booking.StartTime == nil || !strings.HasPrefix(*booking.StartTime, "2026-04-17T09:00:00") {
			t.Fatalf("start_time = %+v", booking.StartTime)
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

		resp, _ := do(t, http.MethodPost, srv.URL+"/booking", strings.NewReader("{not json"))
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

		resp, body := do(t, http.MethodPost, srv.URL+"/booking", validBody())
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
		}
		assertErrorBody(t, body, domain.ErrInvalidTimeRange.Error())
	})

	t.Run("404 on ErrRoomNotFound", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, domain.ErrRoomNotFound
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodPost, srv.URL+"/booking", validBody())
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
		}
		assertErrorBody(t, body, domain.ErrRoomNotFound.Error())
	})

	t.Run("409 on ErrScheduleConflict", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, domain.ErrScheduleConflict
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodPost, srv.URL+"/booking", validBody())
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d, want 409; body=%s", resp.StatusCode, body)
		}
		assertErrorBody(t, body, domain.ErrScheduleConflict.Error())
	})

	t.Run("500 on unknown error", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			createBookingFn: func(ctx context.Context, _ service.CreateBookingInput) (domain.Booking, error) {
				return domain.Booking{}, errors.New("kaboom")
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodPost, srv.URL+"/booking", validBody())
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500; body=%s", resp.StatusCode, body)
		}
	})
}

func TestDeleteBookingId(t *testing.T) {
	t.Parallel()

	t.Run("200 on success", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			cancelBookingFn: func(ctx context.Context, id int) error {
				if id != 5 {
					t.Fatalf("id = %d, want 5", id)
				}
				return nil
			},
		}
		srv := newServer(t, uc)

		resp, _ := do(t, http.MethodDelete, srv.URL+"/booking/5", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("404 on ErrBookingNotFound", func(t *testing.T) {
		t.Parallel()
		uc := stubUseCase{
			cancelBookingFn: func(ctx context.Context, _ int) error {
				return domain.ErrBookingNotFound
			},
		}
		srv := newServer(t, uc)

		resp, body := do(t, http.MethodDelete, srv.URL+"/booking/5", nil)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
		}
		assertErrorBody(t, body, domain.ErrBookingNotFound.Error())
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
