package httpx

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServerShutdownWaitsForActiveRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = w.Write([]byte("finished"))
	})}
	shutdownStarted := make(chan struct{})
	server.RegisterOnShutdown(func() { close(shutdownStarted) })
	url, done := startTestServer(t, ctx, server, 2*time.Second)
	response := make(chan error, 1)
	go func() {
		resp, err := http.Get(url)
		if err == nil {
			defer resp.Body.Close()
			var body []byte
			body, err = io.ReadAll(resp.Body)
			if err == nil && string(body) != "finished" {
				err = errors.New("incomplete response")
			}
		}
		response <- err
	}()
	awaitSignal(t, started)
	cancel()
	awaitSignal(t, shutdownStarted)
	select {
	case err := <-done:
		t.Fatalf("server exited before active request finished: %v", err)
	default:
	}
	if conn, err := net.DialTimeout("tcp", server.Addr, time.Second); err == nil {
		_ = conn.Close()
		t.Fatal("server still accepts connections during shutdown")
	}
	release <- struct{}{}
	if err := awaitResult(t, response); err != nil {
		t.Fatalf("active request: %v", err)
	}
	if err := awaitResult(t, done); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestServerShutdownTimeoutClosesActiveConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	handlerDone := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(handlerDone)
	})}
	url, done := startTestServer(t, ctx, server, 50*time.Millisecond)
	response := make(chan error, 1)
	go func() {
		resp, err := http.Get(url)
		if resp != nil {
			_ = resp.Body.Close()
		}
		response <- err
	}()
	awaitSignal(t, started)
	cancel()
	if err := awaitResult(t, done); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error = %v, want deadline exceeded", err)
	}
	awaitSignal(t, handlerDone)
	if err := awaitResult(t, response); err == nil {
		t.Fatal("request should fail after forced connection close")
	}
}

func TestServerReturnsListenError(t *testing.T) {
	want := errors.New("listen failed")
	err := serveUntilCanceled(context.Background(), &http.Server{}, time.Second, func() error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func startTestServer(t *testing.T, ctx context.Context, server *http.Server, timeout time.Duration) (string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.Addr = listener.Addr().String()
	t.Cleanup(func() { _ = server.Close() })
	done := make(chan error, 1)
	go func() {
		done <- serveUntilCanceled(ctx, server, timeout, func() error { return server.Serve(listener) })
	}()
	return "http://" + server.Addr, done
}

func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for signal")
	}
}

func awaitResult(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
		return nil
	}
}
