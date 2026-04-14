package service

import (
	"context"
	"testing"

	"taskflow-backend/internal/domain"
)

type memUsers struct {
	byEmail map[string]*domain.User
	nextID  int
}

func (m *memUsers) GetByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (m *memUsers) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.byEmail[email], nil
}

func (m *memUsers) Create(ctx context.Context, name, email, passwordHash string) (*domain.User, error) {
	m.nextID++
	u := &domain.User{ID: string(rune('a' + m.nextID)), Name: name, Email: email, PasswordHash: passwordHash}
	m.byEmail[email] = u
	return u, nil
}

func TestAuthRegisterValidates(t *testing.T) {
	svc := NewAuthService(&memUsers{byEmail: map[string]*domain.User{}})
	if _, _, err := svc.Register(context.Background(), "", "x@example.com", "pw"); err != domain.ErrValidation {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
