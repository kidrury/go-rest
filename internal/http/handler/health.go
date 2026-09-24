package handler

import (
	"net/http"
)

func Live(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"live"}`))
}

func Ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

func NotExistTest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func CreatedTest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
}

func PanicTest(w http.ResponseWriter, r *http.Request) {
	panic("I am panicking")
}

func PanicAfterWriteTest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("am I writing?"))
	panic("I am panicking")
}
