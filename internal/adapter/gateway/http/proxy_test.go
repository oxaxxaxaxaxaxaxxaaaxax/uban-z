package gatewayhttp_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gatewayhttp "github.com/oxaxxaxaxaxaxaxxaaaxax/uban-z/internal/adapter/gateway/http"
)

func TestGatewayProxiesAuthRequests(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotAuth string
	authService := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"abc"}`))
	})

	gateway := newTestGateway(t, authService, http.NotFoundHandler(), nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"login":"john","password":"secret"}`))
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotPath != "/api/auth/login" {
		t.Fatalf("auth path = %q, want /api/auth/login", gotPath)
	}
	if gotAuth != "Bearer token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuth)
	}
}

func TestGatewayProxiesAuthMe(t *testing.T) {
	t.Parallel()

	var gotPath string
	authService := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"login":"john","role":"student_b","full_name":"John Smith"}`))
	})

	gateway := newTestGateway(t, authService, http.NotFoundHandler(), nil)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotPath != "/api/auth/me" {
		t.Fatalf("auth path = %q, want /api/auth/me", gotPath)
	}
}

func TestGatewayProxiesBookingRequests(t *testing.T) {
	t.Parallel()

	var gotPath string
	bookingService := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"A-101"}]`))
	})

	gateway := newTestGateway(t, http.NotFoundHandler(), bookingService, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/1", nil)

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if gotPath != "/rooms/1" {
		t.Fatalf("booking path = %q, want /rooms/1", gotPath)
	}
}

func TestGatewayNormalizesPlainTextErrors(t *testing.T) {
	t.Parallel()

	bookingService := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "room not found", http.StatusNotFound)
	})

	gateway := newTestGateway(t, http.NotFoundHandler(), bookingService, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rooms/404", nil)

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v; body = %s", err, rec.Body.String())
	}
	if body["error"] != "room not found" || body["service"] != "booking" {
		t.Fatalf("body = %#v, want normalized booking error", body)
	}
}

func TestGatewayReturnsBadGatewayWhenUpstreamUnavailable(t *testing.T) {
	t.Parallel()

	gateway := newTestGateway(t, http.NotFoundHandler(), http.NotFoundHandler(), errors.New("dial failed"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)

	gateway.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v; body = %s", err, rec.Body.String())
	}
	if body["error"] != "upstream unavailable" || body["service"] != "auth" {
		t.Fatalf("body = %#v, want auth upstream unavailable", body)
	}
}

func TestGatewayDoesNotExposeRemovedUserRoutes(t *testing.T) {
	t.Parallel()

	gateway := newTestGateway(t, http.NotFoundHandler(), http.NotFoundHandler(), nil)

	for _, path := range []string{"/users/me", "/api/users/me"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)

			gateway.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}
}

func newTestGateway(t *testing.T, authHandler http.Handler, bookingHandler http.Handler, roundTripErr error) http.Handler {
	t.Helper()

	handler, err := gatewayhttp.NewHandler(gatewayhttp.Config{
		AuthServiceURL:    "http://auth.service",
		BookingServiceURL: "http://booking.service",
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if roundTripErr != nil {
				return nil, roundTripErr
			}

			rec := httptest.NewRecorder()
			switch req.URL.Host {
			case "auth.service":
				authHandler.ServeHTTP(rec, req)
			case "booking.service":
				bookingHandler.ServeHTTP(rec, req)
			default:
				http.NotFound(rec, req)
			}
			return rec.Result(), nil
		}),
	})
	if err != nil {
		t.Fatalf("new gateway: %v", err)
	}
	return handler
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
