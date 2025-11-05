package dto

import "errors"

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenNotFound      = errors.New("refresh token not found")
)

type (
	RegisterRequest struct {
		Name     string `json:"name" binding:"required,min=2,max=100"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
	}

	RegisterResponse struct {
		User  UserResponse  `json:"user"`
		Token TokenResponse `json:"token"`
	}

	LoginRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	LoginResponse struct {
		User  UserResponse  `json:"user"`
		Token TokenResponse `json:"token"`
	}

	RefreshTokenRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	RefreshTokenResponse struct {
		Token TokenResponse `json:"token"`
	}

	TokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
)

type (
	UserResponse struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	UpdateUserRequest struct {
		Name  string `json:"name" binding:"omitempty,min=2,max=100"`
		Email string `json:"email" binding:"omitempty,email"`
	}
)
