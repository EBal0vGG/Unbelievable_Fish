package httplog

import (
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
)

func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now()
			rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				if recovered := recover(); recovered != nil {
					if !rec.wroteHeader {
						http.Error(rec, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					}
					logger.ErrorContext(
						r.Context(),
						"http_request_panic",
						"component", "http",
						"method", r.Method,
						"path", r.URL.Path,
						"route", routePattern(r),
						"status", http.StatusInternalServerError,
						"duration_ms", time.Since(startedAt).Milliseconds(),
						"correlation_id", r.Header.Get("X-Correlation-ID"),
						"causation_id", r.Header.Get("X-Causation-ID"),
						"remote_addr", remoteAddr(r),
						"panic", recovered,
						"stack", string(debug.Stack()),
					)
					return
				}

				logRequest(logger, r, rec, time.Since(startedAt))
			}()

			next.ServeHTTP(rec, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += n
	return n, err
}

func logRequest(logger *slog.Logger, r *http.Request, rec *responseRecorder, duration time.Duration) {
	if r.URL.Path == "/health" {
		return
	}

	args := []any{
		"component", "http",
		"method", r.Method,
		"path", r.URL.Path,
		"route", routePattern(r),
		"status", rec.status,
		"bytes", rec.bytes,
		"duration_ms", duration.Milliseconds(),
		"correlation_id", r.Header.Get("X-Correlation-ID"),
		"causation_id", r.Header.Get("X-Causation-ID"),
		"remote_addr", remoteAddr(r),
		"user_agent", r.UserAgent(),
	}

	switch {
	case rec.status >= http.StatusInternalServerError:
		logger.ErrorContext(r.Context(), "http_request_completed", args...)
	case rec.status >= http.StatusBadRequest:
		logger.WarnContext(r.Context(), "http_request_completed", args...)
	default:
		logger.InfoContext(r.Context(), "http_request_completed", args...)
	}
}

func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		return rctx.RoutePattern()
	}
	return ""
}

func remoteAddr(r *http.Request) string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if value := r.Header.Get(header); value != "" {
			return value
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
