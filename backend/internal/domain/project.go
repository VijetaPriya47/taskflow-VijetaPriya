package domain

import (
	"context"
	"time"
)

type Project struct {
	ID          string
	Name        string
	Description *string
	OwnerID     string
	CreatedAt   time.Time
}

type ProjectRepository interface {
	GetByID(ctx context.Context, id string) (*Project, error)
	ListAccessible(ctx context.Context, userID string, limit, offset int) ([]*Project, error)
	IsAccessible(ctx context.Context, projectID, userID string) (bool, error)
	Create(ctx context.Context, ownerID, name string, description *string) (*Project, error)
	Update(ctx context.Context, id, name string, description *string) (*Project, error)
	Delete(ctx context.Context, id string) error
}

type ProjectService interface {
	List(ctx context.Context, requesterUserID string, limit, offset int) ([]*Project, error)
	Create(ctx context.Context, requesterUserID, name string, description *string) (*Project, error)
	Get(ctx context.Context, requesterUserID, projectID string) (*Project, []*Task, error)
	Stats(ctx context.Context, requesterUserID, projectID string) (byStatus map[TaskStatus]int64, byAssignee map[string]int64, err error)
	Update(ctx context.Context, requesterUserID, projectID, name string, description *string) (*Project, error)
	Delete(ctx context.Context, requesterUserID, projectID string) error
}
