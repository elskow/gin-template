package providers

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/elskow/go-microservice-template/config"
	"github.com/elskow/go-microservice-template/modules/account/authorization"
	"github.com/elskow/go-microservice-template/modules/account/controller"
	"github.com/elskow/go-microservice-template/modules/account/repository"
	"github.com/elskow/go-microservice-template/modules/account/service"
	"github.com/elskow/go-microservice-template/pkg/apm"
	"github.com/elskow/go-microservice-template/pkg/constants"
	"github.com/elskow/go-microservice-template/pkg/database"
	"github.com/elskow/go-microservice-template/pkg/jwt"
	"github.com/elskow/go-microservice-template/pkg/logger"
	"github.com/elskow/go-microservice-template/pkg/telemetry"
	"github.com/samber/do"
)

func InitLogger(injector *do.Injector) {
	do.ProvideNamed(injector, "logger", func(i *do.Injector) (*slog.Logger, error) {
		log := logger.NewLogger("go-microservice-template", "1.0.0")
		logger.SetDefault(log)
		return log, nil
	})
}

func InitDatabase(injector *do.Injector) {
	do.ProvideNamed(injector, "db", func(i *do.Injector) (*database.TracedDB, error) {
		db := config.SetUpDatabaseConnection()
		return database.NewTracedDB(db), nil
	})
}

func InitTelemetry(injector *do.Injector) {
	do.ProvideNamed(injector, "telemetry", func(i *do.Injector) (*telemetry.Telemetry, error) {
		log := do.MustInvokeNamed[*slog.Logger](i, "logger")
		ctx := context.Background()
		return telemetry.InitTelemetry(ctx, "go-microservice-template", "1.0.0", log)
	})
}

func InitAPM(injector *do.Injector) {
	do.ProvideNamed(injector, "apm", func(i *do.Injector) (*apm.MetricsCollector, error) {
		log := do.MustInvokeNamed[*slog.Logger](i, "logger")
		return apm.NewMetricsCollector(log)
	})
}

func getCacheCleanupInterval() time.Duration {
	if intervalStr := os.Getenv(constants.EnvCacheCleanupInterval); intervalStr != "" {
		if minutes, err := strconv.Atoi(intervalStr); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return constants.DefaultCacheCleanupInterval
}

func InitAuthorizer(injector *do.Injector) {
	do.ProvideNamed(injector, "authorizer", func(i *do.Injector) (*authorization.Authorizer, error) {
		db := do.MustInvokeNamed[*database.TracedDB](i, "db")
		log := do.MustInvokeNamed[*slog.Logger](i, "logger")
		auth := authorization.NewAuthorizer(db, log)

		ctx := context.Background()
		cleanupInterval := getCacheCleanupInterval()
		auth.StartCacheCleanup(ctx, cleanupInterval)

		return auth, nil
	})
}

func RegisterDependencies(injector *do.Injector) {
	InitLogger(injector)
	InitDatabase(injector)
	InitTelemetry(injector)
	InitAPM(injector)
	InitAuthorizer(injector)

	do.ProvideNamed(injector, "jwt-service", func(i *do.Injector) (jwt.Service, error) {
		return jwt.NewService(), nil
	})

	do.ProvideNamed(injector, "repository", func(i *do.Injector) (repository.Repository, error) {
		db := do.MustInvokeNamed[*database.TracedDB](i, "db")
		return repository.NewRepository(db), nil
	})

	do.ProvideNamed(injector, "service", func(i *do.Injector) (service.Service, error) {
		repo := do.MustInvokeNamed[repository.Repository](i, "repository")
		jwtService := do.MustInvokeNamed[jwt.Service](i, "jwt-service")
		db := do.MustInvokeNamed[*database.TracedDB](i, "db")
		auth := do.MustInvokeNamed[*authorization.Authorizer](i, "authorizer")
		return service.NewService(repo, jwtService, db, auth), nil
	})

	do.ProvideNamed(injector, "controller", func(i *do.Injector) (*controller.Controller, error) {
		svc := do.MustInvokeNamed[service.Service](i, "service")
		log := do.MustInvokeNamed[*slog.Logger](i, "logger")
		auth := do.MustInvokeNamed[*authorization.Authorizer](i, "authorizer")
		return controller.NewController(svc, log, auth), nil
	})
}
