package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
		}

		requestId, _ := GetRequestID(r.Context())
		slog.InfoContext(
			r.Context(), "request received",
			"request_id", requestId,
			"method", r.Method,
			"path", r.URL.Path,
		)

		next.ServeHTTP(rw, r)

		durationMs := time.Since(now)

		statusCode := rw.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		logAttr := []any{
			"request_id", requestId,
			"method", r.Method,
			"path", r.URL.Path,
			"status", statusCode,
			"duration_ms", durationMs,
			"bytes", rw.bytesWritten,
			"remote_addr", r.RemoteAddr,
		}

		if statusCode >= 500 {
			slog.ErrorContext(r.Context(), "http request completed with server error", logAttr...)
		} else {
			slog.InfoContext(r.Context(), "http request completed with success", logAttr...)
		}
	})
}
