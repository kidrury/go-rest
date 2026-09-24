package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ContextKey string

const contextKey ContextKey = "request_id"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()

		ctx := context.WithValue(r.Context(), contextKey, requestID)

		r = r.WithContext(ctx)

		w.Header().Set("X-Request-Id", requestID)

		next.ServeHTTP(w, r)
	})
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(contextKey).(string)
	return requestID, ok
}
