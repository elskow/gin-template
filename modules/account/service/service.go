package service

import (
	"context"
	"database/sql"

	"github.com/elskow/go-microservice-template/database/entities"
	"github.com/elskow/go-microservice-template/modules/account/authorization"
	"github.com/elskow/go-microservice-template/modules/account/dto"
	"github.com/elskow/go-microservice-template/modules/account/repository"
	"github.com/elskow/go-microservice-template/pkg/constants"
	"github.com/elskow/go-microservice-template/pkg/database"
	pkgerrors "github.com/elskow/go-microservice-template/pkg/errors"
	"github.com/elskow/go-microservice-template/pkg/helpers"
	"github.com/elskow/go-microservice-template/pkg/jwt"
	"github.com/elskow/go-microservice-template/pkg/tracing"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

type Service interface {
	Register(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.RefreshTokenResponse, error)
	Logout(ctx context.Context, userID string) error

	GetUserByID(ctx context.Context, userID string) (dto.UserResponse, error)
	UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest) (dto.UserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
}

type service struct {
	repo       repository.Repository
	jwtService jwt.Service
	db         *database.TracedDB
	authorizer *authorization.Authorizer
}

func NewService(repo repository.Repository, jwtService jwt.Service, db *database.TracedDB, authorizer *authorization.Authorizer) Service {
	return &service{
		repo:       repo,
		jwtService: jwtService,
		db:         db,
		authorizer: authorizer,
	}
}

func (s *service) Register(ctx context.Context, req dto.RegisterRequest) (dto.RegisterResponse, error) {
	ctx, span := tracing.Auto(ctx, attribute.String(constants.AttrKeyEmail, req.Email))
	defer span.End()

	_, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		pkgerrors.RecordError(span.Span, dto.ErrEmailAlreadyExists)
		return dto.RegisterResponse{}, dto.ErrEmailAlreadyExists
	}
	if !pkgerrors.Is(err, sql.ErrNoRows) {
		err = pkgerrors.Wrap(err, "failed to check existing email")
		pkgerrors.RecordError(span.Span, err)
		return dto.RegisterResponse{}, err
	}

	hashedPassword, err := helpers.HashPassword(req.Password)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to hash password")
		pkgerrors.RecordError(span.Span, err)
		return dto.RegisterResponse{}, err
	}

	user := entities.User{
		ID:       uuid.New(),
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
	}

	created, err := s.repo.CreateUser(ctx, user)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to create user")
		pkgerrors.RecordError(span.Span, err)
		return dto.RegisterResponse{}, err
	}

	if err := s.authorizer.AssignRole(ctx, created.ID.String(), "user"); err != nil {
		err = pkgerrors.Wrap(err, "failed to assign default role")
		pkgerrors.RecordError(span.Span, err)
	}

	accessToken, err := s.jwtService.GenerateAccessToken(created.ID.String(), "user")
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to generate access token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RegisterResponse{}, err
	}

	refreshTokenString, expiresAt, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to generate refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RegisterResponse{}, err
	}

	refreshToken := entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    created.ID,
		Token:     refreshTokenString,
		ExpiresAt: expiresAt,
	}

	_, err = s.repo.CreateRefreshToken(ctx, refreshToken)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to create refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RegisterResponse{}, err
	}

	return dto.RegisterResponse{
		User: dto.UserResponse{
			ID:    created.ID.String(),
			Name:  created.Name,
			Email: created.Email,
		},
		Token: dto.TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshTokenString,
		},
	}, nil
}

func (s *service) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	ctx, span := tracing.Auto(ctx, attribute.String(constants.AttrKeyEmail, req.Email))
	defer span.End()

	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if pkgerrors.Is(err, sql.ErrNoRows) {
			pkgerrors.RecordError(span.Span, dto.ErrInvalidCredentials)
			return dto.LoginResponse{}, dto.ErrInvalidCredentials
		}
		err = pkgerrors.Wrap(err, "failed to get user by email")
		pkgerrors.RecordError(span.Span, err)
		return dto.LoginResponse{}, err
	}

	if !helpers.CheckPassword(req.Password, user.Password) {
		pkgerrors.RecordError(span.Span, dto.ErrInvalidCredentials)
		return dto.LoginResponse{}, dto.ErrInvalidCredentials
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user.ID.String(), "user")
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to generate access token")
		pkgerrors.RecordError(span.Span, err)
		return dto.LoginResponse{}, err
	}

	refreshTokenString, expiresAt, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to generate refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.LoginResponse{}, err
	}

	refreshToken := entities.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     refreshTokenString,
		ExpiresAt: expiresAt,
	}

	_, err = s.repo.CreateRefreshToken(ctx, refreshToken)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to create refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.LoginResponse{}, err
	}

	return dto.LoginResponse{
		User: dto.UserResponse{
			ID:    user.ID.String(),
			Name:  user.Name,
			Email: user.Email,
		},
		Token: dto.TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshTokenString,
		},
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.RefreshTokenResponse, error) {
	ctx, span := tracing.Auto(ctx)
	defer span.End()

	refreshToken, err := s.repo.GetRefreshTokenByToken(ctx, req.RefreshToken)
	if err != nil {
		if pkgerrors.Is(err, sql.ErrNoRows) {
			pkgerrors.RecordError(span.Span, dto.ErrTokenNotFound)
			return dto.RefreshTokenResponse{}, dto.ErrTokenNotFound
		}
		err = pkgerrors.Wrap(err, "failed to get refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RefreshTokenResponse{}, err
	}

	if !refreshToken.IsValid() {
		pkgerrors.RecordError(span.Span, dto.ErrTokenNotFound)
		return dto.RefreshTokenResponse{}, dto.ErrTokenNotFound
	}

	accessToken, err := s.jwtService.GenerateAccessToken(refreshToken.UserID.String(), "user")
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to generate access token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RefreshTokenResponse{}, err
	}

	newRefreshTokenString, expiresAt, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to generate refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RefreshTokenResponse{}, err
	}

	err = s.repo.UpdateRefreshToken(ctx, refreshToken.ID, newRefreshTokenString, expiresAt)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to update refresh token")
		pkgerrors.RecordError(span.Span, err)
		return dto.RefreshTokenResponse{}, err
	}

	return dto.RefreshTokenResponse{
		Token: dto.TokenResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshTokenString,
		},
	}, nil
}

func (s *service) Logout(ctx context.Context, userID string) error {
	ctx, span := tracing.Auto(ctx, attribute.String(constants.AttrKeyUserID, userID))
	defer span.End()

	uid, err := uuid.Parse(userID)
	if err != nil {
		err = pkgerrors.Wrap(err, "invalid user id")
		pkgerrors.RecordError(span.Span, err)
		return err
	}

	err = s.repo.DeleteRefreshTokensByUserID(ctx, uid)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to delete refresh tokens")
		pkgerrors.RecordError(span.Span, err)
		return err
	}

	return nil
}

func (s *service) GetUserByID(ctx context.Context, userID string) (dto.UserResponse, error) {
	ctx, span := tracing.Auto(ctx, attribute.String(constants.AttrKeyUserID, userID))
	defer span.End()

	uid, err := uuid.Parse(userID)
	if err != nil {
		err = pkgerrors.Wrap(err, "invalid user id")
		pkgerrors.RecordError(span.Span, err)
		return dto.UserResponse{}, err
	}

	user, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		if pkgerrors.Is(err, sql.ErrNoRows) {
			pkgerrors.RecordError(span.Span, dto.ErrUserNotFound)
			return dto.UserResponse{}, dto.ErrUserNotFound
		}
		err = pkgerrors.Wrap(err, "failed to get user by id")
		pkgerrors.RecordError(span.Span, err)
		return dto.UserResponse{}, err
	}

	return dto.UserResponse{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}, nil
}

func (s *service) UpdateUser(ctx context.Context, userID string, req dto.UpdateUserRequest) (dto.UserResponse, error) {
	ctx, span := tracing.Auto(ctx, attribute.String(constants.AttrKeyUserID, userID))
	defer span.End()

	uid, err := uuid.Parse(userID)
	if err != nil {
		err = pkgerrors.Wrap(err, "invalid user id")
		pkgerrors.RecordError(span.Span, err)
		return dto.UserResponse{}, err
	}

	user, err := s.repo.GetUserByID(ctx, uid)
	if err != nil {
		if pkgerrors.Is(err, sql.ErrNoRows) {
			pkgerrors.RecordError(span.Span, dto.ErrUserNotFound)
			return dto.UserResponse{}, dto.ErrUserNotFound
		}
		err = pkgerrors.Wrap(err, "failed to get user by id")
		pkgerrors.RecordError(span.Span, err)
		return dto.UserResponse{}, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	updated, err := s.repo.UpdateUser(ctx, user)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to update user")
		pkgerrors.RecordError(span.Span, err)
		return dto.UserResponse{}, err
	}

	return dto.UserResponse{
		ID:    updated.ID.String(),
		Name:  updated.Name,
		Email: updated.Email,
	}, nil
}

func (s *service) DeleteUser(ctx context.Context, userID string) error {
	ctx, span := tracing.Auto(ctx, attribute.String(constants.AttrKeyUserID, userID))
	defer span.End()

	uid, err := uuid.Parse(userID)
	if err != nil {
		err = pkgerrors.Wrap(err, "invalid user id")
		pkgerrors.RecordError(span.Span, err)
		return err
	}

	err = s.repo.DeleteUser(ctx, uid)
	if err != nil {
		err = pkgerrors.Wrap(err, "failed to delete user")
		pkgerrors.RecordError(span.Span, err)
		return err
	}

	return nil
}
