package config

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func SetUpDatabaseConnection() *sqlx.DB {
	cfg := Get()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %v (dsn: host=%s port=%s user=%s dbname=%s)",
			err, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBName))
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)

	if connMaxLifetime := cfg.DBConnMaxLifetime(); connMaxLifetime > 0 {
		db.SetConnMaxLifetime(connMaxLifetime)
	}

	if connMaxIdleTime := cfg.DBConnMaxIdleTime(); connMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(connMaxIdleTime)
	}

	return db
}
