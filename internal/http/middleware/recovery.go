package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				requestId, _ := GetRequestID(r.Context())
				slog.ErrorContext(
					r.Context(), "http handler panicked",
					"request_id", requestId,
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)

				if rw, ok := w.(*responseWriter); ok && rw.Committed() {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal server error"}`))
			}

		}()

		next.ServeHTTP(w, r)
	})
}
