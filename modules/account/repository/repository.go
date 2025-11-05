package repository

import (
	"context"
	"time"

	"github.com/elskow/go-microservice-template/database/entities"
	"github.com/elskow/go-microservice-template/pkg/database"
	pkgerrors "github.com/elskow/go-microservice-template/pkg/errors"
	"github.com/google/uuid"
)

type Repository interface {
	CreateUser(ctx context.Context, user entities.User) (entities.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (entities.User, error)
	GetUserByEmail(ctx context.Context, email string) (entities.User, error)
	UpdateUser(ctx context.Context, user entities.User) (entities.User, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error

	CreateRefreshToken(ctx context.Context, token entities.RefreshToken) (entities.RefreshToken, error)
	GetRefreshTokenByToken(ctx context.Context, token string) (entities.RefreshToken, error)
	UpdateRefreshToken(ctx context.Context, tokenID uuid.UUID, newToken string, expiresAt time.Time) error
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error
}

type repository struct {
	db *database.TracedDB
}

func NewRepository(db *database.TracedDB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateUser(ctx context.Context, user entities.User) (entities.User, error) {
	query := `
		INSERT INTO users (id, name, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, email, password, created_at, updated_at
	`
	var created entities.User
	err := r.db.QueryRowxContext(ctx, query, user.ID, user.Name, user.Email, user.Password).StructScan(&created)
	if err != nil {
		return entities.User{}, pkgerrors.Wrap(err, "failed to create user")
	}
	return created, nil
}

func (r *repository) GetUserByID(ctx context.Context, userID uuid.UUID) (entities.User, error) {
	var user entities.User
	query := `SELECT id, name, email, password, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, userID)
	if err != nil {
		return entities.User{}, pkgerrors.Wrap(err, "failed to get user by id")
	}
	return user, nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (entities.User, error) {
	var user entities.User
	query := `SELECT id, name, email, password, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return entities.User{}, pkgerrors.Wrap(err, "failed to get user by email")
	}
	return user, nil
}

func (r *repository) UpdateUser(ctx context.Context, user entities.User) (entities.User, error) {
	query := `
		UPDATE users SET name = $1, email = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, email, password, created_at, updated_at
	`
	var updated entities.User
	err := r.db.QueryRowxContext(ctx, query, user.Name, user.Email, user.ID).StructScan(&updated)
	if err != nil {
		return entities.User{}, pkgerrors.Wrap(err, "failed to update user")
	}
	return updated, nil
}

func (r *repository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to delete user")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return pkgerrors.Wrap(err, "failed to get rows affected")
	}

	if rows == 0 {
		return pkgerrors.New("user not found")
	}

	return nil
}

func (r *repository) CreateRefreshToken(ctx context.Context, token entities.RefreshToken) (entities.RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, user_id, token, expires_at, created_at, updated_at
	`
	var created entities.RefreshToken
	err := r.db.QueryRowxContext(ctx, query, token.ID, token.UserID, token.Token, token.ExpiresAt).StructScan(&created)
	if err != nil {
		return entities.RefreshToken{}, pkgerrors.Wrap(err, "failed to create refresh token")
	}
	return created, nil
}

func (r *repository) GetRefreshTokenByToken(ctx context.Context, token string) (entities.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, updated_at
		FROM refresh_tokens
		WHERE token = $1
	`
	var result entities.RefreshToken
	err := r.db.GetContext(ctx, &result, query, token)
	if err != nil {
		return entities.RefreshToken{}, pkgerrors.Wrap(err, "failed to get refresh token")
	}
	return result, nil
}

func (r *repository) UpdateRefreshToken(ctx context.Context, tokenID uuid.UUID, newToken string, expiresAt time.Time) error {
	query := `
		UPDATE refresh_tokens
		SET token = $1, expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, newToken, expiresAt, tokenID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to update refresh token")
	}
	return nil
}

func (r *repository) DeleteRefreshToken(ctx context.Context, token string) error {
	query := `DELETE FROM refresh_tokens WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to delete refresh token")
	}
	return nil
}

func (r *repository) DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to delete refresh tokens by user id")
	}
	return nil
}
