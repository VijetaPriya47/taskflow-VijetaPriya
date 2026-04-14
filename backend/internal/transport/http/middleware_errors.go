package httpapi

import (
	"log/slog"
	"net/http"
)

func errorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "recover", rec, "request_id", requestIDFromContext(r.Context()))
				writeServerError(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
