package domain

import (
	"context"
	"time"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, name, email, passwordHash string) (*User, error)
}

type AuthService interface {
	Register(ctx context.Context, name, email, password string) (jwt string, user *User, err error)
	Login(ctx context.Context, email, password string) (jwt string, user *User, err error)
}
