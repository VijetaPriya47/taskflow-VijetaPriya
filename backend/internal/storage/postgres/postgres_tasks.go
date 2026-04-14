package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskflow-backend/internal/domain"
)

type PostgresTaskRepository struct {
	Pool *pgxpool.Pool
}

var _ domain.TaskRepository = (*PostgresTaskRepository)(nil)

func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{Pool: pool}
}

func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	var created, updated time.Time
	if err := row.Scan(
		&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedByUserID,
		&created, &updated,
	); err != nil {
		return nil, err
	}
	t.CreatedAt = created
	t.UpdatedAt = updated
	return &t, nil
}

func (r *PostgresTaskRepository) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	t, err := scanTask(r.Pool.QueryRow(ctx, `
SELECT id::text, title, description, status, priority, project_id::text, assignee_id::text, due_date, created_by_user_id::text, created_at, updated_at
FROM tasks WHERE id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

func (r *PostgresTaskRepository) ListByProject(ctx context.Context, projectID string, f domain.TaskListFilter) ([]*domain.Task, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	var status any
	if f.Status != nil {
		status = string(*f.Status)
	}
	var assignee any
	if f.AssigneeID != nil {
		assignee = *f.AssigneeID
	}

	rows, err := r.Pool.Query(ctx, `
SELECT id::text, title, description, status, priority, project_id::text, assignee_id::text, due_date, created_by_user_id::text, created_at, updated_at
FROM tasks
WHERE project_id = $1::uuid
  AND ($2::text IS NULL OR status = $2)
  AND ($3::uuid IS NULL OR assignee_id = $3::uuid)
ORDER BY created_at DESC, id DESC
LIMIT $4 OFFSET $5`, projectID, status, assignee, f.Limit, f.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Task
	for rows.Next() {
		var t domain.Task
		var created, updated time.Time
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedByUserID,
			&created, &updated,
		); err != nil {
			return nil, err
		}
		t.CreatedAt = created
		t.UpdatedAt = updated
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *PostgresTaskRepository) StatsByProject(ctx context.Context, projectID string) (map[domain.TaskStatus]int64, map[string]int64, error) {
	byStatus := map[domain.TaskStatus]int64{}
	rows, err := r.Pool.Query(ctx, `
SELECT status, COUNT(*) FROM tasks
WHERE project_id = $1::uuid
GROUP BY status`, projectID)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			rows.Close()
			return nil, nil, err
		}
		byStatus[domain.TaskStatus(s)] = n
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()

	byAssignee := map[string]int64{}
	rows2, err := r.Pool.Query(ctx, `
SELECT COALESCE(assignee_id::text, ''), COUNT(*) FROM tasks
WHERE project_id = $1::uuid
GROUP BY COALESCE(assignee_id::text, '')`, projectID)
	if err != nil {
		return nil, nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var aid string
		var n int64
		if err := rows2.Scan(&aid, &n); err != nil {
			return nil, nil, err
		}
		byAssignee[aid] = n
	}
	return byStatus, byAssignee, rows2.Err()
}

func (r *PostgresTaskRepository) Create(ctx context.Context, t *domain.Task) (*domain.Task, error) {
	var id string
	err := r.Pool.QueryRow(ctx, `
INSERT INTO tasks (title, description, status, priority, project_id, assignee_id, due_date, created_by_user_id)
VALUES ($1, $2, $3, $4, $5::uuid, $6::uuid, $7, $8::uuid)
RETURNING id::text`,
		t.Title, t.Description, string(t.Status), string(t.Priority), t.ProjectID, t.AssigneeID, t.DueDate, t.CreatedByUserID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresTaskRepository) Update(ctx context.Context, id string, patch domain.TaskPatch) (*domain.Task, error) {
	sets := make([]string, 0, 8)
	args := make([]any, 0, 10)
	args = append(args, id)
	argN := 2

	if patch.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", argN))
		args = append(args, *patch.Title)
		argN++
	}
	if patch.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argN))
		args = append(args, patch.Description)
		argN++
	}
	if patch.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argN))
		args = append(args, string(*patch.Status))
		argN++
	}
	if patch.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", argN))
		args = append(args, string(*patch.Priority))
		argN++
	}
	if patch.AssigneeID != nil {
		sets = append(sets, fmt.Sprintf("assignee_id = $%d::uuid", argN))
		args = append(args, *patch.AssigneeID)
		argN++
	}
	if patch.DueDate != nil {
		sets = append(sets, fmt.Sprintf("due_date = $%d", argN))
		args = append(args, *patch.DueDate)
		argN++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}
	sets = append(sets, "updated_at = now()")

	q := fmt.Sprintf(`UPDATE tasks SET %s WHERE id = $1::uuid`, strings.Join(sets, ", "))
	tag, err := r.Pool.Exec(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return r.GetByID(ctx, id)
}

func (r *PostgresTaskRepository) Delete(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1::uuid`, id)
	return err
}
