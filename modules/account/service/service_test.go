package service

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/elskow/go-microservice-template/database/entities"
	"github.com/elskow/go-microservice-template/modules/account/authorization"
	"github.com/elskow/go-microservice-template/modules/account/dto"
	"github.com/elskow/go-microservice-template/pkg/database"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Mock JWT Service
type mockJWTService struct {
	generateAccessTokenFunc  func(userID string, role string) (string, error)
	generateRefreshTokenFunc func() (string, time.Time, error)
	getUserIDByTokenFunc     func(token string) (string, error)
}

func (m *mockJWTService) GenerateAccessToken(userID string, role string) (string, error) {
	if m.generateAccessTokenFunc != nil {
		return m.generateAccessTokenFunc(userID, role)
	}
	return "mock_access_token", nil
}

func (m *mockJWTService) GenerateRefreshToken() (string, time.Time, error) {
	if m.generateRefreshTokenFunc != nil {
		return m.generateRefreshTokenFunc()
	}
	return "mock_refresh_token", time.Now().Add(7 * 24 * time.Hour), nil
}

func (m *mockJWTService) ValidateToken(token string) (*jwt.Token, error) {
	return &jwt.Token{Valid: true}, nil
}

func (m *mockJWTService) GetUserIDByToken(token string) (string, error) {
	if m.getUserIDByTokenFunc != nil {
		return m.getUserIDByTokenFunc(token)
	}
	return "user-id", nil
}

// Mock Repository
type mockRepository struct {
	createUserFunc                  func(ctx context.Context, user entities.User) (entities.User, error)
	getUserByIDFunc                 func(ctx context.Context, userID uuid.UUID) (entities.User, error)
	getUserByEmailFunc              func(ctx context.Context, email string) (entities.User, error)
	updateUserFunc                  func(ctx context.Context, user entities.User) (entities.User, error)
	deleteUserFunc                  func(ctx context.Context, userID uuid.UUID) error
	createRefreshTokenFunc          func(ctx context.Context, token entities.RefreshToken) (entities.RefreshToken, error)
	getRefreshTokenByTokenFunc      func(ctx context.Context, token string) (entities.RefreshToken, error)
	updateRefreshTokenFunc          func(ctx context.Context, tokenID uuid.UUID, newToken string, expiresAt time.Time) error
	deleteRefreshTokenFunc          func(ctx context.Context, token string) error
	deleteRefreshTokensByUserIDFunc func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockRepository) CreateUser(ctx context.Context, user entities.User) (entities.User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, user)
	}
	return user, nil
}

func (m *mockRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (entities.User, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(ctx, userID)
	}
	return entities.User{}, nil
}

func (m *mockRepository) GetUserByEmail(ctx context.Context, email string) (entities.User, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(ctx, email)
	}
	return entities.User{}, sql.ErrNoRows
}

func (m *mockRepository) UpdateUser(ctx context.Context, user entities.User) (entities.User, error) {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, user)
	}
	return user, nil
}

func (m *mockRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, userID)
	}
	return nil
}

func (m *mockRepository) CreateRefreshToken(ctx context.Context, token entities.RefreshToken) (entities.RefreshToken, error) {
	if m.createRefreshTokenFunc != nil {
		return m.createRefreshTokenFunc(ctx, token)
	}
	return token, nil
}

func (m *mockRepository) GetRefreshTokenByToken(ctx context.Context, token string) (entities.RefreshToken, error) {
	if m.getRefreshTokenByTokenFunc != nil {
		return m.getRefreshTokenByTokenFunc(ctx, token)
	}
	return entities.RefreshToken{}, nil
}

func (m *mockRepository) UpdateRefreshToken(ctx context.Context, tokenID uuid.UUID, newToken string, expiresAt time.Time) error {
	if m.updateRefreshTokenFunc != nil {
		return m.updateRefreshTokenFunc(ctx, tokenID, newToken, expiresAt)
	}
	return nil
}

func (m *mockRepository) DeleteRefreshToken(ctx context.Context, token string) error {
	if m.deleteRefreshTokenFunc != nil {
		return m.deleteRefreshTokenFunc(ctx, token)
	}
	return nil
}

func (m *mockRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteRefreshTokensByUserIDFunc != nil {
		return m.deleteRefreshTokensByUserIDFunc(ctx, userID)
	}
	return nil
}

func setupTestService(t *testing.T) (*service, *mockRepository, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := &database.TracedDB{DB: sqlxDB}

	// Create a real authorizer with mocked DB for role assignment
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	auth := authorization.NewAuthorizer(tracedDB, logger)

	repo := &mockRepository{}
	jwtSvc := &mockJWTService{}

	svc := &service{
		repo:       repo,
		jwtService: jwtSvc,
		db:         tracedDB,
		authorizer: auth,
	}

	return svc, repo, mock
}

func TestService_Register_Success(t *testing.T) {
	svc, repo, mock := setupTestService(t)
	ctx := context.Background()

	// Mock role assignment
	mock.ExpectExec(`INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := dto.RegisterRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}

	repo.getUserByEmailFunc = func(ctx context.Context, email string) (entities.User, error) {
		return entities.User{}, sql.ErrNoRows
	}

	repo.createUserFunc = func(ctx context.Context, user entities.User) (entities.User, error) {
		user.ID = uuid.New()
		return user, nil
	}

	repo.createRefreshTokenFunc = func(ctx context.Context, token entities.RefreshToken) (entities.RefreshToken, error) {
		return token, nil
	}

	resp, err := svc.Register(ctx, req)

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.User.ID)
	assert.Equal(t, req.Name, resp.User.Name)
	assert.Equal(t, req.Email, resp.User.Email)
	assert.NotEmpty(t, resp.Token.AccessToken)
	assert.NotEmpty(t, resp.Token.RefreshToken)
}

func TestService_Register_EmailAlreadyExists(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	req := dto.RegisterRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: "password123",
	}

	repo.getUserByEmailFunc = func(ctx context.Context, email string) (entities.User, error) {
		return entities.User{ID: uuid.New()}, nil
	}

	_, err := svc.Register(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, dto.ErrEmailAlreadyExists, err)
}

func TestService_Login_Success(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	password := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), 4)

	existingUser := entities.User{
		ID:       uuid.New(),
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: string(hashedPassword),
	}

	req := dto.LoginRequest{
		Email:    "john@example.com",
		Password: password,
	}

	repo.getUserByEmailFunc = func(ctx context.Context, email string) (entities.User, error) {
		return existingUser, nil
	}

	repo.createRefreshTokenFunc = func(ctx context.Context, token entities.RefreshToken) (entities.RefreshToken, error) {
		return token, nil
	}

	resp, err := svc.Login(ctx, req)

	assert.NoError(t, err)
	assert.Equal(t, existingUser.ID.String(), resp.User.ID)
	assert.Equal(t, existingUser.Name, resp.User.Name)
	assert.NotEmpty(t, resp.Token.AccessToken)
	assert.NotEmpty(t, resp.Token.RefreshToken)
}

func TestService_Login_InvalidCredentials_UserNotFound(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	req := dto.LoginRequest{
		Email:    "john@example.com",
		Password: "password123",
	}

	repo.getUserByEmailFunc = func(ctx context.Context, email string) (entities.User, error) {
		return entities.User{}, sql.ErrNoRows
	}

	_, err := svc.Login(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, dto.ErrInvalidCredentials, err)
}

func TestService_Login_InvalidCredentials_WrongPassword(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), 4)

	existingUser := entities.User{
		ID:       uuid.New(),
		Name:     "John Doe",
		Email:    "john@example.com",
		Password: string(hashedPassword),
	}

	req := dto.LoginRequest{
		Email:    "john@example.com",
		Password: "wrongpassword",
	}

	repo.getUserByEmailFunc = func(ctx context.Context, email string) (entities.User, error) {
		return existingUser, nil
	}

	_, err := svc.Login(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, dto.ErrInvalidCredentials, err)
}

func TestService_RefreshToken_Success(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	tokenString := "valid_refresh_token"
	refreshToken := entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     tokenString,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	req := dto.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	repo.getRefreshTokenByTokenFunc = func(ctx context.Context, token string) (entities.RefreshToken, error) {
		return refreshToken, nil
	}

	repo.updateRefreshTokenFunc = func(ctx context.Context, tokenID uuid.UUID, newToken string, expiresAt time.Time) error {
		return nil
	}

	resp, err := svc.RefreshToken(ctx, req)

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token.AccessToken)
	assert.NotEmpty(t, resp.Token.RefreshToken)
}

func TestService_RefreshToken_NotFound(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	req := dto.RefreshTokenRequest{
		RefreshToken: "invalid_token",
	}

	repo.getRefreshTokenByTokenFunc = func(ctx context.Context, token string) (entities.RefreshToken, error) {
		return entities.RefreshToken{}, sql.ErrNoRows
	}

	_, err := svc.RefreshToken(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, dto.ErrTokenNotFound, err)
}

func TestService_RefreshToken_Expired(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	tokenString := "expired_token"
	expiredToken := entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Token:     tokenString,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	req := dto.RefreshTokenRequest{
		RefreshToken: tokenString,
	}

	repo.getRefreshTokenByTokenFunc = func(ctx context.Context, token string) (entities.RefreshToken, error) {
		return expiredToken, nil
	}

	_, err := svc.RefreshToken(ctx, req)

	assert.Error(t, err)
	assert.Equal(t, dto.ErrTokenNotFound, err)
}

func TestService_Logout_Success(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	userID := uuid.New().String()

	repo.deleteRefreshTokensByUserIDFunc = func(ctx context.Context, uid uuid.UUID) error {
		return nil
	}

	err := svc.Logout(ctx, userID)

	assert.NoError(t, err)
}

func TestService_GetUserByID_Success(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	userID := uuid.New()
	expectedUser := entities.User{
		ID:    userID,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	repo.getUserByIDFunc = func(ctx context.Context, uid uuid.UUID) (entities.User, error) {
		return expectedUser, nil
	}

	resp, err := svc.GetUserByID(ctx, userID.String())

	assert.NoError(t, err)
	assert.Equal(t, expectedUser.ID.String(), resp.ID)
	assert.Equal(t, expectedUser.Name, resp.Name)
	assert.Equal(t, expectedUser.Email, resp.Email)
}

func TestService_GetUserByID_NotFound(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	userID := uuid.New()

	repo.getUserByIDFunc = func(ctx context.Context, uid uuid.UUID) (entities.User, error) {
		return entities.User{}, sql.ErrNoRows
	}

	_, err := svc.GetUserByID(ctx, userID.String())

	assert.Error(t, err)
	assert.Equal(t, dto.ErrUserNotFound, err)
}

func TestService_UpdateUser_Success(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	userID := uuid.New()
	existingUser := entities.User{
		ID:    userID,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	req := dto.UpdateUserRequest{
		Name:  "John Updated",
		Email: "john.updated@example.com",
	}

	repo.getUserByIDFunc = func(ctx context.Context, uid uuid.UUID) (entities.User, error) {
		return existingUser, nil
	}

	repo.updateUserFunc = func(ctx context.Context, user entities.User) (entities.User, error) {
		return user, nil
	}

	resp, err := svc.UpdateUser(ctx, userID.String(), req)

	assert.NoError(t, err)
	assert.Equal(t, req.Name, resp.Name)
	assert.Equal(t, req.Email, resp.Email)
}

func TestService_DeleteUser_Success(t *testing.T) {
	svc, repo, _ := setupTestService(t)
	ctx := context.Background()

	userID := uuid.New()

	repo.deleteUserFunc = func(ctx context.Context, uid uuid.UUID) error {
		return nil
	}

	err := svc.DeleteUser(ctx, userID.String())

	assert.NoError(t, err)
}
