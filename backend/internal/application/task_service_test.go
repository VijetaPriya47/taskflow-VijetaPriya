package service

import (
	"context"
	"testing"

	"taskflow-backend/internal/domain"
)

type fakeTasksRepo struct {
	task *domain.Task
}

func (f *fakeTasksRepo) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	return f.task, nil
}
func (f *fakeTasksRepo) ListByProject(ctx context.Context, projectID string, flt domain.TaskListFilter) ([]*domain.Task, error) {
	return nil, nil
}
func (f *fakeTasksRepo) StatsByProject(ctx context.Context, projectID string) (map[domain.TaskStatus]int64, map[string]int64, error) {
	return nil, nil, nil
}
func (f *fakeTasksRepo) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	return t, nil
}
func (f *fakeTasksRepo) Update(ctx context.Context, id string, patch domain.TaskPatch) (*domain.Task, error) {
	return f.task, nil
}
func (f *fakeTasksRepo) Delete(ctx context.Context, id string) error { return nil }

type fakeProjectsRepo struct {
	project *domain.Project
}

func (f *fakeProjectsRepo) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	return f.project, nil
}
func (f *fakeProjectsRepo) ListAccessible(ctx context.Context, userID string, limit, offset int) ([]*domain.Project, error) {
	return nil, nil
}
func (f *fakeProjectsRepo) IsAccessible(ctx context.Context, projectID, userID string) (bool, error) {
	return true, nil
}
func (f *fakeProjectsRepo) Create(ctx context.Context, ownerID, name string, description *string) (*domain.Project, error) {
	return nil, nil
}
func (f *fakeProjectsRepo) Update(ctx context.Context, id, name string, description *string) (*domain.Project, error) {
	return nil, nil
}
func (f *fakeProjectsRepo) Delete(ctx context.Context, id string) error { return nil }

func TestTaskDeleteOwnerOrCreator(t *testing.T) {
	task := &domain.Task{ID: "t1", ProjectID: "p1", CreatedByUserID: "creator"}
	project := &domain.Project{ID: "p1", OwnerID: "owner"}
	svc := NewTaskService(&fakeProjectsRepo{project: project}, &fakeTasksRepo{task: task})

	if err := svc.Delete(context.Background(), "other", "t1"); err != domain.ErrTaskDeleteForbidden {
		t.Fatalf("expected ErrTaskDeleteForbidden, got %v", err)
	}
	if err := svc.Delete(context.Background(), "creator", "t1"); err != nil {
		t.Fatalf("creator delete: %v", err)
	}

	task2 := &domain.Task{ID: "t2", ProjectID: "p1", CreatedByUserID: "creator"}
	svc2 := NewTaskService(&fakeProjectsRepo{project: project}, &fakeTasksRepo{task: task2})
	if err := svc2.Delete(context.Background(), "owner", "t2"); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
}

func TestTaskCreateAnyAuthenticatedUser(t *testing.T) {
	project := &domain.Project{ID: "p1", OwnerID: "owner"}
	svc := NewTaskService(&fakeProjectsRepo{project: project}, &fakeTasksRepo{})

	_, err := svc.Create(context.Background(), "not-owner", "p1", "Title", nil, domain.TaskPriorityLow, nil, nil)
	if err != nil {
		t.Fatalf("expected non-owner create ok, got %v", err)
	}
}
