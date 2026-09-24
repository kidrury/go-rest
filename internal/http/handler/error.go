package handler

// import (
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"log/slog"
// 	"net/http"

// 	"github.com/kidrury/rest-pro/internal/domain"
// )

// type errorResponse struct {
// 	Error errorBody `json:"error"`
// }

// type errorBody struct {
// 	Code    string            `json:"code"`
// 	Message string            `json:"message"`
// 	Fields  map[string]string `json:"fields,omitempty"`
// }

// func statusFromError(err error) int {
// 	var domainError domain.Error

// 	if errors.As(err, &domainError) {
// 		switch domainError.Code {
// 		case "NOT_FOUND":
// 			return http.StatusNotFound
// 		case "INVALID_REQUEST":
// 			return http.StatusBadRequest
// 		case "VALIDATION_FAILED":
// 			return http.StatusBadRequest
// 		case "INVALID_PATH_PARAMETER":
// 			return http.StatusBadRequest
// 		case "INVALID_QUERY_PARAMETER":
// 			return http.StatusBadRequest
// 		case "UNAUTHORIZED":
// 			return http.StatusUnauthorized
// 		case "FORBIDDEN":
// 			return http.StatusForbidden
// 		}
// 	}
// 	return http.StatusInternalServerError
// }

// func writeJSONError(w http.ResponseWriter, status int, code string, message string, fields map[string]string) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(status)
// 	err := errorResponse{
// 		errorBody{
// 			Code:    code,
// 			Message: message,
// 			Fields:  fields,
// 		},
// 	}
// 	json.NewEncoder(w).Encode(err)
// }

// func HandleError(w http.ResponseWriter, r *http.Request, err error) {
// 	var domainErr domain.Error

// 	if errors.As(err, &domainErr) {
// 		status := statusFromError(err)
// 		writeJSONError(w, status, domainErr.Code, domainErr.Message, domainErr.Fields)

// 		return
// 	}

// 	slog.ErrorContext(
// 		r.Context(), "unhandled application error",
// 		"method", r.Method,
// 		"path", r.URL.Path,
// 		"error", err,
// 	)
// 	fmt.Println("error is:")
// 	fmt.Println(err)

// 	writeJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
// }
