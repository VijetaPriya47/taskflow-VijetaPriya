package httpapi

import (
	"os"
	"strings"

	"github.com/go-chi/cors"
)

func corsMiddleware() *cors.Cors {
	origins := os.Getenv("CORS_ALLOWED_ORIGINS")
	allowed := []string{"http://localhost:3000"}
	if origins != "" {
		allowed = nil
		for _, o := range strings.Split(origins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowed = append(allowed, o)
			}
		}
	}
	return cors.New(cors.Options{
		AllowedOrigins: allowed,
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders: []string{"X-Request-Id"},
		MaxAge:         300,
	})
}
