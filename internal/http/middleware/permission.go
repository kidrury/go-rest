package middleware

import (
	"net/http"

	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/authz"
	"github.com/kidrury/rest-pro/internal/domain"
	"github.com/kidrury/rest-pro/internal/http/response"
	"github.com/kidrury/rest-pro/internal/service"
)

func RequirePermission(userService *service.UserService, permission authz.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := auth.IdentityFromContext(r.Context())
			if !ok {
				response.HandleError(w, r, domain.Error{
					Code:    "UNAUTHORIZED",
					Message: "authentication required",
				})
				return
			}

			err := userService.RequirePermission(r.Context(), actor, permission)

			if err != nil {
				response.HandleError(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
