package middleware

import (
	"net/http"

	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/domain"
	"github.com/kidrury/rest-pro/internal/http/response"
)

const accessCookieName = "__Host-access"

func Auth(tm *auth.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(accessCookieName)
			if err != nil {
				response.HandleError(w, r, domain.Error{Code: "UNAUTHORIZED", Message: "authentication required"})
				return
			}

			rawToken := cookie.Value

			claims, err := tm.VerifyAccessToken(rawToken)
			if err != nil {
				response.HandleError(w, r, domain.Error{Code: "UNAUTHORIZED", Message: "authentication required"})
				return
			}

			ctx := auth.WithIdentity(r.Context(), claims.Subject)

			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
