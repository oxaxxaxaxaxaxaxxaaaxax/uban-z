package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/platform/httpx"
)

func TestCORSAllowedOriginPreflight(t *testing.T) {
	t.Parallel()

	handler := httpx.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("preflight must not call next handler")
		}),
		httpx.CORS([]string{"http://localhost:3000"}),
	)

	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("allow origin = %q, want http://localhost:3000", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allow credentials = %q, want true", got)
	}
}

func TestCORSDisallowedOriginStillCallsNext(t *testing.T) {
	t.Parallel()

	handler := httpx.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		}),
		httpx.CORS([]string{"http://localhost:3000"}),
	)

	req := httptest.NewRequest(http.MethodGet, "/rooms", nil)
	req.Header.Set("Origin", "http://example.test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow origin = %q, want empty", got)
	}
}
