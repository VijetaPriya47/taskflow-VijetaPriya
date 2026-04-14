package service

import (
	"context"
	"time"

	"taskflow-backend/internal/domain"
)

type projectServiceWithActivity struct {
	next domain.ProjectService
	act  domain.ActivityService
}

func WithProjectActivity(next domain.ProjectService, act domain.ActivityService) domain.ProjectService {
	if next == nil {
		return nil
	}
	return &projectServiceWithActivity{next: next, act: act}
}

func (s *projectServiceWithActivity) List(ctx context.Context, requesterUserID string, limit, offset int) ([]*domain.Project, error) {
	return s.next.List(ctx, requesterUserID, limit, offset)
}

func (s *projectServiceWithActivity) Create(ctx context.Context, requesterUserID, name string, description *string) (*domain.Project, error) {
	p, err := s.next.Create(ctx, requesterUserID, name, description)
	if err == nil && p != nil {
		_ = s.act.Record(ctx, requesterUserID, "project.create", "project", p.ID, map[string]any{"name": p.Name})
	}
	return p, err
}

func (s *projectServiceWithActivity) Get(ctx context.Context, requesterUserID, projectID string) (*domain.Project, []*domain.Task, error) {
	return s.next.Get(ctx, requesterUserID, projectID)
}

func (s *projectServiceWithActivity) Stats(ctx context.Context, requesterUserID, projectID string) (map[domain.TaskStatus]int64, map[string]int64, error) {
	return s.next.Stats(ctx, requesterUserID, projectID)
}

func (s *projectServiceWithActivity) Update(ctx context.Context, requesterUserID, projectID, name string, description *string) (*domain.Project, error) {
	p, err := s.next.Update(ctx, requesterUserID, projectID, name, description)
	if err == nil && p != nil {
		_ = s.act.Record(ctx, requesterUserID, "project.update", "project", p.ID, map[string]any{"name": p.Name})
	}
	return p, err
}

func (s *projectServiceWithActivity) Delete(ctx context.Context, requesterUserID, projectID string) error {
	err := s.next.Delete(ctx, requesterUserID, projectID)
	if err == nil {
		_ = s.act.Record(ctx, requesterUserID, "project.delete", "project", projectID, nil)
	}
	return err
}

var _ domain.ProjectService = (*projectServiceWithActivity)(nil)

type taskServiceWithActivity struct {
	next domain.TaskService
	act  domain.ActivityService
}

func WithTaskActivity(next domain.TaskService, act domain.ActivityService) domain.TaskService {
	if next == nil {
		return nil
	}
	return &taskServiceWithActivity{next: next, act: act}
}

func (s *taskServiceWithActivity) ListByProject(ctx context.Context, requesterUserID, projectID string, f domain.TaskListFilter) ([]*domain.Task, error) {
	return s.next.ListByProject(ctx, requesterUserID, projectID, f)
}

func (s *taskServiceWithActivity) Create(ctx context.Context, requesterUserID, projectID, title string, description *string, priority domain.TaskPriority, assigneeID *string, dueDate *time.Time) (*domain.Task, error) {
	t, err := s.next.Create(ctx, requesterUserID, projectID, title, description, priority, assigneeID, dueDate)
	if err == nil && t != nil {
		_ = s.act.Record(ctx, requesterUserID, "task.create", "task", t.ID, map[string]any{"project_id": t.ProjectID, "title": t.Title})
	}
	return t, err
}

func (s *taskServiceWithActivity) Update(ctx context.Context, requesterUserID, taskID string, patch domain.TaskPatch) (*domain.Task, error) {
	t, err := s.next.Update(ctx, requesterUserID, taskID, patch)
	if err == nil && t != nil {
		_ = s.act.Record(ctx, requesterUserID, "task.update", "task", t.ID, map[string]any{"project_id": t.ProjectID})
	}
	return t, err
}

func (s *taskServiceWithActivity) Delete(ctx context.Context, requesterUserID, taskID string) error {
	err := s.next.Delete(ctx, requesterUserID, taskID)
	if err == nil {
		_ = s.act.Record(ctx, requesterUserID, "task.delete", "task", taskID, nil)
	}
	return err
}

var _ domain.TaskService = (*taskServiceWithActivity)(nil)
