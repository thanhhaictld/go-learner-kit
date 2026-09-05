package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/haidodev/user-service/internal/domain"
	"github.com/haidodev/user-service/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) CreateUser(
	ctx context.Context,
	email string,
	name string) (*domain.User, error) {
	user := &domain.User{
		Id:        uuid.New(),
		Email:     email,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}

	err := s.repo.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetUser(
	ctx context.Context,
	id uuid.UUID) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.repo.ListUsers(ctx)
}
