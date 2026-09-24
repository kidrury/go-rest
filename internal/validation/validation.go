package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/kidrury/rest-pro/internal/domain"
)

var validate = newValidator()

// initialize the validator and have it use json names for validation
func newValidator() *validator.Validate {
	v := validator.New(
		validator.WithRequiredStructEnabled(),
	)

	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}

		return name
	})

	return v
}

// the function used for the actual validation
// the validation rules should be within the struct type of the val as their tags
func Struct(val any) error {
	err := validate.Struct(val)

	//
	if err != nil {
		fmt.Println("we have an error!", err)
		var validationErrors validator.ValidationErrors

		if errors.As(err, &validationErrors) {
			return domain.Error{
				Code:    "VALIDATION_FAILED",
				Message: "validation failed",
				Fields:  fields(validationErrors),
			}
		}

		return err
	}
	return nil
}

func message(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "email":
		return "is not a valid email"
	case "min":
		return fmt.Sprintf("is less than minimum of %s characters", err.Param())
	case "max":
		return fmt.Sprintf("is more than maximum of %s characters", err.Param())
	case "len":
		return fmt.Sprintf("length must be exactly %s characters", err.Param())
	case "eqfield":
		return fmt.Sprintf("value must be equal with field: %s", err.Param())
	default:
		return "is invalid"
	}
}

func fields(errs validator.ValidationErrors) map[string]string {
	output := make(map[string]string, len(errs))

	for _, err := range errs {
		output[err.Field()] = message(err)
	}

	return output
}
