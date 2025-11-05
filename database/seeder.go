package database

import (
	"github.com/elskow/go-microservice-template/database/seeders/seeds"
	"github.com/jmoiron/sqlx"
)

func Seeder(db *sqlx.DB) error {
	if err := seeds.ListUserSeeder(db); err != nil {
		return err
	}

	return nil
}
