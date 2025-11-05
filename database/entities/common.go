package entities

import (
	"time"
)

type Timestamp struct {
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Authorization struct {
	Token string `json:"token" binding:"required"`
	Role  string `json:"role" binding:"required,oneof=user admin"`
}
