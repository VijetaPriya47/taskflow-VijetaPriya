package service

import (
	"context"
	"strings"
	"time"

	"taskflow-backend/internal/domain"
)

type activityService struct {
	repo domain.ActivityRepository
	now  func() time.Time
}

func NewActivityService(repo domain.ActivityRepository) domain.ActivityService {
	return &activityService{repo: repo, now: time.Now}
}

func (s *activityService) Record(ctx context.Context, actorID, action, entityType, entityID string, metadata map[string]any) error {
	if s.repo == nil {
		return nil
	}
	if strings.TrimSpace(actorID) == "" {
		return nil
	}
	if strings.TrimSpace(action) == "" || strings.TrimSpace(entityType) == "" || strings.TrimSpace(entityID) == "" {
		return nil
	}
	_, err := s.repo.Create(ctx, &domain.Activity{
		ActorID:    strings.TrimSpace(actorID),
		Action:     strings.TrimSpace(action),
		EntityType: strings.TrimSpace(entityType),
		EntityID:   strings.TrimSpace(entityID),
		Metadata:   metadata,
		CreatedAt:  s.now(),
	})
	return err
}

var _ domain.ActivityService = (*activityService)(nil)
