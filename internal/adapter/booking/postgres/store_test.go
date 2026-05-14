package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/core/booking/domain"
)

func TestTranslateInsertError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   error
		wantErr error
	}{
		{
			name: "exclusion-violation on overlap constraint → ErrScheduleConflict",
			input: &pgconn.PgError{
				Code:           pgCodeExclusionViol,
				ConstraintName: constraintNoOverlap,
			},
			wantErr: domain.ErrScheduleConflict,
		},
		{
			name: "check-violation on time-range constraint → ErrInvalidTimeRange",
			input: &pgconn.PgError{
				Code:           pgCodeCheckViolation,
				ConstraintName: constraintTimeRange,
			},
			wantErr: domain.ErrInvalidTimeRange,
		},
		{
			name: "exclusion-violation on unknown constraint stays generic",
			input: &pgconn.PgError{
				Code:           pgCodeExclusionViol,
				ConstraintName: "other_excl",
			},
		},
		{
			name: "check-violation on unknown constraint stays generic",
			input: &pgconn.PgError{
				Code:           pgCodeCheckViolation,
				ConstraintName: "other_chk",
			},
		},
		{
			name:  "non-pg error stays generic",
			input: errors.New("connection refused"),
		},
		{
			name:  "wrapped pg error is still translated",
			input: fmt.Errorf("layer: %w", &pgconn.PgError{Code: pgCodeExclusionViol, ConstraintName: constraintNoOverlap}),
			wantErr: domain.ErrScheduleConflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := translateInsertError(tc.input)
			if tc.wantErr != nil {
				if !errors.Is(got, tc.wantErr) {
					t.Fatalf("translateInsertError = %v, want %v", got, tc.wantErr)
				}
				return
			}
			if errors.Is(got, domain.ErrScheduleConflict) || errors.Is(got, domain.ErrInvalidTimeRange) {
				t.Fatalf("translateInsertError mistranslated %v as domain error: %v", tc.input, got)
			}
		})
	}
}
