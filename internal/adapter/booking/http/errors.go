package bookinghttp

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

var errInvalidRequest = errors.New("invalid request")

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, errInvalidRequest), errors.Is(err, domain.ErrInvalidTimeRange):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrRoomNotFound), errors.Is(err, domain.ErrBookingNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrScheduleConflict):
		status = http.StatusConflict
	}

	writeJSON(w, status, errorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
