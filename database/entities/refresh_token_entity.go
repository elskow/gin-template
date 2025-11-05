package entities

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Token     string    `db:"token" json:"token"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`

	Timestamp
}

func (rt *RefreshToken) IsValid() bool {
	return time.Now().Before(rt.ExpiresAt)
}

type RefreshTokenWithUser struct {
	RefreshToken
	User User `db:"user"`
}
