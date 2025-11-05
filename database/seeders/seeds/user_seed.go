package seeds

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/elskow/go-microservice-template/database/entities"
	"github.com/elskow/go-microservice-template/pkg/helpers"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func ListUserSeeder(db *sqlx.DB) error {
	jsonFile, err := os.Open("./database/seeders/json/users.json")
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	var listUser []entities.User
	if err := json.Unmarshal(jsonData, &listUser); err != nil {
		return err
	}

	for _, data := range listUser {
		var existingUser entities.User
		err := db.Get(&existingUser, "SELECT id FROM users WHERE email = $1", data.Email)

		if errors.Is(err, sql.ErrNoRows) {
			hashedPassword, err := helpers.HashPassword(data.Password)
			if err != nil {
				return err
			}

			if data.ID == uuid.Nil {
				data.ID = uuid.New()
			}

			query := `
				INSERT INTO users (id, name, email, password, created_at, updated_at)
				VALUES ($1, $2, $3, $4, NOW(), NOW())
			`
			_, err = db.Exec(query, data.ID, data.Name, data.Email, hashedPassword)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	return nil
}
