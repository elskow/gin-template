package authorization

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/elskow/go-microservice-template/config"
	"github.com/elskow/go-microservice-template/pkg/database"
	"github.com/google/uuid"
)

type Permission struct {
	Name     string
	Resource string
	Action   string
}

type UserPermissions struct {
	Permissions []Permission
	LoadedAt    time.Time
}

type Authorizer struct {
	db            *database.TracedDB
	logger        *slog.Logger
	cache         map[string]*UserPermissions
	cacheMutex    sync.RWMutex
	cacheTTL      time.Duration
	enableCaching bool
}

func NewAuthorizer(db *database.TracedDB, logger *slog.Logger) *Authorizer {
	cfg := config.Get()
	return &Authorizer{
		db:            db,
		logger:        logger,
		cache:         make(map[string]*UserPermissions),
		cacheTTL:      cfg.CacheTTL(),
		enableCaching: true,
	}
}

func (a *Authorizer) SetCacheTTL(ttl time.Duration) {
	a.cacheTTL = ttl
}

func (a *Authorizer) DisableCache() {
	a.enableCaching = false
}

func (a *Authorizer) EnableCache() {
	a.enableCaching = true
}

func (a *Authorizer) HasPermission(ctx context.Context, userID string, permissionName string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	if a.enableCaching {
		if hasPermission, found := a.checkCache(userID, permissionName); found {
			return hasPermission, nil
		}
	}

	permissions, err := a.loadUserPermissions(ctx, uid)
	if err != nil {
		return false, fmt.Errorf("failed to load user permissions: %w", err)
	}

	if a.enableCaching {
		a.updateCache(userID, permissions)
	}

	for _, p := range permissions {
		if p.Name == permissionName {
			return true, nil
		}
	}

	return false, nil
}

func (a *Authorizer) HasAnyPermission(ctx context.Context, userID string, permissionNames []string) (bool, error) {
	for _, permissionName := range permissionNames {
		hasPermission, err := a.HasPermission(ctx, userID, permissionName)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}
	return false, nil
}

func (a *Authorizer) HasAllPermissions(ctx context.Context, userID string, permissionNames []string) (bool, error) {
	for _, permissionName := range permissionNames {
		hasPermission, err := a.HasPermission(ctx, userID, permissionName)
		if err != nil {
			return false, err
		}
		if !hasPermission {
			return false, nil
		}
	}
	return true, nil
}

func (a *Authorizer) HasRole(ctx context.Context, userID string, roleName string) (bool, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return false, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT EXISTS(
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1 AND r.name = $2
		)
	`

	var exists bool
	err = a.db.GetContext(ctx, &exists, query, uid, roleName)
	if err != nil {
		return false, fmt.Errorf("failed to check role: %w", err)
	}

	return exists, nil
}

func (a *Authorizer) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	var roles []string
	err = a.db.SelectContext(ctx, &roles, query, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	return roles, nil
}

func (a *Authorizer) AssignRole(ctx context.Context, userID string, roleName string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE name = $2
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err = a.db.ExecContext(ctx, query, uid, roleName)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	a.invalidateCache(userID)

	return nil
}

func (a *Authorizer) RemoveRole(ctx context.Context, userID string, roleName string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE name = $2)
	`

	_, err = a.db.ExecContext(ctx, query, uid, roleName)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	a.invalidateCache(userID)

	return nil
}

func (a *Authorizer) loadUserPermissions(ctx context.Context, userID uuid.UUID) ([]Permission, error) {
	query := `
		SELECT DISTINCT p.name, p.resource, p.action
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		JOIN roles r ON rp.role_id = r.id
		JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	var permissions []Permission
	err := a.db.SelectContext(ctx, &permissions, query, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return permissions, nil
}

func (a *Authorizer) checkCache(userID string, permissionName string) (bool, bool) {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

	userPerms, exists := a.cache[userID]
	if !exists {
		return false, false
	}

	if time.Since(userPerms.LoadedAt) > a.cacheTTL {
		return false, false
	}

	for _, p := range userPerms.Permissions {
		if p.Name == permissionName {
			return true, true
		}
	}

	return false, true
}

func (a *Authorizer) updateCache(userID string, permissions []Permission) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.cache[userID] = &UserPermissions{
		Permissions: permissions,
		LoadedAt:    time.Now(),
	}
}

func (a *Authorizer) invalidateCache(userID string) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	delete(a.cache, userID)
}

func (a *Authorizer) InvalidateAllCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	for k := range a.cache {
		delete(a.cache, k)
	}
}

func (a *Authorizer) StartCacheCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.cleanExpiredCache()
			}
		}
	}()
}

func (a *Authorizer) cleanExpiredCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	now := time.Now()
	for userID, userPerms := range a.cache {
		if now.Sub(userPerms.LoadedAt) > a.cacheTTL {
			delete(a.cache, userID)
		}
	}
}
