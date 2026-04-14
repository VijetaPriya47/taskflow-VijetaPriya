package httpapi

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"taskflow-backend/internal/domain"
)

type accessTokenClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func authMiddleware() func(http.Handler) http.Handler {
	secret := []byte(os.Getenv("JWT_SECRET"))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" || !strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			raw := strings.TrimSpace(authz[len("Bearer "):])
			if raw == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if len(secret) == 0 {
				writeError(w, http.StatusServiceUnavailable, "auth not configured")
				return
			}

			var claims accessTokenClaims
			tok, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, domain.ErrInvalidToken
				}
				return secret, nil
			}, jwt.WithLeeway(1*time.Minute))
			if err != nil || tok == nil || !tok.Valid {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.Email) == "" {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			uid := strings.TrimSpace(claims.UserID)
			email := strings.TrimSpace(claims.Email)
			if info := requestInfoFromContext(r.Context()); info != nil {
				info.UserID = uid
				info.UserEmail = email
			}
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), uid, email)))
		})
	}
}
