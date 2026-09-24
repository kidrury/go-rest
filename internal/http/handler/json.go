package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kidrury/rest-pro/internal/domain"
)

const maxJSONSize = 1 << 20

func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONSize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(destination)

	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return domain.Error{
				Code:    "INVALID_REQUEST",
				Message: "request body is required",
			}
		case errors.As(err, new(*http.MaxBytesError)):
			return domain.Error{
				Code: "INVALID_REQUEST",
			}
		default:
			fmt.Println("error is ", err)
			return domain.Error{
				Code:    "INVALID_REQUEST",
				Message: "request body contains invalid JSON",
			}
		}
	}

	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return domain.Error{
			Code:    "INVALID_REQUEST",
			Message: "must only have a single JSON in body",
		}
	}
	return nil
}
