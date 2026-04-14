package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"taskflow-backend/internal/domain"
	"taskflow-backend/internal/validator"
)

type taskResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	ProjectID   string  `json:"project_id"`
	AssigneeID  *string `json:"assignee_id,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toTaskResponse(t *domain.Task) *taskResponse {
	var due *string
	if t.DueDate != nil {
		s := t.DueDate.UTC().Format("2006-01-02")
		due = &s
	}
	return &taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Priority:    string(t.Priority),
		ProjectID:   t.ProjectID,
		AssigneeID:  t.AssigneeID,
		DueDate:     due,
		CreatedAt:   t.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func handleProjectTasksList(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Tasks == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		projectID := strings.TrimSpace(chi.URLParam(r, "id"))
		if projectID == "" {
			return InvalidURL()
		}

		limit, offset, fields := parsePageLimit(r)
		if fields != nil {
			return Validation(fields)
		}

		var f domain.TaskListFilter
		f.Limit, f.Offset = limit, offset

		if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
			s := domain.TaskStatus(v)
			switch s {
			case domain.TaskStatusTodo, domain.TaskStatusInProgress, domain.TaskStatusDone:
				f.Status = &s
			default:
				return Validation(map[string]string{"status": "must be todo|in_progress|done"})
			}
		}
		if v := strings.TrimSpace(r.URL.Query().Get("assignee")); v != "" {
			f.AssigneeID = &v
		}

		tasks, err := d.Tasks.ListByProject(r.Context(), userID, projectID, f)
		if err != nil {
			return err
		}
		out := make([]*taskResponse, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, toTaskResponse(t))
		}
		writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
		return nil
	}
}

type createTaskRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Priority    string  `json:"priority"`
	AssigneeID  *string `json:"assignee_id"`
	DueDate     *string `json:"due_date"`
}

func handleProjectTasksCreate(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Tasks == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		projectID := strings.TrimSpace(chi.URLParam(r, "id"))
		if projectID == "" {
			return InvalidURL()
		}

		var req createTaskRequest
		if err := decodeJSON(r, &req); err != nil {
			return BadJSON()
		}
		req.Title = strings.TrimSpace(req.Title)
		pri := domain.TaskPriority(strings.TrimSpace(req.Priority))
		if pri == "" {
			pri = domain.TaskPriorityMedium
		}

		v := validator.New()
		v.Required("title", req.Title)
		v.OneOf("priority", string(pri), string(domain.TaskPriorityLow), string(domain.TaskPriorityMedium), string(domain.TaskPriorityHigh))
		var due *time.Time
		if req.DueDate != nil {
			due = v.DateYYYYMMDD("due_date", *req.DueDate)
		}
		if !v.Ok() {
			// keep previous API error message for priority
			if v.Fields["priority"] == "is invalid" {
				v.Fields["priority"] = "must be low|medium|high"
			}
			return Validation(v.Fields)
		}

		t, err := d.Tasks.Create(r.Context(), userID, projectID, req.Title, req.Description, pri, req.AssigneeID, due)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusCreated, toTaskResponse(t))
		return nil
	}
}

type patchTaskRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Status      *string  `json:"status"`
	Priority    *string  `json:"priority"`
	AssigneeID  **string `json:"assignee_id"`
	DueDate     **string `json:"due_date"`
}

func handleTasksPatch(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Tasks == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		taskID := strings.TrimSpace(chi.URLParam(r, "id"))
		if taskID == "" {
			return InvalidURL()
		}

		var req patchTaskRequest
		if err := decodeJSON(r, &req); err != nil {
			return BadJSON()
		}

		var patch domain.TaskPatch
		if req.Title != nil {
			s := strings.TrimSpace(*req.Title)
			patch.Title = &s
		}
		if req.Description != nil {
			patch.Description = req.Description
		}
		if req.Status != nil {
			st := domain.TaskStatus(strings.TrimSpace(*req.Status))
			patch.Status = &st
		}
		if req.Priority != nil {
			p := domain.TaskPriority(strings.TrimSpace(*req.Priority))
			patch.Priority = &p
		}
		if req.AssigneeID != nil {
			patch.AssigneeID = req.AssigneeID
		}
		if req.DueDate != nil {
			if *req.DueDate == nil || strings.TrimSpace(**req.DueDate) == "" {
				patch.DueDate = func() **time.Time { var n *time.Time; return &n }()
			} else {
				v := validator.New()
				parsed := v.DateYYYYMMDD("due_date", **req.DueDate)
				if !v.Ok() || parsed == nil {
					return Validation(v.Fields)
				}
				tt := *parsed
				patch.DueDate = func() **time.Time { p := &tt; return &p }()
			}
		}

		t, err := d.Tasks.Update(r.Context(), userID, taskID, patch)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, toTaskResponse(t))
		return nil
	}
}

func handleTasksDelete(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Tasks == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		taskID := strings.TrimSpace(chi.URLParam(r, "id"))
		if taskID == "" {
			return InvalidURL()
		}
		if err := d.Tasks.Delete(r.Context(), userID, taskID); err != nil {
			return err
		}
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}

func handleProjectStats(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Projects == nil {
			return ServiceUnavailable("service unavailable")
		}
		userID, err := RequireUser(r.Context())
		if err != nil {
			return err
		}
		projectID := strings.TrimSpace(chi.URLParam(r, "id"))
		if projectID == "" {
			return InvalidURL()
		}
		byStatus, byAssignee, err := d.Projects.Stats(r.Context(), userID, projectID)
		if err != nil {
			return err
		}

		statusOut := map[string]int64{}
		for k, v := range byStatus {
			statusOut[string(k)] = v
		}
		assigneeOut := map[string]int64{}
		for k, v := range byAssignee {
			if k == "" {
				assigneeOut["unassigned"] = v
			} else {
				assigneeOut[k] = v
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"by_status":   statusOut,
			"by_assignee": assigneeOut,
		})
		return nil
	}
}
