package response

import (
	"net/http"
	"time"
)

const AcessCookieName = "__Host-access"
const RefreshCookieName = "__Secure-refresh"

func SetAccessCookie(w http.ResponseWriter, accessToken string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     AcessCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(maxAge.Seconds()),
	})
}

func ClearAcessCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AcessCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}

func SetRefreshCookie(w http.ResponseWriter, refreshToken string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    refreshToken,
		Path:     "/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(maxAge.Seconds()),
	})
}

func ClearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     "/auth",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
