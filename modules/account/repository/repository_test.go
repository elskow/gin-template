package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/elskow/go-microservice-template/database/entities"
	"github.com/elskow/go-microservice-template/pkg/database"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*database.TracedDB, sqlmock.Sqlmock, func()) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := &database.TracedDB{DB: sqlxDB}

	cleanup := func() {
		mockDB.Close()
	}

	return tracedDB, mock, cleanup
}

func TestNewRepository(t *testing.T) {
	db, _, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)

	assert.NotNil(t, repo)
	assert.Implements(t, (*Repository)(nil), repo)
}

func TestRepository_CreateUser(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	user := entities.User{
		ID:       userID,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	query := `
		INSERT INTO users (id, name, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, email, password, created_at, updated_at
	`

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "created_at", "updated_at"}).
		AddRow(user.ID, user.Name, user.Email, user.Password, time.Now(), time.Now())

	mock.ExpectQuery(query).
		WithArgs(user.ID, user.Name, user.Email, user.Password).
		WillReturnRows(rows)

	created, err := repo.CreateUser(ctx, user)

	assert.NoError(t, err)
	assert.Equal(t, user.ID, created.ID)
	assert.Equal(t, user.Name, created.Name)
	assert.Equal(t, user.Email, created.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateUser_Error(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	user := entities.User{
		ID:       uuid.New(),
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
	}

	query := `
		INSERT INTO users (id, name, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, name, email, password, created_at, updated_at
	`

	mock.ExpectQuery(query).
		WithArgs(user.ID, user.Name, user.Email, user.Password).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.CreateUser(ctx, user)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create user")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	now := time.Now()
	expectedUser := entities.User{
		ID:       userID,
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "hashedpassword",
		Timestamp: entities.Timestamp{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	query := `SELECT id, name, email, password, created_at, updated_at FROM users WHERE id = $1`

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "created_at", "updated_at"}).
		AddRow(expectedUser.ID, expectedUser.Name, expectedUser.Email, expectedUser.Password,
			expectedUser.Timestamp.CreatedAt, expectedUser.Timestamp.UpdatedAt)

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	user, err := repo.GetUserByID(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.ID, user.ID)
	assert.Equal(t, expectedUser.Name, user.Name)
	assert.Equal(t, expectedUser.Email, user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByID_NotFound(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	query := `SELECT id, name, email, password, created_at, updated_at FROM users WHERE id = $1`

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetUserByID(ctx, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user by id")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetUserByEmail(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	email := "john@example.com"
	now := time.Now()
	expectedUser := entities.User{
		ID:       uuid.New(),
		Name:     "John Doe",
		Email:    email,
		Password: "hashedpassword",
		Timestamp: entities.Timestamp{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	query := `SELECT id, name, email, password, created_at, updated_at FROM users WHERE email = $1`

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "created_at", "updated_at"}).
		AddRow(expectedUser.ID, expectedUser.Name, expectedUser.Email, expectedUser.Password,
			expectedUser.Timestamp.CreatedAt, expectedUser.Timestamp.UpdatedAt)

	mock.ExpectQuery(query).
		WithArgs(email).
		WillReturnRows(rows)

	user, err := repo.GetUserByEmail(ctx, email)

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.Email, user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateUser(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	user := entities.User{
		ID:    uuid.New(),
		Name:  "John Updated",
		Email: "john.updated@example.com",
	}

	query := `
		UPDATE users SET name = $1, email = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, email, password, created_at, updated_at
	`

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "created_at", "updated_at"}).
		AddRow(user.ID, user.Name, user.Email, "hashedpassword", time.Now(), time.Now())

	mock.ExpectQuery(query).
		WithArgs(user.Name, user.Email, user.ID).
		WillReturnRows(rows)

	updated, err := repo.UpdateUser(ctx, user)

	assert.NoError(t, err)
	assert.Equal(t, user.Name, updated.Name)
	assert.Equal(t, user.Email, updated.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUser(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	query := `DELETE FROM users WHERE id = $1`

	mock.ExpectExec(query).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteUser(ctx, userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteUser_NotFound(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	query := `DELETE FROM users WHERE id = $1`

	mock.ExpectExec(query).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeleteUser(ctx, userID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_CreateRefreshToken(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	token := entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     "refresh_token_string",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, user_id, token, expires_at, created_at, updated_at
	`

	rows := sqlmock.NewRows([]string{"id", "user_id", "token", "expires_at", "created_at", "updated_at"}).
		AddRow(token.ID, token.UserID, token.Token, token.ExpiresAt, time.Now(), time.Now())

	mock.ExpectQuery(query).
		WithArgs(token.ID, token.UserID, token.Token, token.ExpiresAt).
		WillReturnRows(rows)

	created, err := repo.CreateRefreshToken(ctx, token)

	assert.NoError(t, err)
	assert.Equal(t, token.ID, created.ID)
	assert.Equal(t, token.Token, created.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetRefreshTokenByToken(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	tokenString := "refresh_token_string"
	now := time.Now()
	expectedToken := entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     tokenString,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
		Timestamp: entities.Timestamp{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	query := `
		SELECT id, user_id, token, expires_at, created_at, updated_at
		FROM refresh_tokens
		WHERE token = $1
	`

	rows := sqlmock.NewRows([]string{"id", "user_id", "token", "expires_at", "created_at", "updated_at"}).
		AddRow(expectedToken.ID, expectedToken.UserID, expectedToken.Token, expectedToken.ExpiresAt,
			expectedToken.Timestamp.CreatedAt, expectedToken.Timestamp.UpdatedAt)

	mock.ExpectQuery(query).
		WithArgs(tokenString).
		WillReturnRows(rows)

	token, err := repo.GetRefreshTokenByToken(ctx, tokenString)

	assert.NoError(t, err)
	assert.Equal(t, expectedToken.Token, token.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateRefreshToken(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	tokenID := uuid.New()
	newToken := "new_refresh_token"
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	query := `
		UPDATE refresh_tokens
		SET token = $1, expires_at = $2, updated_at = NOW()
		WHERE id = $3
	`

	mock.ExpectExec(query).
		WithArgs(newToken, expiresAt, tokenID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateRefreshToken(ctx, tokenID, newToken, expiresAt)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteRefreshToken(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	tokenString := "refresh_token_to_delete"
	query := `DELETE FROM refresh_tokens WHERE token = $1`

	mock.ExpectExec(query).
		WithArgs(tokenString).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteRefreshToken(ctx, tokenString)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_DeleteRefreshTokensByUserID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`

	mock.ExpectExec(query).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := repo.DeleteRefreshTokensByUserID(ctx, userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
