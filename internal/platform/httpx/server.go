package httpx

import (
	"context"
	"errors"
	"net/http"
	"time"
)

func ListenAndServe(ctx context.Context, server *http.Server, timeout time.Duration) error {
	return serveUntilCanceled(ctx, server, timeout, server.ListenAndServe)
}

func serveUntilCanceled(ctx context.Context, server *http.Server, timeout time.Duration, serve func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- serve() }()

	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := server.Shutdown(shutdownCtx)
	if err != nil {
		_ = server.Close()
	}
	<-done
	return err
}
