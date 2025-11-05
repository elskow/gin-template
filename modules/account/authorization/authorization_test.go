package authorization

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/elskow/go-microservice-template/pkg/constants"
	"github.com/elskow/go-microservice-template/pkg/database"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*database.TracedDB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	return tracedDB, mock
}

func setupAuthorizer(t *testing.T) (*Authorizer, sqlmock.Sqlmock, func()) {
	db, mock := setupMockDB(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(db, logger)

	cleanup := func() {
		db.DB.Close()
	}

	return authorizer, mock, cleanup
}

func TestNewAuthorizer(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.DB.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(db, logger)

	assert.NotNil(t, authorizer)
	assert.NotNil(t, authorizer.cache)
	assert.Equal(t, constants.DefaultCacheTTL, authorizer.cacheTTL)
	assert.True(t, authorizer.enableCaching)
}

func TestAuthorizer_SetCacheTTL(t *testing.T) {
	authorizer, _, cleanup := setupAuthorizer(t)
	defer cleanup()

	newTTL := 10 * time.Minute
	authorizer.SetCacheTTL(newTTL)

	assert.Equal(t, newTTL, authorizer.cacheTTL)
}

func TestAuthorizer_DisableEnableCache(t *testing.T) {
	authorizer, _, cleanup := setupAuthorizer(t)
	defer cleanup()

	// Initially enabled
	assert.True(t, authorizer.enableCaching)

	// Disable cache
	authorizer.DisableCache()
	assert.False(t, authorizer.enableCaching)

	// Enable cache
	authorizer.EnableCache()
	assert.True(t, authorizer.enableCaching)
}

func TestAuthorizer_HasPermission_InvalidUserID(t *testing.T) {
	authorizer, _, cleanup := setupAuthorizer(t)
	defer cleanup()

	ctx := context.Background()
	hasPermission, err := authorizer.HasPermission(ctx, "invalid-uuid", "read:users")

	assert.Error(t, err)
	assert.False(t, hasPermission)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestAuthorizer_HasPermission_WithPermission(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read").
		AddRow("write:users", "users", "write")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	hasPermission, err := authorizer.HasPermission(ctx, userID.String(), "read:users")

	assert.NoError(t, err)
	assert.True(t, hasPermission)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasPermission_WithoutPermission(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	hasPermission, err := authorizer.HasPermission(ctx, userID.String(), "delete:users")

	assert.NoError(t, err)
	assert.False(t, hasPermission)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasPermission_NoPermissions(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"})

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	hasPermission, err := authorizer.HasPermission(ctx, userID.String(), "read:users")

	assert.NoError(t, err)
	assert.False(t, hasPermission)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasPermission_CacheHit(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	// First call - cache miss
	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	hasPermission1, err := authorizer.HasPermission(ctx, userID.String(), "read:users")
	assert.NoError(t, err)
	assert.True(t, hasPermission1)

	// Second call - cache hit, no DB query expected
	hasPermission2, err := authorizer.HasPermission(ctx, userID.String(), "read:users")
	assert.NoError(t, err)
	assert.True(t, hasPermission2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasPermission_CacheDisabled(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	authorizer.DisableCache()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows1 := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")
	rows2 := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	// Both calls should hit the database
	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows1)
	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows2)

	hasPermission1, err := authorizer.HasPermission(ctx, userID.String(), "read:users")
	assert.NoError(t, err)
	assert.True(t, hasPermission1)

	hasPermission2, err := authorizer.HasPermission(ctx, userID.String(), "read:users")
	assert.NoError(t, err)
	assert.True(t, hasPermission2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasAnyPermission(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	permissions := []string{"delete:users", "read:users", "write:users"}
	hasAny, err := authorizer.HasAnyPermission(ctx, userID.String(), permissions)

	assert.NoError(t, err)
	assert.True(t, hasAny)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasAnyPermission_NoMatch(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	permissions := []string{"delete:users", "write:users"}
	hasAny, err := authorizer.HasAnyPermission(ctx, userID.String(), permissions)

	assert.NoError(t, err)
	assert.False(t, hasAny)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasAllPermissions(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read").
		AddRow("write:users", "users", "write")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	permissions := []string{"read:users", "write:users"}
	hasAll, err := authorizer.HasAllPermissions(ctx, userID.String(), permissions)

	assert.NoError(t, err)
	assert.True(t, hasAll)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasAllPermissions_MissingOne(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	permissions := []string{"read:users", "write:users", "delete:users"}
	hasAll, err := authorizer.HasAllPermissions(ctx, userID.String(), permissions)

	assert.NoError(t, err)
	assert.False(t, hasAll)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasRole(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT EXISTS(
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1 AND r.name = $2
		)
	`

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

	mock.ExpectQuery(query).
		WithArgs(userID, "admin").
		WillReturnRows(rows)

	hasRole, err := authorizer.HasRole(ctx, userID.String(), "admin")

	assert.NoError(t, err)
	assert.True(t, hasRole)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_HasRole_InvalidUserID(t *testing.T) {
	authorizer, _, cleanup := setupAuthorizer(t)
	defer cleanup()

	ctx := context.Background()
	hasRole, err := authorizer.HasRole(ctx, "invalid-uuid", "admin")

	assert.Error(t, err)
	assert.False(t, hasRole)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestAuthorizer_GetUserRoles(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	rows := sqlmock.NewRows([]string{"name"}).
		AddRow("admin").
		AddRow("user")

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnRows(rows)

	roles, err := authorizer.GetUserRoles(ctx, userID.String())

	assert.NoError(t, err)
	assert.Equal(t, []string{"admin", "user"}, roles)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_GetUserRoles_InvalidUserID(t *testing.T) {
	authorizer, _, cleanup := setupAuthorizer(t)
	defer cleanup()

	ctx := context.Background()
	roles, err := authorizer.GetUserRoles(ctx, "invalid-uuid")

	assert.Error(t, err)
	assert.Nil(t, roles)
	assert.Contains(t, err.Error(), "invalid user ID")
}

func TestAuthorizer_AssignRole(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	mock.ExpectExec(query).
		WithArgs(userID, "admin").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := authorizer.AssignRole(ctx, userID.String(), "admin")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_AssignRole_InvalidatesCache(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	// First, populate cache
	query1 := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`
	rows1 := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query1).WithArgs(userID).WillReturnRows(rows1)
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	// Assign role
	query2 := `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	mock.ExpectExec(query2).
		WithArgs(userID, "admin").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := authorizer.AssignRole(ctx, userID.String(), "admin")
	assert.NoError(t, err)

	// Cache should be invalidated
	authorizer.cacheMutex.RLock()
	_, exists := authorizer.cache[userID.String()]
	authorizer.cacheMutex.RUnlock()
	assert.False(t, exists)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_RemoveRole(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE name = $2)
	`

	mock.ExpectExec(query).
		WithArgs(userID, "admin").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := authorizer.RemoveRole(ctx, userID.String(), "admin")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_InvalidateAllCache(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	// Populate cache
	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`
	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	// Verify cache has entry
	authorizer.cacheMutex.RLock()
	lenBefore := len(authorizer.cache)
	authorizer.cacheMutex.RUnlock()
	assert.Equal(t, 1, lenBefore)

	// Invalidate all
	authorizer.InvalidateAllCache()

	// Verify cache is empty
	authorizer.cacheMutex.RLock()
	lenAfter := len(authorizer.cache)
	authorizer.cacheMutex.RUnlock()
	assert.Equal(t, 0, lenAfter)
}

func TestAuthorizer_CacheExpiry(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	// Set a very short TTL
	authorizer.SetCacheTTL(100 * time.Millisecond)

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	rows1 := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")
	rows2 := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	// First call
	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows1)
	hasPermission1, err := authorizer.HasPermission(ctx, userID.String(), "read:users")
	assert.NoError(t, err)
	assert.True(t, hasPermission1)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second call should hit DB again
	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows2)
	hasPermission2, err := authorizer.HasPermission(ctx, userID.String(), "read:users")
	assert.NoError(t, err)
	assert.True(t, hasPermission2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthorizer_CleanExpiredCache(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	// Set a very short TTL
	authorizer.SetCacheTTL(100 * time.Millisecond)

	userID := uuid.New()
	ctx := context.Background()

	// Populate cache
	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`
	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	// Verify cache has entry
	authorizer.cacheMutex.RLock()
	lenBefore := len(authorizer.cache)
	authorizer.cacheMutex.RUnlock()
	assert.Equal(t, 1, lenBefore)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Clean expired cache
	authorizer.cleanExpiredCache()

	// Verify cache is empty
	authorizer.cacheMutex.RLock()
	lenAfter := len(authorizer.cache)
	authorizer.cacheMutex.RUnlock()
	assert.Equal(t, 0, lenAfter)
}

func TestAuthorizer_StartCacheCleanup(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	// Set a very short TTL
	authorizer.SetCacheTTL(50 * time.Millisecond)

	userID := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Populate cache
	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`
	rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
		AddRow("read:users", "users", "read")

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	// Start cache cleanup with very short interval
	authorizer.StartCacheCleanup(ctx, 100*time.Millisecond)

	// Verify cache has entry
	authorizer.cacheMutex.RLock()
	lenBefore := len(authorizer.cache)
	authorizer.cacheMutex.RUnlock()
	assert.Equal(t, 1, lenBefore)

	// Wait for cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Cache should be cleaned
	authorizer.cacheMutex.RLock()
	lenAfter := len(authorizer.cache)
	authorizer.cacheMutex.RUnlock()
	assert.Equal(t, 0, lenAfter)
}

func TestAuthorizer_DatabaseError(t *testing.T) {
	authorizer, mock, cleanup := setupAuthorizer(t)
	defer cleanup()

	userID := uuid.New()
	ctx := context.Background()

	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	mock.ExpectQuery(query).
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	hasPermission, err := authorizer.HasPermission(ctx, userID.String(), "read:users")

	assert.Error(t, err)
	assert.False(t, hasPermission)
	assert.Contains(t, err.Error(), "failed to load user permissions")
	assert.NoError(t, mock.ExpectationsWereMet())
}
