package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskflow-backend/internal/domain"
)

type activityRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresActivityRepository(pool *pgxpool.Pool) domain.ActivityRepository {
	return &activityRepo{pool: pool}
}

func (r *activityRepo) Create(ctx context.Context, a *domain.Activity) (*domain.Activity, error) {
	if a == nil {
		return nil, nil
	}
	meta := []byte("null")
	if a.Metadata != nil {
		b, err := json.Marshal(a.Metadata)
		if err != nil {
			return nil, err
		}
		meta = b
	}

	var out domain.Activity
	var metaRaw []byte
	row := r.pool.QueryRow(ctx, `
INSERT INTO activities (actor_id, action, entity_type, entity_id, metadata)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, actor_id, action, entity_type, entity_id, metadata, created_at
`, a.ActorID, a.Action, a.EntityType, a.EntityID, meta)

	if err := row.Scan(&out.ID, &out.ActorID, &out.Action, &out.EntityType, &out.EntityID, &metaRaw, &out.CreatedAt); err != nil {
		return nil, err
	}
	if len(metaRaw) > 0 && string(metaRaw) != "null" {
		_ = json.Unmarshal(metaRaw, &out.Metadata)
	}
	return &out, nil
}

func (r *activityRepo) ListByActor(ctx context.Context, actorID string, limit, offset int) ([]*domain.Activity, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, actor_id, action, entity_type, entity_id, metadata, created_at
FROM activities
WHERE actor_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3
`, actorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*domain.Activity{}
	for rows.Next() {
		var a domain.Activity
		var metaRaw []byte
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.EntityType, &a.EntityID, &metaRaw, &a.CreatedAt); err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 && string(metaRaw) != "null" {
			_ = json.Unmarshal(metaRaw, &a.Metadata)
		}
		out = append(out, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

var _ domain.ActivityRepository = (*activityRepo)(nil)

