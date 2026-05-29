package gatewayhttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const defaultUpstreamTimeout = 30 * time.Second

type Config struct {
	AuthServiceURL    string
	BookingServiceURL string
	Timeout           time.Duration
	Transport         http.RoundTripper
}

type Handler struct {
	mux *http.ServeMux
}

func NewHandler(cfg Config) (*Handler, error) {
	authURL, err := parseUpstreamURL(cfg.AuthServiceURL, "auth service")
	if err != nil {
		return nil, err
	}
	bookingURL, err := parseUpstreamURL(cfg.BookingServiceURL, "booking service")
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultUpstreamTimeout
	}
	transport := cfg.Transport
	if transport == nil {
		defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
		defaultTransport.ResponseHeaderTimeout = timeout
		transport = defaultTransport
	}

	h := &Handler{mux: http.NewServeMux()}
	authProxy := newProxy("auth", authURL, transport, ensureAPIPrefix)
	bookingBareProxy := newProxy("booking", bookingURL, transport, keepPath)

	h.handleAuth(authProxy)
	h.handleBooking(bookingBareProxy)
	h.mux.HandleFunc("/", notFound)

	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleAuth(proxy http.Handler) {
	h.mux.Handle("/auth/", proxy)
}

func (h *Handler) handleBooking(proxy http.Handler) {
	h.mux.Handle("/rooms", proxy)
	h.mux.Handle("/rooms/", proxy)
	h.mux.Handle("/booking", proxy)
	h.mux.Handle("/booking/", proxy)
}

type pathRewrite func(string) string

func newProxy(service string, target *url.URL, transport http.RoundTripper, rewrite pathRewrite) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, rewrite(req.URL.Path))
		req.URL.RawPath = ""
		req.Host = target.Host
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		return normalizeErrorResponse(service, resp)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeGatewayError(w, http.StatusBadGateway, service, "upstream unavailable")
	}

	return proxy
}

func parseUpstreamURL(rawURL string, name string) (*url.URL, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("%s url is required", name)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s url: %w", name, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%s url must include scheme and host", name)
	}

	return parsed, nil
}

func keepPath(path string) string {
	return path
}

func ensureAPIPrefix(path string) string {
	if path == "/api" || strings.HasPrefix(path, "/api/") {
		return path
	}
	return "/api" + path
}

func normalizeErrorResponse(service string, resp *http.Response) error {
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		replaceBody(resp, gatewayError{Error: http.StatusText(resp.StatusCode), Service: service})
		return nil
	}

	if isJSONResponse(resp.Header.Get("Content-Type")) || json.Valid(trimmed) {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
		return nil
	}

	replaceBody(resp, gatewayError{Error: string(trimmed), Service: service})
	return nil
}

func replaceBody(resp *http.Response, value gatewayError) {
	body, err := json.Marshal(value)
	if err != nil {
		err = errors.New("failed to marshal gateway error")
		body = []byte(`{"error":"` + err.Error() + `"}`)
	}
	body = append(body, '\n')

	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
}

func isJSONResponse(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "application/json")
}

func singleJoiningSlash(basePath string, requestPath string) string {
	switch {
	case basePath == "":
		return requestPath
	case requestPath == "":
		return basePath
	case strings.HasSuffix(basePath, "/") && strings.HasPrefix(requestPath, "/"):
		return basePath + requestPath[1:]
	case !strings.HasSuffix(basePath, "/") && !strings.HasPrefix(requestPath, "/"):
		return basePath + "/" + requestPath
	default:
		return basePath + requestPath
	}
}

func notFound(w http.ResponseWriter, r *http.Request) {
	writeGatewayError(w, http.StatusNotFound, "gateway", "route not found")
}

func writeGatewayError(w http.ResponseWriter, status int, service string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(gatewayError{
		Error:   message,
		Service: service,
	})
}

type gatewayError struct {
	Error   string `json:"error"`
	Service string `json:"service"`
}
