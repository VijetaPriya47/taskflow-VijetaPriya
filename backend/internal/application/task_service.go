package service

import (
	"context"
	"strings"
	"time"

	"taskflow-backend/internal/domain"
)

type taskService struct {
	projects domain.ProjectRepository
	tasks    domain.TaskRepository
}

func NewTaskService(projects domain.ProjectRepository, tasks domain.TaskRepository) domain.TaskService {
	return &taskService{projects: projects, tasks: tasks}
}

func (s *taskService) ListByProject(ctx context.Context, requesterUserID, projectID string, f domain.TaskListFilter) ([]*domain.Task, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, domain.ErrUnauthorized
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrValidation
	}
	ok, err := s.projects.IsAccessible(ctx, projectID, requesterUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.ErrNotFound
	}
	return s.tasks.ListByProject(ctx, projectID, f)
}

func (s *taskService) Create(ctx context.Context, requesterUserID, projectID, title string, description *string, priority domain.TaskPriority, assigneeID *string, dueDate *time.Time) (*domain.Task, error) {
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
	// Any authenticated user may create a task in any existing project.
	// Task delete is limited to project owner or the user who created the task (see Delete).

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.ErrValidation
	}

	if priority == "" {
		priority = domain.TaskPriorityMedium
	}
	switch priority {
	case domain.TaskPriorityLow, domain.TaskPriorityMedium, domain.TaskPriorityHigh:
	default:
		return nil, domain.ErrValidation
	}

	t := &domain.Task{
		Title:           title,
		Description:     description,
		Status:          domain.TaskStatusTodo,
		Priority:        priority,
		ProjectID:       projectID,
		AssigneeID:      assigneeID,
		DueDate:         dueDate,
		CreatedByUserID: requesterUserID,
	}
	return s.tasks.Create(ctx, t)
}

func (s *taskService) Update(ctx context.Context, requesterUserID, taskID string, patch domain.TaskPatch) (*domain.Task, error) {
	if strings.TrimSpace(requesterUserID) == "" {
		return nil, domain.ErrUnauthorized
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, domain.ErrValidation
	}
	existing, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.ErrNotFound
	}
	p, err := s.projects.GetByID(ctx, existing.ProjectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domain.ErrNotFound
	}
	isOwner := p.OwnerID == requesterUserID
	isCreator := existing.CreatedByUserID == requesterUserID
	isAssignee := existing.AssigneeID != nil && *existing.AssigneeID == requesterUserID
	if !isOwner && !isCreator && !isAssignee {
		return nil, domain.ErrForbidden
	}

	if patch.Status != nil {
		switch *patch.Status {
		case domain.TaskStatusTodo, domain.TaskStatusInProgress, domain.TaskStatusDone:
		default:
			return nil, domain.ErrValidation
		}
	}
	if patch.Priority != nil {
		switch *patch.Priority {
		case domain.TaskPriorityLow, domain.TaskPriorityMedium, domain.TaskPriorityHigh:
		default:
			return nil, domain.ErrValidation
		}
	}
	if patch.Title != nil && strings.TrimSpace(*patch.Title) == "" {
		return nil, domain.ErrValidation
	}

	updated, err := s.tasks.Update(ctx, taskID, patch)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, domain.ErrNotFound
	}
	return updated, nil
}

func (s *taskService) Delete(ctx context.Context, requesterUserID, taskID string) error {
	if strings.TrimSpace(requesterUserID) == "" {
		return domain.ErrUnauthorized
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return domain.ErrValidation
	}
	existing, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	p, err := s.projects.GetByID(ctx, existing.ProjectID)
	if err != nil {
		return err
	}
	if p == nil {
		return domain.ErrNotFound
	}
	// Only the project owner or the user who created this task (POST) may delete it.
	if p.OwnerID != requesterUserID && existing.CreatedByUserID != requesterUserID {
		return domain.ErrTaskDeleteForbidden
	}
	return s.tasks.Delete(ctx, taskID)
}
