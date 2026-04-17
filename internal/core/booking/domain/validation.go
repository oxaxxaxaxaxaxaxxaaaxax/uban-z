package domain

import "time"

func ValidateTimeRange(startTime, endTime time.Time) error {
	if !startTime.Before(endTime) {
		return ErrInvalidTimeRange
	}

	return nil
}
