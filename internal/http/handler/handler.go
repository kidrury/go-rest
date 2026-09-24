package handler

import (
	"net/http"

	"github.com/kidrury/rest-pro/internal/http/response"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		response.HandleError(w, r, err)
	}
}
