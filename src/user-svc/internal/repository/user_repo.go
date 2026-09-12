package repository

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/haidodev/user-service/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrEmailAlreadyExists = errors.New("email already exists")

type UserRepository interface {
	// CreateUser creates a new user in the repository.
	CreateUser(ctx context.Context, user *domain.User) error

	// GetUserByID retrieves a user by their ID from the repository.
	GetUserByID(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) (*domain.User, error)

	// ListUsers retrieves all users from the repository.
	ListUsers(ctx context.Context, organizationID uuid.UUID) ([]domain.User, error)
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
	if err := repo.dbContext.WithContext(ctx).Create(user).Error; err != nil {
		if isUniqueEmailViolation(err) {
			return ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func isUniqueEmailViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "ux_users_organization_email"
}

func (repo *ImplUserRepository) GetUserByID(ctx context.Context, id uuid.UUID, organizationID uuid.UUID) (*domain.User, error) {
	log.Printf("querying with userid: %s", id)
	// Implementation for retrieving a user by ID in the repository (e.g., database)
	// This is a placeholder; actual implementation will depend on the database being used.
	var user domain.User
	err := repo.dbContext.WithContext(ctx).Where("id = ? AND organization_id = ?", id, organizationID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *ImplUserRepository) ListUsers(ctx context.Context, organizationID uuid.UUID) ([]domain.User, error) {
	var users []domain.User
	if err := repo.dbContext.WithContext(ctx).Where("organization_id = ?", organizationID).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
