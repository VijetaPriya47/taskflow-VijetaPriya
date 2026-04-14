package httpapi

import (
	"net/http"

	"github.com/google/uuid"
)

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = uuid.NewString()
		}
		w.Header().Set("X-Request-Id", rid)
		info := &requestInfo{RequestID: rid}
		ctx := withRequestID(r.Context(), rid)
		ctx = withRequestInfo(ctx, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
