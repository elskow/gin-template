package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/elskow/go-microservice-template/config"
	"github.com/golang-jwt/jwt/v4"
)

type Service interface {
	GenerateAccessToken(userID string, role string) (string, error)
	GenerateRefreshToken() (string, time.Time, error)
	ValidateToken(token string) (*jwt.Token, error)
	GetUserIDByToken(token string) (string, error)
}

type jwtCustomClaim struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type service struct {
	secretKey     string
	issuer        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewService() Service {
	cfg := config.Get()
	return &service{
		secretKey:     cfg.JWTSecret,
		issuer:        "Template",
		accessExpiry:  time.Minute * 15,
		refreshExpiry: time.Hour * 24 * 7,
	}
}

func (j *service) GenerateAccessToken(userID string, role string) (string, error) {
	claims := jwtCustomClaim{
		userID,
		role,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessExpiry)),
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tx, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tx, nil
}

func (j *service) GenerateRefreshToken() (string, time.Time, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	encodedLen := base64.StdEncoding.EncodedLen(32)
	buf := make([]byte, encodedLen)
	base64.StdEncoding.Encode(buf, b)
	refreshToken := string(buf)
	expiresAt := time.Now().Add(j.refreshExpiry)

	return refreshToken, expiresAt, nil
}

func (j *service) parseToken(t_ *jwt.Token) (any, error) {
	if _, ok := t_.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method %v", t_.Header["alg"])
	}
	return []byte(j.secretKey), nil
}

func (j *service) ValidateToken(token string) (*jwt.Token, error) {
	return jwt.Parse(token, j.parseToken)
}

func (j *service) GetUserIDByToken(token string) (string, error) {
	tToken, err := j.ValidateToken(token)
	if err != nil {
		return "", err
	}

	claims := tToken.Claims.(jwt.MapClaims)
	id := fmt.Sprintf("%v", claims["user_id"])
	return id, nil
}
