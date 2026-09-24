package middleware

import "net/http"

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriter) WriteHeader(code int) {
	if w.statusCode != 0 {
		return
	}

	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.statusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(body)
	w.bytesWritten += n

	return n, err
}

func (w *responseWriter) StatusCode() int {
	return w.statusCode
}

func (w *responseWriter) Committed() bool {
	return w.statusCode != 0
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
