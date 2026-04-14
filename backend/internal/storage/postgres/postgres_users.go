package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskflow-backend/internal/domain"
)

type PostgresUserRepository struct {
	Pool *pgxpool.Pool
}

var _ domain.UserRepository = (*PostgresUserRepository)(nil)

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{Pool: pool}
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var created time.Time
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &created); err != nil {
		return nil, err
	}
	u.CreatedAt = created
	return &u, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u, err := scanUser(r.Pool.QueryRow(ctx, `
SELECT id::text, name, email, password_hash, created_at
FROM users WHERE id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := scanUser(r.Pool.QueryRow(ctx, `
SELECT id::text, name, email, password_hash, created_at
FROM users WHERE lower(email) = $1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *PostgresUserRepository) Create(ctx context.Context, name, email, passwordHash string) (*domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var id string
	err := r.Pool.QueryRow(ctx, `
INSERT INTO users (name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING id::text`, name, email, passwordHash).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}
