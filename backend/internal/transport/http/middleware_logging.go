package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		info := requestInfoFromContext(r.Context())
		rid, uid, email := "", "", ""
		if info != nil {
			rid, uid, email = info.RequestID, info.UserID, info.UserEmail
		}
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_ip", r.RemoteAddr,
			"request_id", rid,
			"user_id", uid,
			"user_email", email,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
