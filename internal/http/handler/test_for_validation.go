package handler

import (
	"fmt"
	"net/http"

	"github.com/kidrury/rest-pro/internal/validation"
)

type Address struct {
	City       string `json:"city" validate:"required,min=2,max=20"`
	PostalCode string `json:"postalCode" validate:"required,len=5"`
}

type Item struct {
	ProductID string `json:"productId" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

type ValidationExerciseRequest struct {
	Password        string  `json:"password" validate:"required,min=8,max=30"`
	ConfirmPassword string  `json:"confirmPassword" validate:"required,eqfield=Password"`
	Address         Address `json:"address" validate:"required"`
	Items           []Item  `json:"items" validate:"required,min=1,dive"`
}

func TestValidation(w http.ResponseWriter, r *http.Request) error {
	var decoded ValidationExerciseRequest
	if err := DecodeJSON(w, r, &decoded); err != nil {
		return err
	}

	fmt.Println("going for validation now...")
	if err := validation.Struct(decoded); err != nil {
		return err
	}

	w.WriteHeader(http.StatusCreated)

	return nil
}
