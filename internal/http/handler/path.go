package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/kidrury/rest-pro/internal/domain"
)

func Int64PathValue(r *http.Request, name string) (int64, error) {
	v := r.PathValue(name)
	if v == "" {
		return 0, domain.Error{
			Code:    "INVALID_PATH_PARAMETER",
			Message: fmt.Sprintf("path parameter %s required", name),
		}
	}

	parsed, err := strconv.ParseInt(v, 10, 64)

	if err != nil {
		return 0, domain.Error{
			Code:    "INVALID_PATH_PARAMETER",
			Message: fmt.Sprintf("path parameter %s must be a valid integer", name),
		}
	}

	return parsed, nil
}

func UUIDPathValue(r *http.Request, name string) (uuid.UUID, error) {
	v := r.PathValue(name)
	if v == "" {
		return uuid.Nil, domain.Error{
			Code:    "INVALID_PATH_PARAMETER",
			Message: fmt.Sprintf("path parameter %s required", name),
		}
	}

	uuidParsed, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, domain.Error{
			Code:    "INVALID_PATH_PARAMETER",
			Message: fmt.Sprintf("path parameter %s must be a valid uuid", name),
		}
	}

	return uuidParsed, nil
}
