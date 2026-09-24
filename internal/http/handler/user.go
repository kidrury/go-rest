package handler

import (
	"encoding/json"
	"net/http"

	"github.com/kidrury/rest-pro/internal/auth"
	"github.com/kidrury/rest-pro/internal/domain"
	"github.com/kidrury/rest-pro/internal/service"
	"github.com/kidrury/rest-pro/internal/validation"
)

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=3,max=10"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type ListUsersQuery struct {
	Limit  int64 `validate:"gte=1,lte=100"`
	Offset int64 `validate:"gte=0"`
}

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func CreateUser(w http.ResponseWriter, r *http.Request) error {
	var decoded CreateUserRequest
	if err := DecodeJSON(w, r, &decoded); err != nil {
		return err
	}

	if err := validation.Struct(decoded); err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)
	return nil
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	targetUserID, err := UUIDPathValue(r, "id")
	if err != nil {
		return err
	}

	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		return domain.Error{
			Code:    "UNAUTHORIZED",
			Message: "authentication required",
		}
	}

	user, err := h.userService.GetUserByID(r.Context(), identity, targetUserID.String())
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return json.NewEncoder(w).Encode(map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}

func ListUsers(w http.ResponseWriter, r *http.Request) error {
	//under construction :D
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return json.NewEncoder(w).Encode(map[string]any{"success": "true"})

}
