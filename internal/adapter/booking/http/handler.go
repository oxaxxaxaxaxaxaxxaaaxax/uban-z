package bookinghttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

// Handler implements the generated booking HTTP server interface.
type Handler struct {
	useCase service.UseCase
	logger  *slog.Logger
}

func NewHandler(useCase service.UseCase, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{useCase: useCase, logger: logger}
}

func (h *Handler) GetRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.useCase.ListRooms(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapRooms(rooms))
}

func (h *Handler) GetRoomsId(w http.ResponseWriter, r *http.Request, id int) {
	schedule, err := h.useCase.GetRoomSchedule(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapSchedule(schedule))
}

func (h *Handler) PostBooking(w http.ResponseWriter, r *http.Request) {
	identity, ok := httpx.RequireIdentity(w, r)
	if !ok {
		return
	}

	var request bookingserver.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, errInvalidRequest)
		return
	}

	booking, err := h.useCase.CreateBooking(r.Context(), service.CreateBookingInput{
		Caller: service.Caller{
			UserID: identity.UserID,
			Login:  identity.Login,
			Role:   identity.Role,
		},
		RoomID:    request.RoomId,
		StartTime: request.StartTime,
		EndTime:   request.EndTime,
	})
	if err != nil {
		if errors.Is(err, domain.ErrScheduleConflict) {
			h.logger.WarnContext(r.Context(), "booking.conflict",
				slog.String("event", "booking.conflict"),
				slog.String("action", "Booking rejected"),
				slog.String("actor", identity.Login),
				slog.String("details", "User "+identity.Login+" tried to book room with an occupied time slot"),
				slog.String("request_id", httpx.RequestIDFrom(r.Context())),
				slog.Int("room_id", request.RoomId),
				slog.Int("user_id", identity.UserID),
				slog.String("login", identity.Login),
				slog.String("role", string(identity.Role)),
				slog.Time("start_time", request.StartTime),
				slog.Time("end_time", request.EndTime),
			)
		}
		writeError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "booking.created",
		slog.String("event", "booking.created"),
		slog.String("action", "Booking created"),
		slog.String("actor", identity.Login),
		slog.String("details", "User "+identity.Login+" booked a room"),
		slog.String("request_id", httpx.RequestIDFrom(r.Context())),
		slog.Int("booking_id", booking.ID),
		slog.Int("room_id", booking.RoomID),
		slog.Int("user_id", booking.UserID),
		slog.String("login", identity.Login),
		slog.String("creator_role", string(booking.CreatorRole)),
		slog.Time("start_time", booking.StartTime),
		slog.Time("end_time", booking.EndTime),
	)
	writeJSON(w, http.StatusOK, mapBooking(booking))
}

func (h *Handler) GetBookingMy(w http.ResponseWriter, r *http.Request) {
	identity, ok := httpx.RequireIdentity(w, r)
	if !ok {
		return
	}

	bookings, err := h.useCase.ListMyBookings(r.Context(), service.Caller{
		UserID: identity.UserID,
		Login:  identity.Login,
		Role:   identity.Role,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapBookings(bookings))
}

func (h *Handler) DeleteBookingId(w http.ResponseWriter, r *http.Request, id int) {
	identity, ok := httpx.RequireIdentity(w, r)
	if !ok {
		return
	}

	err := h.useCase.CancelBooking(r.Context(), id, service.Caller{
		UserID: identity.UserID,
		Login:  identity.Login,
		Role:   identity.Role,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	h.logger.InfoContext(r.Context(), "booking.cancelled",
		slog.String("event", "booking.cancelled"),
		slog.String("action", "Booking cancelled"),
		slog.String("actor", identity.Login),
		slog.String("details", "User "+identity.Login+" cancelled booking"),
		slog.String("request_id", httpx.RequestIDFrom(r.Context())),
		slog.Int("booking_id", id),
		slog.Int("cancelled_by_user_id", identity.UserID),
		slog.String("cancelled_by_login", identity.Login),
		slog.String("cancelled_by_role", string(identity.Role)),
	)
	w.WriteHeader(http.StatusOK)
}

var _ bookingserver.ServerInterface = (*Handler)(nil)
