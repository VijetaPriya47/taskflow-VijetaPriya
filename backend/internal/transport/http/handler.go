package httpapi

import (
	"log/slog"
	"net/http"
)

type handlerFunc func(http.ResponseWriter, *http.Request) error

type responseWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(p []byte) (int, error) {
	w.wroteHeader = true
	return w.ResponseWriter.Write(p)
}

func (h handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rw := &responseWriter{ResponseWriter: w}
	if err := h(rw, r); err != nil {
		slog.Debug("handler error", "err", err, "request_id", requestIDFromContext(r.Context()))
		if !rw.wroteHeader {
			WriteError(rw, r, err)
		}
	}
}
