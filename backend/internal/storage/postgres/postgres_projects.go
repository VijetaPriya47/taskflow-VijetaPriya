package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskflow-backend/internal/domain"
)

type PostgresProjectRepository struct {
	Pool *pgxpool.Pool
}

var _ domain.ProjectRepository = (*PostgresProjectRepository)(nil)

func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{Pool: pool}
}

func scanProject(row pgx.Row) (*domain.Project, error) {
	var p domain.Project
	var created time.Time
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &created); err != nil {
		return nil, err
	}
	p.CreatedAt = created
	return &p, nil
}

func (r *PostgresProjectRepository) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	p, err := scanProject(r.Pool.QueryRow(ctx, `
SELECT id::text, name, description, owner_id::text, created_at
FROM projects WHERE id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}

func (r *PostgresProjectRepository) ListAccessible(ctx context.Context, userID string, limit, offset int) ([]*domain.Project, error) {
	rows, err := r.Pool.Query(ctx, `
SELECT DISTINCT p.id::text, p.name, p.description, p.owner_id::text, p.created_at
FROM projects p
LEFT JOIN tasks t
  ON t.project_id = p.id
 AND (t.assignee_id = $1::uuid OR t.created_by_user_id = $1::uuid)
WHERE p.owner_id = $1::uuid OR t.id IS NOT NULL
ORDER BY p.created_at DESC, p.id::text DESC
LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Project
	for rows.Next() {
		var p domain.Project
		var created time.Time
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &created); err != nil {
			return nil, err
		}
		p.CreatedAt = created
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *PostgresProjectRepository) IsAccessible(ctx context.Context, projectID, userID string) (bool, error) {
	var n int
	err := r.Pool.QueryRow(ctx, `
SELECT COUNT(*) FROM projects p
LEFT JOIN tasks t
  ON t.project_id = p.id
 AND (t.assignee_id = $2::uuid OR t.created_by_user_id = $2::uuid)
WHERE p.id = $1::uuid AND (p.owner_id = $2::uuid OR t.id IS NOT NULL)`,
		projectID, userID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *PostgresProjectRepository) Create(ctx context.Context, ownerID, name string, description *string) (*domain.Project, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `
INSERT INTO projects (name, description, owner_id)
VALUES ($1, $2, $3::uuid)
RETURNING id::text`, name, description, ownerID).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresProjectRepository) Update(ctx context.Context, id, name string, description *string) (*domain.Project, error) {
	tag, err := r.Pool.Exec(ctx, `
UPDATE projects SET name = $2, description = $3
WHERE id = $1::uuid`, id, name, description)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresProjectRepository) Delete(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM projects WHERE id = $1::uuid`, id)
	return err
}
