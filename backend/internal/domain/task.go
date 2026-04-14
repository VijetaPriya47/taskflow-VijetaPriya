package domain

import (
	"context"
	"time"
)

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

type Task struct {
	ID              string
	Title           string
	Description     *string
	Status          TaskStatus
	Priority        TaskPriority
	ProjectID       string
	AssigneeID      *string
	DueDate         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedByUserID string
}

type TaskListFilter struct {
	Status     *TaskStatus
	AssigneeID *string
	Limit      int
	Offset     int
}

type TaskRepository interface {
	GetByID(ctx context.Context, id string) (*Task, error)
	ListByProject(ctx context.Context, projectID string, f TaskListFilter) ([]*Task, error)
	StatsByProject(ctx context.Context, projectID string) (byStatus map[TaskStatus]int64, byAssignee map[string]int64, err error)
	Create(ctx context.Context, t *Task) (*Task, error)
	Update(ctx context.Context, id string, patch TaskPatch) (*Task, error)
	Delete(ctx context.Context, id string) error
}

type TaskPatch struct {
	Title       *string
	Description *string
	Status      *TaskStatus
	Priority    *TaskPriority
	AssigneeID  **string
	DueDate     **time.Time
}

type TaskService interface {
	ListByProject(ctx context.Context, requesterUserID, projectID string, f TaskListFilter) ([]*Task, error)
	Create(ctx context.Context, requesterUserID, projectID, title string, description *string, priority TaskPriority, assigneeID *string, dueDate *time.Time) (*Task, error)
	Update(ctx context.Context, requesterUserID, taskID string, patch TaskPatch) (*Task, error)
	Delete(ctx context.Context, requesterUserID, taskID string) error
}
