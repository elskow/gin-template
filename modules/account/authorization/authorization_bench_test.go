package authorization

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/elskow/go-microservice-template/pkg/database"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func BenchmarkAuthorizer_HasPermission_CacheHit(b *testing.B) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

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
		AddRow("write:users", "users", "write").
		AddRow("delete:users", "users", "delete")

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)

	// Warm up cache
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")
	}
}

func BenchmarkAuthorizer_HasPermission_CacheMiss(b *testing.B) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)
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

	for i := 0; i < b.N; i++ {
		rows := sqlmock.NewRows([]string{"name", "resource", "action"}).
			AddRow("read:users", "users", "read").
			AddRow("write:users", "users", "write").
			AddRow("delete:users", "users", "delete")
		mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")
	}
}

func BenchmarkAuthorizer_HasAnyPermission(b *testing.B) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

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

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)

	// Warm up cache
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	permissions := []string{"delete:users", "write:users", "update:users"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = authorizer.HasAnyPermission(ctx, userID.String(), permissions)
	}
}

func BenchmarkAuthorizer_HasAllPermissions(b *testing.B) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

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
		AddRow("write:users", "users", "write").
		AddRow("delete:users", "users", "delete")

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)

	// Warm up cache
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	permissions := []string{"read:users", "write:users"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = authorizer.HasAllPermissions(ctx, userID.String(), permissions)
	}
}

func BenchmarkAuthorizer_CacheInvalidation(b *testing.B) {
	mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

	userID := uuid.New()

	// Pre-populate cache with multiple users
	for i := 0; i < 100; i++ {
		authorizer.cache[uuid.New().String()] = &UserPermissions{
			Permissions: []Permission{
				{Name: "read:users", Resource: "users", Action: "read"},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authorizer.invalidateCache(userID.String())
	}
}

func BenchmarkAuthorizer_InvalidateAllCache(b *testing.B) {
	mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Pre-populate cache
		for j := 0; j < 100; j++ {
			authorizer.cache[uuid.New().String()] = &UserPermissions{
				Permissions: []Permission{
					{Name: "read:users", Resource: "users", Action: "read"},
				},
			}
		}
		b.StartTimer()

		authorizer.InvalidateAllCache()
	}
}

func BenchmarkAuthorizer_CleanExpiredCache(b *testing.B) {
	mockDB, _, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Pre-populate cache with mix of expired and non-expired entries
		expiredTime := time.Now().Add(-authorizer.cacheTTL * 2)
		for j := 0; j < 100; j++ {
			authorizer.cache[uuid.New().String()] = &UserPermissions{
				Permissions: []Permission{
					{Name: "read:users", Resource: "users", Action: "read"},
				},
				LoadedAt: expiredTime, // Expired
			}
		}
		b.StartTimer()

		authorizer.cleanExpiredCache()
	}
}

func BenchmarkAuthorizer_ConcurrentReads(b *testing.B) {
	mockDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := database.NewTracedDB(sqlxDB)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authorizer := NewAuthorizer(tracedDB, logger)

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

	mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)

	// Warm up cache
	_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = authorizer.HasPermission(ctx, userID.String(), "read:users")
		}
	})
}
