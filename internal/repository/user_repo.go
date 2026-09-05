package repository

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/haidodev/user-service/internal/domain"
	"gorm.io/gorm"
)

type UserRepository interface {
	// CreateUser creates a new user in the repository.
	CreateUser(ctx context.Context, user *domain.User) error

	// GetUserByID retrieves a user by their ID from the repository.
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// ListUsers retrieves all users from the repository.
	ListUsers(ctx context.Context) ([]domain.User, error)
}

type ImplUserRepository struct {
	// Add any necessary fields for the repository implementation, such as a database connection.
	dbContext *gorm.DB
}

func NewUserRepository(dbContext *gorm.DB) UserRepository {
	return &ImplUserRepository{
		dbContext: dbContext,
	}
}

func (repo *ImplUserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	// Implementation for creating a user in the repository (e.g., database)
	// This is a placeholder; actual implementation will depend on the database being used.
	return repo.dbContext.WithContext(ctx).Create(user).Error
}

func (repo *ImplUserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	log.Printf("querying with userid: %s", id)
	// Implementation for retrieving a user by ID in the repository (e.g., database)
	// This is a placeholder; actual implementation will depend on the database being used.
	var user domain.User
	err := repo.dbContext.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *ImplUserRepository) ListUsers(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	if err := repo.dbContext.WithContext(ctx).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
