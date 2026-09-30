package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const scheduleImportLockID int64 = 82471901

// Each attempt uses a transaction-scoped lock, released even if the process dies.
// Waiting replicas release their connection so they do not exhaust the pool.
func (s *Store) WithScheduleImportLock(ctx context.Context, run func() error) error {
	for {
		acquired := false
		err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
			if err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", scheduleImportLockID).Scan(&acquired); err != nil {
				return fmt.Errorf("lock schedule import: %w", err)
			}
			if !acquired {
				return nil
			}
			return run()
		})
		if err != nil || acquired {
			return err
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
