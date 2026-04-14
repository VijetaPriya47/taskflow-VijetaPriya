package service

import (
	"context"
	"testing"

	"taskflow-backend/internal/domain"
)

type fakeProjects struct {
	p *domain.Project
}

func (f *fakeProjects) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	return f.p, nil
}
func (f *fakeProjects) ListAccessible(ctx context.Context, userID string, limit, offset int) ([]*domain.Project, error) {
	return []*domain.Project{f.p}, nil
}
func (f *fakeProjects) IsAccessible(ctx context.Context, projectID, userID string) (bool, error) {
	return true, nil
}
func (f *fakeProjects) Create(ctx context.Context, ownerID, name string, description *string) (*domain.Project, error) {
	return &domain.Project{ID: "p1", OwnerID: ownerID, Name: name, Description: description}, nil
}
func (f *fakeProjects) Update(ctx context.Context, id, name string, description *string) (*domain.Project, error) {
	f.p.Name = name
	f.p.Description = description
	return f.p, nil
}
func (f *fakeProjects) Delete(ctx context.Context, id string) error { return nil }

type fakeTasks struct{}

func (f *fakeTasks) GetByID(ctx context.Context, id string) (*domain.Task, error) { return nil, nil }
func (f *fakeTasks) ListByProject(ctx context.Context, projectID string, flt domain.TaskListFilter) ([]*domain.Task, error) {
	return nil, nil
}
func (f *fakeTasks) StatsByProject(ctx context.Context, projectID string) (map[domain.TaskStatus]int64, map[string]int64, error) {
	return nil, nil, nil
}
func (f *fakeTasks) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	return nil, nil
}
func (f *fakeTasks) Update(ctx context.Context, id string, patch domain.TaskPatch) (*domain.Task, error) {
	return nil, nil
}
func (f *fakeTasks) Delete(ctx context.Context, id string) error { return nil }

func TestProjectUpdateOwnerOnly(t *testing.T) {
	p := &domain.Project{ID: "p1", OwnerID: "owner", Name: "n"}
	svc := NewProjectService(&fakeProjects{p: p}, &fakeTasks{})
	if _, err := svc.Update(context.Background(), "other", "p1", "x", nil); err != domain.ErrProjectOwnerOnly {
		t.Fatalf("expected ErrProjectOwnerOnly, got %v", err)
	}
}
