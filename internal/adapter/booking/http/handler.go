package bookinghttp

import (
	"encoding/json"
	"net/http"

	bookingserver "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/booking/bookingserver"
	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/service"
)

// Handler implements the generated booking HTTP server interface.
type Handler struct {
	useCase service.UseCase
}

func NewHandler(useCase service.UseCase) *Handler {
	return &Handler{useCase: useCase}
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
	var request bookingserver.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, errInvalidRequest)
		return
	}

	booking, err := h.useCase.CreateBooking(r.Context(), service.CreateBookingInput{
		RoomID:    request.RoomId,
		StartTime: request.StartTime,
		EndTime:   request.EndTime,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, mapBooking(booking))
}

func (h *Handler) DeleteBookingId(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.useCase.CancelBooking(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

var _ bookingserver.ServerInterface = (*Handler)(nil)
