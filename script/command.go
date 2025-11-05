package script

import (
	"log/slog"
	"os"

	"github.com/elskow/go-microservice-template/database"
	pkgDB "github.com/elskow/go-microservice-template/pkg/database"
	"github.com/samber/do"
)

func Commands(injector *do.Injector) bool {
	tracedDB := do.MustInvokeNamed[*pkgDB.TracedDB](injector, "db")
	db := tracedDB.DB
	logger := do.MustInvokeNamed[*slog.Logger](injector, "logger")

	migrate := false
	seed := false
	run := false

	for _, arg := range os.Args[1:] {
		if arg == "--migrate" {
			migrate = true
		}
		if arg == "--seed" {
			seed = true
		}
		if arg == "--run" {
			run = true
		}
	}

	if migrate {
		if err := database.Migrate(db); err != nil {
			logger.Error("migration failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migration completed successfully")
	}

	if seed {
		if err := database.Seeder(db); err != nil {
			logger.Error("seeder failed", "error", err)
			os.Exit(1)
		}
		logger.Info("seeder completed successfully")
	}

	if run {
		return true
	}

	return false
}
