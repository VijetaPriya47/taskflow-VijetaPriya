package service

import (
	"context"
	"strings"

	"taskflow-backend/internal/domain"
)

type projectService struct {
	projects domain.ProjectRepository
	tasks    domain.TaskRepository
}

func NewProjectService(projects domain.ProjectRepository, tasks domain.TaskRepository) domain.ProjectService {
	return &projectService{projects: projects, tasks: tasks}
}

func (s *projectService) List(ctx context.Context, requesterUserID string, limit, offset int) ([]*domain.Project, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.projects.ListAccessible(ctx, requesterUserID, limit, offset)
}

func (s *projectService) Create(ctx context.Context, requesterUserID, name string, description *string) (*domain.Project, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, domain.ErrUnauthorized
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	return s.projects.Create(ctx, requesterUserID, name, description)
}

func (s *projectService) Get(ctx context.Context, requesterUserID, projectID string) (*domain.Project, []*domain.Task, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, nil, domain.ErrUnauthorized
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, nil, domain.ErrValidation
	}

	ok, err := s.projects.IsAccessible(ctx, projectID, requesterUserID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		// avoid leaking existence
		return nil, nil, domain.ErrNotFound
	}

	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}
	if p == nil {
		return nil, nil, domain.ErrNotFound
	}

	tasks, err := s.tasks.ListByProject(ctx, projectID, domain.TaskListFilter{Limit: 200, Offset: 0})
	if err != nil {
		return nil, nil, err
	}
	return p, tasks, nil
}

func (s *projectService) Stats(ctx context.Context, requesterUserID, projectID string) (map[domain.TaskStatus]int64, map[string]int64, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, nil, domain.ErrUnauthorized
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, nil, domain.ErrValidation
	}
	ok, err := s.projects.IsAccessible(ctx, projectID, requesterUserID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, domain.ErrNotFound
	}
	return s.tasks.StatsByProject(ctx, projectID)
}

func (s *projectService) Update(ctx context.Context, requesterUserID, projectID, name string, description *string) (*domain.Project, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, domain.ErrUnauthorized
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrValidation
	}

	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	if p.OwnerID != requesterUserID {
		return nil, domain.ErrProjectOwnerOnly
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = p.Name
	}
	if description == nil {
		description = p.Description
	}

	updated, err := s.projects.Update(ctx, projectID, name, description)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, domain.ErrNotFound
	}
	return updated, nil
}

func (s *projectService) Delete(ctx context.Context, requesterUserID, projectID string) error {
	if strings.TrimSpace(requesterUserID) == "" {
		return domain.ErrUnauthorized
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return domain.ErrValidation
	}
	p, err := s.projects.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if p == nil {
		return domain.ErrNotFound
	}
	if p.OwnerID != requesterUserID {
		return domain.ErrProjectOwnerOnly
	}
	return s.projects.Delete(ctx, projectID)
}
