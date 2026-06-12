package httplog

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMiddlewareSkipsHealthCheckLogging(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))

	router := chi.NewRouter()
	router.Use(Middleware(logger))
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no log for /health, got %s", out.String())
	}
}

func TestMiddlewareLogsCompletedRequest(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))

	router := chi.NewRouter()
	router.Use(Middleware(logger))
	router.Get("/items", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	logLine := out.String()
	for _, want := range []string{`"msg":"http_request_completed"`, `"status":204`, `"route":"/items"`} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("expected log to contain %s, got %s", want, logLine)
		}
	}
}

func TestMiddlewareLogsPanic(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&out, nil))

	router := chi.NewRouter()
	router.Use(Middleware(logger))
	router.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	logLine := out.String()
	for _, want := range []string{`"msg":"http_request_panic"`, `"status":500`, `"route":"/panic"`} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("expected log to contain %s, got %s", want, logLine)
		}
	}
}
