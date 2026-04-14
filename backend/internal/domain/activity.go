package domain

import (
	"context"
	"time"
)

type Activity struct {
	ID         string
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
	CreatedAt  time.Time
}

type ActivityRepository interface {
	Create(ctx context.Context, a *Activity) (*Activity, error)
	ListByActor(ctx context.Context, actorID string, limit, offset int) ([]*Activity, error)
}

type ActivityService interface {
	Record(ctx context.Context, actorID, action, entityType, entityID string, metadata map[string]any) error
}

