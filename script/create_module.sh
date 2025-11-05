#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: ./create_module.sh <module_name>"
    exit 1
fi

MODULE_NAME=$1
GO_MODULE=$(grep "^module" go.mod | awk '{print $2}')

PASCAL_MODULE_NAME=$(echo "$MODULE_NAME" | awk -F'_' '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2)} }1' OFS='')

CAMEL_MODULE_NAME="$(tr '[:upper:]' '[:lower:]' <<< ${PASCAL_MODULE_NAME:0:1})${PASCAL_MODULE_NAME:1}"

echo "Creating module: $MODULE_NAME"

mkdir -p modules/$MODULE_NAME/controller
mkdir -p modules/$MODULE_NAME/service
mkdir -p modules/$MODULE_NAME/repository
mkdir -p modules/$MODULE_NAME/dto

cat > modules/$MODULE_NAME/controller/${MODULE_NAME}_controller.go << EOF
package controller

import (
	"log/slog"
	"net/http"

	"${GO_MODULE}/modules/$MODULE_NAME/dto"
	"${GO_MODULE}/modules/$MODULE_NAME/service"
	"${GO_MODULE}/pkg/constants"
	pkgerrors "${GO_MODULE}/pkg/errors"
	"${GO_MODULE}/pkg/response"
	"${GO_MODULE}/pkg/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Controller struct {
	service service.Service
	logger  *slog.Logger
}

func NewController(service service.Service, logger *slog.Logger) *Controller {
	return &Controller{
		service: service,
		logger:  logger,
	}
}

func (c *Controller) logError(ginCtx *gin.Context, msg string, err error) {
	spanCtx := trace.SpanContextFromContext(ginCtx.Request.Context())

	if spanCtx.IsValid() {
		c.logger.Error(msg,
			constants.AttrKeyTraceID, spanCtx.TraceID().String(),
			"error", err.Error(),
		)
	} else {
		c.logger.Error(msg,
			"error", err.Error(),
		)
	}
}

// Create handles POST /$MODULE_NAME
// TODO: implement your create logic
func (c *Controller) Create(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	var req dto.CreateRequest
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		c.logError(ginCtx, "invalid request body", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusBadRequest, response.Error[dto.Response](
			response.ErrCodeValidationFailed,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	// TODO: add validation attributes to span
	// span.SetAttributes(attribute.String("field_name", req.FieldName))

	result, err := c.service.Create(ctx, req)
	if err != nil {
		c.logError(ginCtx, "create ${MODULE_NAME} failed", err)
		pkgerrors.RecordError(span.Span, err)

		// TODO: handle different error types appropriately
		ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.Response](
			response.ErrCodeInternalServerError,
			constants.ErrMsgUnexpected,
		))
		return
	}

	span.SetAttributes(attribute.String("result_id", result.ID))
	ginCtx.JSON(http.StatusCreated, response.Success(result))
}

// GetByID handles GET /$MODULE_NAME/:id
// TODO: implement your get by ID logic
func (c *Controller) GetByID(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	id := ginCtx.Param("id")
	span.SetAttributes(attribute.String("${MODULE_NAME}_id", id))

	// TODO: implement get by ID
	result, err := c.service.GetByID(ctx, id)
	if err != nil {
		c.logError(ginCtx, "get ${MODULE_NAME} by id failed", err)
		pkgerrors.RecordError(span.Span, err)

		// TODO: handle different error types (not found, etc.)
		ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.Response](
			response.ErrCodeInternalServerError,
			constants.ErrMsgUnexpected,
		))
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(result))
}

// List handles GET /$MODULE_NAME
// TODO: implement your list logic
func (c *Controller) List(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	// TODO: implement pagination
	// page := ginCtx.DefaultQuery("page", "1")
	// limit := ginCtx.DefaultQuery("limit", "10")

	results, err := c.service.List(ctx)
	if err != nil {
		c.logError(ginCtx, "list ${MODULE_NAME} failed", err)
		pkgerrors.RecordError(span.Span, err)

		ginCtx.JSON(http.StatusInternalServerError, response.Error[[]dto.Response](
			response.ErrCodeInternalServerError,
			constants.ErrMsgUnexpected,
		))
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(results))
}

// Update handles PUT /$MODULE_NAME/:id
// TODO: implement your update logic
func (c *Controller) Update(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	id := ginCtx.Param("id")
	span.SetAttributes(attribute.String("${MODULE_NAME}_id", id))

	var req dto.UpdateRequest
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		c.logError(ginCtx, "invalid request body", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusBadRequest, response.Error[dto.Response](
			response.ErrCodeValidationFailed,
			"Invalid request body: "+err.Error(),
		))
		return
	}

	result, err := c.service.Update(ctx, id, req)
	if err != nil {
		c.logError(ginCtx, "update ${MODULE_NAME} failed", err)
		pkgerrors.RecordError(span.Span, err)

		// TODO: handle different error types appropriately
		ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.Response](
			response.ErrCodeInternalServerError,
			constants.ErrMsgUnexpected,
		))
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(result))
}

// Delete handles DELETE /$MODULE_NAME/:id
// TODO: implement your delete logic
func (c *Controller) Delete(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	id := ginCtx.Param("id")
	span.SetAttributes(attribute.String("${MODULE_NAME}_id", id))

	err := c.service.Delete(ctx, id)
	if err != nil {
		c.logError(ginCtx, "delete ${MODULE_NAME} failed", err)
		pkgerrors.RecordError(span.Span, err)

		// TODO: handle different error types appropriately
		ginCtx.JSON(http.StatusInternalServerError, response.Error[any](
			response.ErrCodeInternalServerError,
			constants.ErrMsgUnexpected,
		))
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(map[string]string{"message": "${MODULE_NAME} deleted successfully"}))
}
EOF

cat > modules/$MODULE_NAME/service/${MODULE_NAME}_service.go << EOF
package service

import (
	"context"

	"${GO_MODULE}/modules/$MODULE_NAME/dto"
	"${GO_MODULE}/modules/$MODULE_NAME/repository"
	"${GO_MODULE}/pkg/constants"
	"${GO_MODULE}/pkg/database"
	pkgerrors "${GO_MODULE}/pkg/errors"
	"${GO_MODULE}/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type Service interface {
	Create(ctx context.Context, req dto.CreateRequest) (dto.Response, error)
	GetByID(ctx context.Context, id string) (dto.Response, error)
	List(ctx context.Context) ([]dto.Response, error)
	Update(ctx context.Context, id string, req dto.UpdateRequest) (dto.Response, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo repository.Repository
	db   *database.TracedDB
}

func NewService(repo repository.Repository, db *database.TracedDB) Service {
	return &service{
		repo: repo,
		db:   db,
	}
}

// Create creates a new ${MODULE_NAME}
// TODO: implement your business logic here
func (s *service) Create(ctx context.Context, req dto.CreateRequest) (dto.Response, error) {
	ctx, span := tracing.Auto(ctx)
	defer span.End()

	// TODO: add span attributes for important fields
	// span.SetAttributes(attribute.String("field_name", req.FieldName))

	// TODO: implement validation logic
	// if err := s.validateCreateRequest(req); err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return dto.Response{}, err
	// }

	// TODO: implement create logic
	// Example:
	// entity := entities.${PASCAL_MODULE_NAME}{
	//     ID: uuid.New(),
	//     // ... map fields from req
	// }
	//
	// created, err := s.repo.Create(ctx, entity)
	// if err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return dto.Response{}, err
	// }
	//
	// return dto.Response{
	//     ID: created.ID.String(),
	//     // ... map fields from created entity
	// }, nil

	return dto.Response{}, nil
}

// GetByID retrieves a ${MODULE_NAME} by ID
// TODO: implement your get by ID logic
func (s *service) GetByID(ctx context.Context, id string) (dto.Response, error) {
	ctx, span := tracing.Auto(ctx, attribute.String("${MODULE_NAME}_id", id))
	defer span.End()

	// TODO: implement get by ID logic
	// Example:
	// entity, err := s.repo.GetByID(ctx, id)
	// if err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return dto.Response{}, err
	// }
	//
	// return dto.Response{
	//     ID: entity.ID.String(),
	//     // ... map fields from entity
	// }, nil

	return dto.Response{}, nil
}

// List retrieves all ${MODULE_NAME}s
// TODO: implement your list logic with pagination
func (s *service) List(ctx context.Context) ([]dto.Response, error) {
	ctx, span := tracing.Auto(ctx)
	defer span.End()

	// TODO: implement list logic
	// Example:
	// entities, err := s.repo.List(ctx)
	// if err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return nil, err
	// }
	//
	// results := make([]dto.Response, len(entities))
	// for i, entity := range entities {
	//     results[i] = dto.Response{
	//         ID: entity.ID.String(),
	//         // ... map fields from entity
	//     }
	// }
	//
	// span.SetAttributes(attribute.Int("count", len(results)))
	// return results, nil

	return []dto.Response{}, nil
}

// Update updates a ${MODULE_NAME}
// TODO: implement your update logic
func (s *service) Update(ctx context.Context, id string, req dto.UpdateRequest) (dto.Response, error) {
	ctx, span := tracing.Auto(ctx, attribute.String("${MODULE_NAME}_id", id))
	defer span.End()

	// TODO: implement update logic
	// Example:
	// // Check if exists
	// existing, err := s.repo.GetByID(ctx, id)
	// if err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return dto.Response{}, err
	// }
	//
	// // Update fields
	// // existing.Field = req.Field
	//
	// updated, err := s.repo.Update(ctx, existing)
	// if err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return dto.Response{}, err
	// }
	//
	// return dto.Response{
	//     ID: updated.ID.String(),
	//     // ... map fields from updated entity
	// }, nil

	return dto.Response{}, nil
}

// Delete deletes a ${MODULE_NAME}
// TODO: implement your delete logic
func (s *service) Delete(ctx context.Context, id string) error {
	ctx, span := tracing.Auto(ctx, attribute.String("${MODULE_NAME}_id", id))
	defer span.End()

	// TODO: implement delete logic
	// Example:
	// err := s.repo.Delete(ctx, id)
	// if err != nil {
	//     pkgerrors.RecordError(span.Span, err)
	//     return err
	// }
	//
	// return nil

	return nil
}
EOF

cat > modules/$MODULE_NAME/repository/${MODULE_NAME}_repository.go << EOF
package repository

import (
	"${GO_MODULE}/pkg/database"
)

type Repository interface {
	// TODO: define your repository methods here
	// Example methods:
	// Create(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error)
	// GetByID(ctx context.Context, id string) (entities.${PASCAL_MODULE_NAME}, error)
	// List(ctx context.Context) ([]entities.${PASCAL_MODULE_NAME}, error)
	// Update(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error)
	// Delete(ctx context.Context, id string) error
}

type repository struct {
	db *database.TracedDB
}

func NewRepository(db *database.TracedDB) Repository {
	return &repository{
		db: db,
	}
}

// Example implementation:
//
// import (
//     "context"
//     "${GO_MODULE}/pkg/tracing"
//     "go.opentelemetry.io/otel/attribute"
// )
//
// func (r *repository) Create(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error) {
// 	ctx, span := tracing.Auto(ctx)
// 	defer span.End()
//
// 	query := \`
// 		INSERT INTO ${MODULE_NAME}s (id, created_at, updated_at)
// 		VALUES (:id, :created_at, :updated_at)
// 		RETURNING *
// 	\`
//
// 	// Add span attributes for database operation
// 	span.SetAttributes(
// 		attribute.String("db.operation", "INSERT"),
// 		attribute.String("db.table", "${MODULE_NAME}s"),
// 	)
//
// 	stmt, err := r.db.PrepareNamedContext(ctx, query)
// 	if err != nil {
// 		pkgerrors.RecordError(span.Span, err)
// 		return entities.${PASCAL_MODULE_NAME}{}, err
// 	}
// 	defer stmt.Close()
//
// 	var result entities.${PASCAL_MODULE_NAME}
// 	err = stmt.GetContext(ctx, &result, entity)
// 	if err != nil {
// 		pkgerrors.RecordError(span.Span, err)
// 		return entities.${PASCAL_MODULE_NAME}{}, err
// 	}
//
// 	span.SetAttributes(attribute.String("${MODULE_NAME}_id", result.ID.String()))
// 	return result, nil
// }
//
// func (r *repository) GetByID(ctx context.Context, id string) (entities.${PASCAL_MODULE_NAME}, error) {
// 	ctx, span := tracing.Auto(ctx, attribute.String("${MODULE_NAME}_id", id))
// 	defer span.End()
//
// 	query := \`SELECT * FROM ${MODULE_NAME}s WHERE id = \$1\`
//
// 	span.SetAttributes(
// 		attribute.String("db.operation", "SELECT"),
// 		attribute.String("db.table", "${MODULE_NAME}s"),
// 	)
//
// 	var result entities.${PASCAL_MODULE_NAME}
// 	err := r.db.GetContext(ctx, &result, query, id)
// 	if err != nil {
// 		pkgerrors.RecordError(span.Span, err)
// 		return entities.${PASCAL_MODULE_NAME}{}, err
// 	}
//
// 	return result, nil
// }
EOF

cat > modules/$MODULE_NAME/dto/${MODULE_NAME}_dto.go << EOF
package dto

import "errors"

// Request DTOs
type (
	CreateRequest struct {
		// TODO: add your request fields here
		// Example:
		// Name        string \`json:"name" binding:"required"\`
		// Description string \`json:"description"\`
		// Status      string \`json:"status" binding:"required,oneof=active inactive"\`
	}

	UpdateRequest struct {
		// TODO: add your update request fields here
		// Example:
		// Name        *string \`json:"name,omitempty"\`
		// Description *string \`json:"description,omitempty"\`
		// Status      *string \`json:"status,omitempty" binding:"omitempty,oneof=active inactive"\`
	}
)

// Response DTOs
type (
	Response struct {
		ID string \`json:"id"\`
		// TODO: add your response fields here
		// Example:
		// Name        string    \`json:"name"\`
		// Description string    \`json:"description"\`
		// Status      string    \`json:"status"\`
		// CreatedAt   time.Time \`json:"created_at"\`
		// UpdatedAt   time.Time \`json:"updated_at"\`
	}
)

// Custom errors
var (
	// TODO: define your custom errors here
	// Example:
	// ErrNotFound          = errors.New("${MODULE_NAME} not found")
	// ErrAlreadyExists     = errors.New("${MODULE_NAME} already exists")
	// ErrInvalidStatus     = errors.New("invalid ${MODULE_NAME} status")
	// ErrUnauthorized      = errors.New("unauthorized to access ${MODULE_NAME}")

	ErrNotImplemented = errors.New("${MODULE_NAME} operation not implemented yet")
)
EOF

# Create Repository Tests
cat > modules/$MODULE_NAME/repository/${MODULE_NAME}_repository_test.go << EOF
package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"${GO_MODULE}/database/entities"
	"${GO_MODULE}/pkg/database"
	"github.com/DATA-DOG/go-sqlmock"
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

func TestRepository_Create(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	id := uuid.New()
	entity := entities.${PASCAL_MODULE_NAME}{
		ID: id,
		// TODO: Add your entity fields
	}

	// TODO: Update query to match your schema
	query := \`INSERT INTO ${MODULE_NAME}s (...) VALUES (...) RETURNING ...\`

	rows := sqlmock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectQuery(query).
		WillReturnRows(rows)

	created, err := repo.Create(ctx, entity)

	assert.NoError(t, err)
	assert.Equal(t, id, created.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	id := uuid.New()

	// TODO: Update query and columns to match your schema
	query := \`SELECT * FROM ${MODULE_NAME}s WHERE id = $1\`

	rows := sqlmock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectQuery(query).
		WithArgs(id).
		WillReturnRows(rows)

	result, err := repo.GetByID(ctx, id)

	assert.NoError(t, err)
	assert.Equal(t, id, result.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	db, mock, cleanup := setupMockDB(t)
	defer cleanup()

	repo := NewRepository(db)
	ctx := context.Background()

	id := uuid.New()
	query := \`SELECT * FROM ${MODULE_NAME}s WHERE id = $1\`

	mock.ExpectQuery(query).
		WithArgs(id).
		WillReturnError(sql.ErrNoRows)

	_, err := repo.GetByID(ctx, id)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TODO: Add more repository tests (Update, Delete, List, etc.)
EOF

# Create Service Tests
cat > modules/$MODULE_NAME/service/${MODULE_NAME}_service_test.go << EOF
package service

import (
	"context"
	"database/sql"
	"testing"

	"${GO_MODULE}/database/entities"
	"${GO_MODULE}/modules/${MODULE_NAME}/dto"
	"${GO_MODULE}/pkg/database"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Repository
type mockRepository struct {
	createFunc  func(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error)
	getByIDFunc func(ctx context.Context, id uuid.UUID) (entities.${PASCAL_MODULE_NAME}, error)
	updateFunc  func(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error)
	deleteFunc  func(ctx context.Context, id uuid.UUID) error
	listFunc    func(ctx context.Context) ([]entities.${PASCAL_MODULE_NAME}, error)
}

func (m *mockRepository) Create(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, entity)
	}
	return entity, nil
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (entities.${PASCAL_MODULE_NAME}, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return entities.${PASCAL_MODULE_NAME}{}, nil
}

func (m *mockRepository) Update(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, entity)
	}
	return entity, nil
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockRepository) List(ctx context.Context) ([]entities.${PASCAL_MODULE_NAME}, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []entities.${PASCAL_MODULE_NAME}{}, nil
}

func setupTestService(t *testing.T) (*service, *mockRepository) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tracedDB := &database.TracedDB{DB: sqlxDB}

	repo := &mockRepository{}

	svc := &service{
		repo: repo,
		db:   tracedDB,
	}

	return svc, repo
}

func TestService_Create_Success(t *testing.T) {
	svc, repo := setupTestService(t)
	ctx := context.Background()

	req := dto.CreateRequest{
		// TODO: Fill in your request fields
	}

	repo.createFunc = func(ctx context.Context, entity entities.${PASCAL_MODULE_NAME}) (entities.${PASCAL_MODULE_NAME}, error) {
		entity.ID = uuid.New()
		return entity, nil
	}

	resp, err := svc.Create(ctx, req)

	assert.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
}

func TestService_GetByID_Success(t *testing.T) {
	svc, repo := setupTestService(t)
	ctx := context.Background()

	id := uuid.New()
	expected := entities.${PASCAL_MODULE_NAME}{
		ID: id,
		// TODO: Fill in fields
	}

	repo.getByIDFunc = func(ctx context.Context, entityID uuid.UUID) (entities.${PASCAL_MODULE_NAME}, error) {
		return expected, nil
	}

	resp, err := svc.GetByID(ctx, id.String())

	assert.NoError(t, err)
	assert.Equal(t, id.String(), resp.ID)
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc, repo := setupTestService(t)
	ctx := context.Background()

	id := uuid.New()

	repo.getByIDFunc = func(ctx context.Context, entityID uuid.UUID) (entities.${PASCAL_MODULE_NAME}, error) {
		return entities.${PASCAL_MODULE_NAME}{}, sql.ErrNoRows
	}

	_, err := svc.GetByID(ctx, id.String())

	assert.Error(t, err)
}

// TODO: Add more service tests (Update, Delete, List, validation, error cases, etc.)
EOF

cat > modules/$MODULE_NAME/routes.go << EOF
package $MODULE_NAME

import (
	"log/slog"

	"${GO_MODULE}/modules/$MODULE_NAME/controller"
	"${GO_MODULE}/modules/$MODULE_NAME/repository"
	"${GO_MODULE}/modules/$MODULE_NAME/service"
	"${GO_MODULE}/pkg/database"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(server gin.IRouter, db *database.TracedDB, logger *slog.Logger) {
	repo := repository.NewRepository(db)
	svc := service.NewService(repo, db)
	ctrl := controller.NewController(svc, logger)

	// Base route group for this module
	${CAMEL_MODULE_NAME}Routes := server.Group("/api/v1/$MODULE_NAME")
	{
		// Public routes (no authentication required)
		${CAMEL_MODULE_NAME}Routes.GET("", ctrl.List)        // GET /api/v1/$MODULE_NAME
		${CAMEL_MODULE_NAME}Routes.GET("/:id", ctrl.GetByID) // GET /api/v1/$MODULE_NAME/:id

		// Protected routes (authentication required)
		// TODO: uncomment and adjust middleware as needed
		// authenticated := ${CAMEL_MODULE_NAME}Routes.Group("")
		// authenticated.Use(middlewares.AuthMiddleware())
		// {
		// 	authenticated.POST("", ctrl.Create)           // POST /api/v1/$MODULE_NAME
		// 	authenticated.PUT("/:id", ctrl.Update)        // PUT /api/v1/$MODULE_NAME/:id
		// 	authenticated.DELETE("/:id", ctrl.Delete)     // DELETE /api/v1/$MODULE_NAME/:id
		// }

		// For now, all routes are public (remove this in production)
		${CAMEL_MODULE_NAME}Routes.POST("", ctrl.Create)      // POST /api/v1/$MODULE_NAME
		${CAMEL_MODULE_NAME}Routes.PUT("/:id", ctrl.Update)   // PUT /api/v1/$MODULE_NAME/:id
		${CAMEL_MODULE_NAME}Routes.DELETE("/:id", ctrl.Delete) // DELETE /api/v1/$MODULE_NAME/:id
	}

	logger.Info("registered ${MODULE_NAME} routes",
		"base_path", "/api/v1/$MODULE_NAME",
		"endpoints", []string{
			"GET /api/v1/$MODULE_NAME",
			"GET /api/v1/$MODULE_NAME/:id",
			"POST /api/v1/$MODULE_NAME",
			"PUT /api/v1/$MODULE_NAME/:id",
			"DELETE /api/v1/$MODULE_NAME/:id",
		},
	)
}
EOF




echo ""
echo "Module '$MODULE_NAME' created successfully"
echo ""
echo "Files generated:"
echo "  modules/$MODULE_NAME/controller/${MODULE_NAME}_controller.go"
echo "  modules/$MODULE_NAME/service/${MODULE_NAME}_service.go"
echo "  modules/$MODULE_NAME/repository/${MODULE_NAME}_repository.go"
echo "  modules/$MODULE_NAME/repository/${MODULE_NAME}_repository_test.go"
echo "  modules/$MODULE_NAME/service/${MODULE_NAME}_service_test.go"
echo "  modules/$MODULE_NAME/dto/${MODULE_NAME}_dto.go"
echo "  modules/$MODULE_NAME/routes.go"
echo ""
echo "Next steps:"
echo ""
echo "1. Register routes in cmd/main.go:"
echo "   import \"${GO_MODULE}/modules/${MODULE_NAME}\""
echo "   ${MODULE_NAME}.RegisterRoutes(server, db, logger)"
echo ""
echo "2. Create database entity:"
echo "   database/entities/${MODULE_NAME}.go"
echo ""
echo "3. Create database migration:"
echo "   database/migrations/YYYYMMDDHHMMSS_create_${MODULE_NAME}s_table.sql"
echo ""
echo "4. Implement TODO items in generated files"
echo ""
echo "5. Run tests:"
echo "   go test ./modules/${MODULE_NAME}/... -v"
echo "   make test-coverage"
echo ""
