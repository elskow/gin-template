package entities

import (
	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Resource    string    `db:"resource" json:"resource"`
	Action      string    `db:"action" json:"action"`

	Timestamp
}
