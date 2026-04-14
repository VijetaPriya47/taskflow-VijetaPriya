package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"taskflow-backend/internal/domain"
)

type Deps struct {
	Auth     domain.AuthService
	Projects domain.ProjectService
	Tasks    domain.TaskService
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(requestIDMiddleware)
	r.Use(errorMiddleware)
	r.Use(corsMiddleware().Handler)
	r.Use(requestLogMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/auth", func(r chi.Router) {
		r.Method(http.MethodPost, "/register", handlerFunc(handleAuthRegister(d)))
		r.Method(http.MethodPost, "/login", handlerFunc(handleAuthLogin(d)))
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware())

		r.Method(http.MethodGet, "/projects", handlerFunc(handleProjectsList(d)))
		r.Method(http.MethodPost, "/projects", handlerFunc(handleProjectsCreate(d)))
		r.Method(http.MethodGet, "/projects/{id}", handlerFunc(handleProjectsGet(d)))
		r.Method(http.MethodPatch, "/projects/{id}", handlerFunc(handleProjectsPatch(d)))
		r.Method(http.MethodDelete, "/projects/{id}", handlerFunc(handleProjectsDelete(d)))
		r.Method(http.MethodGet, "/projects/{id}/tasks", handlerFunc(handleProjectTasksList(d)))
		r.Method(http.MethodPost, "/projects/{id}/tasks", handlerFunc(handleProjectTasksCreate(d)))
		r.Method(http.MethodGet, "/projects/{id}/stats", handlerFunc(handleProjectStats(d)))

		r.Method(http.MethodPatch, "/tasks/{id}", handlerFunc(handleTasksPatch(d)))
		r.Method(http.MethodDelete, "/tasks/{id}", handlerFunc(handleTasksDelete(d)))
	})

	return r
}
