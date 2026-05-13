package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

func TestValidateTimeRange(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.April, 17, 9, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		start   time.Time
		end     time.Time
		wantErr error
	}{
		{
			name:    "start before end is valid",
			start:   base,
			end:     base.Add(time.Hour),
			wantErr: nil,
		},
		{
			name:    "one nanosecond gap is valid",
			start:   base,
			end:     base.Add(time.Nanosecond),
			wantErr: nil,
		},
		{
			name:    "start equal to end is rejected",
			start:   base,
			end:     base,
			wantErr: domain.ErrInvalidTimeRange,
		},
		{
			name:    "start after end is rejected",
			start:   base.Add(time.Hour),
			end:     base,
			wantErr: domain.ErrInvalidTimeRange,
		},
		{
			name:    "two zero times are rejected",
			start:   time.Time{},
			end:     time.Time{},
			wantErr: domain.ErrInvalidTimeRange,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := domain.ValidateTimeRange(tc.start, tc.end)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ValidateTimeRange(%v, %v) = %v, want %v", tc.start, tc.end, err, tc.wantErr)
			}
		})
	}
}
