package controller

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/elskow/go-microservice-template/config"
	"github.com/elskow/go-microservice-template/modules/account/authorization"
	"github.com/elskow/go-microservice-template/modules/account/dto"
	"github.com/elskow/go-microservice-template/modules/account/service"
	"github.com/elskow/go-microservice-template/pkg/constants"
	pkgerrors "github.com/elskow/go-microservice-template/pkg/errors"
	"github.com/elskow/go-microservice-template/pkg/response"
	"github.com/elskow/go-microservice-template/pkg/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Controller struct {
	service       service.Service
	logger        *slog.Logger
	authorizer    *authorization.Authorizer
	isDevelopment bool
}

func NewController(service service.Service, logger *slog.Logger, authorizer *authorization.Authorizer) *Controller {
	cfg := config.Get()
	return &Controller{
		service:       service,
		logger:        logger,
		authorizer:    authorizer,
		isDevelopment: cfg.IsDevelopment(),
	}
}

func buildErrorMessage(prefix, errMsg string) string {
	var builder strings.Builder
	builder.Grow(len(prefix) + 2 + len(errMsg))
	builder.WriteString(prefix)
	builder.WriteString(": ")
	builder.WriteString(errMsg)
	return builder.String()
}

func (c *Controller) logError(ginCtx *gin.Context, msg, userID, email string, err error) {
	spanCtx := trace.SpanContextFromContext(ginCtx.Request.Context())

	const attributePairSize = 2
	capacity := attributePairSize
	if userID != "" {
		capacity += attributePairSize
	}
	if email != "" {
		capacity += attributePairSize
	}

	attrs := make([]any, 0, capacity)

	if spanCtx.IsValid() {
		attrs = append(attrs, constants.AttrKeyTraceID, spanCtx.TraceID().String())
	}

	if userID != "" {
		attrs = append(attrs, constants.AttrKeyUserID, userID)
	}
	if email != "" {
		attrs = append(attrs, constants.AttrKeyEmail, email)
	}

	var errStr string
	if c.isDevelopment {
		errStr = fmt.Sprintf("%+v", err)
	} else {
		errStr = err.Error()
	}

	attrs = append(attrs, "error", errStr)

	c.logger.Error(msg, attrs...)
}

func (c *Controller) Register(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	var req dto.RegisterRequest
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		c.logError(ginCtx, "invalid request body", "", req.Email, err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusBadRequest, response.Error[dto.RegisterResponse](
			response.ErrCodeValidationFailed,
			buildErrorMessage("Invalid request body", err.Error()),
		))
		return
	}

	span.SetAttributes(attribute.String(constants.AttrKeyEmail, req.Email))

	result, err := c.service.Register(ctx, req)
	if err != nil {
		c.logError(ginCtx, "registration failed", "", req.Email, err)
		pkgerrors.RecordError(span.Span, err)
		switch {
		case pkgerrors.Is(err, dto.ErrEmailAlreadyExists):
			ginCtx.JSON(http.StatusConflict, response.Error[dto.RegisterResponse](
				response.ErrCodeConflict,
				err.Error(),
			))
		default:
			ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.RegisterResponse](
				response.ErrCodeInternalServerError,
				"An unexpected error occurred. Please try again later.",
			))
		}
		return
	}

	ginCtx.JSON(http.StatusCreated, response.Success(result))
}

func (c *Controller) Login(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	var req dto.LoginRequest
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		c.logError(ginCtx, "invalid request body", "", req.Email, err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusBadRequest, response.Error[dto.LoginResponse](
			response.ErrCodeValidationFailed,
			buildErrorMessage("Invalid request body", err.Error()),
		))
		return
	}

	span.SetAttributes(attribute.String(constants.AttrKeyEmail, req.Email))

	result, err := c.service.Login(ctx, req)
	if err != nil {
		c.logError(ginCtx, "login failed", "", req.Email, err)
		pkgerrors.RecordError(span.Span, err)
		switch {
		case pkgerrors.Is(err, dto.ErrInvalidCredentials):
			ginCtx.JSON(http.StatusUnauthorized, response.Error[dto.LoginResponse](
				response.ErrCodeInvalidCredentials,
				err.Error(),
			))
		default:
			ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.LoginResponse](
				response.ErrCodeInternalServerError,
				"An unexpected error occurred. Please try again later.",
			))
		}
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(result))
}

func (c *Controller) RefreshToken(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	var req dto.RefreshTokenRequest
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		c.logError(ginCtx, "invalid request body", "", "", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusBadRequest, response.Error[dto.RefreshTokenResponse](
			response.ErrCodeValidationFailed,
			buildErrorMessage("Invalid request body", err.Error()),
		))
		return
	}

	result, err := c.service.RefreshToken(ctx, req)
	if err != nil {
		c.logError(ginCtx, "token refresh failed", "", "", err)
		pkgerrors.RecordError(span.Span, err)
		switch {
		case pkgerrors.Is(err, dto.ErrTokenNotFound):
			ginCtx.JSON(http.StatusNotFound, response.Error[dto.RefreshTokenResponse](
				response.ErrCodeNotFound,
				err.Error(),
			))
		default:
			ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.RefreshTokenResponse](
				response.ErrCodeInternalServerError,
				"An unexpected error occurred. Please try again later.",
			))
		}
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(result))
}

func (c *Controller) Logout(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	userID := ginCtx.MustGet(constants.CtxKeyUserID).(string)
	span.SetAttributes(attribute.String(constants.AttrKeyUserID, userID))

	err := c.service.Logout(ctx, userID)
	if err != nil {
		c.logError(ginCtx, "logout failed", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusInternalServerError, response.Error[any](
			response.ErrCodeInternalServerError,
			"An unexpected error occurred. Please try again later.",
		))
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(map[string]string{"message": "logout successful"}))
}

func (c *Controller) Me(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	userID := ginCtx.MustGet(constants.CtxKeyUserID).(string)
	span.SetAttributes(attribute.String(constants.AttrKeyUserID, userID))

	result, err := c.service.GetUserByID(ctx, userID)
	if err != nil {
		c.logError(ginCtx, "get user failed", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		switch {
		case pkgerrors.Is(err, dto.ErrUserNotFound):
			ginCtx.JSON(http.StatusNotFound, response.Error[dto.UserResponse](
				response.ErrCodeNotFound,
				err.Error(),
			))
		default:
			ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.UserResponse](
				response.ErrCodeInternalServerError,
				"An unexpected error occurred. Please try again later.",
			))
		}
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(result))
}

func (c *Controller) UpdateUser(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	userID := ginCtx.MustGet(constants.CtxKeyUserID).(string)
	span.SetAttributes(attribute.String(constants.AttrKeyUserID, userID))

	var req dto.UpdateUserRequest
	if err := ginCtx.ShouldBindJSON(&req); err != nil {
		c.logError(ginCtx, "invalid request body", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusBadRequest, response.Error[dto.UserResponse](
			response.ErrCodeValidationFailed,
			buildErrorMessage("Invalid request body", err.Error()),
		))
		return
	}

	hasPermission, err := c.authorizer.HasPermission(ctx, userID, "user.update")
	if err != nil {
		c.logError(ginCtx, "permission check failed", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.UserResponse](
			response.ErrCodeInternalServerError,
			"Failed to verify permissions",
		))
		return
	}

	if !hasPermission {
		c.logError(ginCtx, "permission denied", userID, "", pkgerrors.New("permission denied"))
		ginCtx.JSON(http.StatusForbidden, response.Error[dto.UserResponse](
			response.ErrCodeForbidden,
			"You do not have permission to perform this action.",
		))
		return
	}

	result, err := c.service.UpdateUser(ctx, userID, req)
	if err != nil {
		c.logError(ginCtx, "update user failed", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		switch {
		case pkgerrors.Is(err, dto.ErrUserNotFound):
			ginCtx.JSON(http.StatusNotFound, response.Error[dto.UserResponse](
				response.ErrCodeNotFound,
				err.Error(),
			))
		default:
			ginCtx.JSON(http.StatusInternalServerError, response.Error[dto.UserResponse](
				response.ErrCodeInternalServerError,
				"An unexpected error occurred. Please try again later.",
			))
		}
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(result))
}

func (c *Controller) DeleteUser(ginCtx *gin.Context) {
	ctx, span := tracing.Auto(ginCtx.Request.Context())
	defer span.End()

	userID := ginCtx.MustGet(constants.CtxKeyUserID).(string)
	span.SetAttributes(attribute.String(constants.AttrKeyUserID, userID))

	hasPermission, err := c.authorizer.HasPermission(ctx, userID, "user.delete")
	if err != nil {
		c.logError(ginCtx, "permission check failed", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		ginCtx.JSON(http.StatusInternalServerError, response.Error[any](
			response.ErrCodeInternalServerError,
			"Failed to verify permissions",
		))
		return
	}

	if !hasPermission {
		c.logError(ginCtx, "permission denied", userID, "", pkgerrors.New("permission denied"))
		ginCtx.JSON(http.StatusForbidden, response.Error[any](
			response.ErrCodeForbidden,
			"You do not have permission to perform this action.",
		))
		return
	}

	err = c.service.DeleteUser(ctx, userID)
	if err != nil {
		c.logError(ginCtx, "delete user failed", userID, "", err)
		pkgerrors.RecordError(span.Span, err)
		switch {
		case pkgerrors.Is(err, dto.ErrUserNotFound):
			ginCtx.JSON(http.StatusNotFound, response.Error[any](
				response.ErrCodeNotFound,
				err.Error(),
			))
		default:
			ginCtx.JSON(http.StatusInternalServerError, response.Error[any](
				response.ErrCodeInternalServerError,
				"An unexpected error occurred. Please try again later.",
			))
		}
		return
	}

	ginCtx.JSON(http.StatusOK, response.Success(map[string]string{"message": "user deleted successfully"}))
}
