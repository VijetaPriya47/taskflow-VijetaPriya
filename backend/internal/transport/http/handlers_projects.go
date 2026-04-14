package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"taskflow-backend/internal/domain"
	"taskflow-backend/internal/validator"
)

type projectResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	OwnerID     string  `json:"owner_id"`
	CreatedAt   string  `json:"created_at"`
}

type projectWithTasksResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	OwnerID     string          `json:"owner_id"`
	CreatedAt   string          `json:"created_at"`
	Tasks       []*taskResponse `json:"tasks"`
}

func toProjectResponse(p *domain.Project) *projectResponse {
	return &projectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		OwnerID:     p.OwnerID,
		CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func handleProjectsList(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Projects == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		limit, offset, fields := parsePageLimit(r)
		if fields != nil {
			return Validation(fields)
		}
		ps, err := d.Projects.List(r.Context(), userID, limit, offset)
		if err != nil {
			slog.Error("projects list failed", "err", err, "request_id", requestIDFromContext(r.Context()), "user_id", userID)
			return err
		}
		out := make([]*projectResponse, 0, len(ps))
		for _, p := range ps {
			out = append(out, toProjectResponse(p))
		}
		writeJSON(w, http.StatusOK, map[string]any{"projects": out})
		return nil
	}
}

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

func handleProjectsCreate(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Projects == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		var req createProjectRequest
		if err := decodeJSON(r, &req); err != nil {
			return BadJSON()
		}
		req.Name = strings.TrimSpace(req.Name)
		v := validator.New()
		v.Required("name", req.Name)
		if !v.Ok() {
			return Validation(v.Fields)
		}
		p, err := d.Projects.Create(r.Context(), userID, req.Name, req.Description)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusCreated, toProjectResponse(p))
		return nil
	}
}

func handleProjectsGet(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Projects == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		id := chi.URLParam(r, "id")
		id = strings.TrimSpace(id)
		if id == "" {
			return InvalidURL()
		}
		p, tasks, err := d.Projects.Get(r.Context(), userID, id)
		if err != nil {
			return err
		}
		tr := make([]*taskResponse, 0, len(tasks))
		for _, t := range tasks {
			tr = append(tr, toTaskResponse(t))
		}
		writeJSON(w, http.StatusOK, projectWithTasksResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			OwnerID:     p.OwnerID,
			CreatedAt:   p.CreatedAt.UTC().Format(time.RFC3339),
			Tasks:       tr,
		})
		return nil
	}
}

type patchProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func handleProjectsPatch(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Projects == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			return InvalidURL()
		}
		var req patchProjectRequest
		if err := decodeJSON(r, &req); err != nil {
			return BadJSON()
		}
		name := ""
		if req.Name != nil {
			name = strings.TrimSpace(*req.Name)
		}
		p, err := d.Projects.Update(r.Context(), userID, id, name, req.Description)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, toProjectResponse(p))
		return nil
	}
}

func handleProjectsDelete(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Projects == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		id := strings.TrimSpace(chi.URLParam(r, "id"))
		if id == "" {
			return InvalidURL()
		}
		if err := d.Projects.Delete(r.Context(), userID, id); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}
