package handler

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"time"

	"github.com/kidrury/rest-pro/internal/domain"
	"github.com/kidrury/rest-pro/internal/http/response"
	"github.com/kidrury/rest-pro/internal/service"
	"github.com/kidrury/rest-pro/internal/validation"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=7"`
}

type AuthHandler struct {
	authService *service.AuthService
	jwtTTL      time.Duration
}

func NewAuthHandler(s *service.AuthService, ttl time.Duration) *AuthHandler {
	return &AuthHandler{
		authService: s,
		jwtTTL:      ttl,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var loginRequest LoginRequest
	err := DecodeJSON(w, r, &loginRequest)
	if err != nil {
		return err
	}

	err = validation.Struct(loginRequest)
	if err != nil {
		return err
	}

	result, err := h.authService.Login(r.Context(), loginRequest.Email, loginRequest.Password)
	if err != nil {
		return err
	}

	response.SetAccessCookie(w, result.AccessToken, h.jwtTTL)
	response.SetRefreshCookie(w, result.RefreshToken, time.Until(result.RefreshUntil))

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(response.RefreshCookieName)
	if err != nil {
		return domain.Error{
			Code:    "UNAUTHORIZED",
			Message: "authentication required",
		}
	}

	fmt.Printf("REFRESH cookie value: %s\n", cookie.Value)

	cookieHash := sha256.Sum256([]byte(cookie.Value))
	fmt.Printf("REFRESH cookie hash: %x\n", cookieHash)

	result, err := h.authService.Refresh(r.Context(), cookie.Value)
	if err != nil {
		return err
	}

	response.SetAccessCookie(w, result.AccessToken, h.jwtTTL)
	response.SetRefreshCookie(w, result.RefreshToken, time.Until(result.RefreshUntil))

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(response.RefreshCookieName)

	//IDEMPOTENT
	if err == nil {
		err = h.authService.Logout(r.Context(), cookie.Value)
		if err != nil {
			return err
		}

	}

	response.ClearAcessCookie(w)
	response.ClearRefreshCookie(w)

	w.WriteHeader(http.StatusNoContent)

	return nil
}
