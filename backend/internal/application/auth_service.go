package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"taskflow-backend/internal/domain"
	"taskflow-backend/internal/infrastructure/security"
)

type authService struct {
	users domain.UserRepository
	now   func() time.Time
}

func NewAuthService(users domain.UserRepository) domain.AuthService {
	return &authService{
		users: users,
		now:   time.Now,
	}
}

func (s *authService) Register(ctx context.Context, name, email, password string) (string, *domain.User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)
	if name == "" || email == "" || password == "" {
		return "", nil, domain.ErrValidation
	}

	existing, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if existing != nil {
		return "", nil, domain.ErrConflict
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return "", nil, err
	}
	u, err := s.users.Create(ctx, name, email, hash)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			return "", nil, domain.ErrConflict
		}
		return "", nil, err
	}

	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	tok, err := security.SignAccessToken(jwtSecret, u.ID, u.Email, s.now())
	if err != nil {
		return "", nil, err
	}
	return tok, u, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return "", nil, domain.ErrValidation
	}
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		return "", nil, domain.ErrInvalidCredentials
	}
	if err := security.ComparePasswordHash(u.PasswordHash, password); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))
	tok, err := security.SignAccessToken(jwtSecret, u.ID, u.Email, s.now())
	if err != nil {
		return "", nil, err
	}
	return tok, u, nil
}
