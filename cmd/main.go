package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/elskow/go-microservice-template/config"
	"github.com/elskow/go-microservice-template/middlewares"
	"github.com/elskow/go-microservice-template/modules/account"
	"github.com/elskow/go-microservice-template/pkg/apm"
	"github.com/elskow/go-microservice-template/pkg/constants"
	pkgLogger "github.com/elskow/go-microservice-template/pkg/logger"
	"github.com/elskow/go-microservice-template/pkg/telemetry"
	"github.com/elskow/go-microservice-template/providers"
	"github.com/elskow/go-microservice-template/script"
	"github.com/grafana/pyroscope-go"
	"github.com/samber/do"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func args(injector *do.Injector) bool {
	if len(os.Args) > 1 {
		flag := script.Commands(injector)
		return flag
	}

	return true
}

func getBlacklistPaths(cfg *config.Config) []string {
	if cfg.LogBlacklistPaths == "" {
		return nil
	}

	paths := strings.Split(cfg.LogBlacklistPaths, ",")
	for i := range paths {
		paths[i] = strings.TrimSpace(paths[i])
	}
	return paths
}

func isPathBlacklisted(path string, blacklist []string) bool {
	for _, blacklisted := range blacklist {
		if blacklisted == path {
			return true
		}
	}
	return false
}

const (
	defaultPort   = "8888"
	localhostEnv  = "localhost"
	allInterfaces = "0.0.0.0"
)

func run(server *gin.Engine, logger *slog.Logger, cfg *config.Config) {
	var serve string
	if cfg.IsLocalhost() {
		serve = allInterfaces + ":" + cfg.Port
	} else {
		serve = ":" + cfg.Port
	}

	logger.Info("server starting", "address", serve)

	if err := server.Run(serve); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func main() {
	var (
		injector = do.New()
	)

	// Load configuration first
	cfg := config.Load()

	providers.RegisterDependencies(injector)

	logger := do.MustInvokeNamed[*slog.Logger](injector, "logger")
	tel := do.MustInvokeNamed[*telemetry.Telemetry](injector, "telemetry")
	apmCollector := do.MustInvokeNamed[*apm.MetricsCollector](injector, "apm")

	const (
		migrateFlag = "--migrate"
		seedFlag    = "--seed"
	)

	argsStr := strings.Join(os.Args, " ")
	isMigrationCommand := len(os.Args) > 1 && (strings.Contains(argsStr, migrateFlag) || strings.Contains(argsStr, seedFlag))

	if cfg.EnableProfiling && !isMigrationCommand {
		profiler, err := pyroscope.Start(pyroscope.Config{
			ApplicationName: cfg.AppName,
			ServerAddress:   cfg.PyroscopeServerAddr,
			Logger:          pyroscope.StandardLogger,
			Tags: map[string]string{
				"env":     cfg.AppEnv,
				"version": cfg.AppVersion,
			},
			ProfileTypes: []pyroscope.ProfileType{
				pyroscope.ProfileCPU,
				pyroscope.ProfileAllocObjects,
				pyroscope.ProfileAllocSpace,
				pyroscope.ProfileInuseObjects,
				pyroscope.ProfileInuseSpace,
				pyroscope.ProfileGoroutines,
				pyroscope.ProfileMutexCount,
				pyroscope.ProfileMutexDuration,
				pyroscope.ProfileBlockCount,
				pyroscope.ProfileBlockDuration,
			},
		})
		if err != nil {
			logger.Warn("failed to start pyroscope profiler", "error", err)
		} else {
			logger.Info("pyroscope profiler started", "server", cfg.PyroscopeServerAddr)
			defer profiler.Stop()
		}
	} else {
		logger.Info("profiling disabled")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultShutdownTimeout)
		defer cancel()

		if err := pkgLogger.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown logger", "error", err)
		}

		if err := tel.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown telemetry", "error", err)
		}
	}()

	if !args(injector) {
		return
	}

	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	server := gin.New()
	server.Use(gin.Recovery())
	server.Use(middlewares.RequestIDMiddleware())

	blacklistPaths := getBlacklistPaths(cfg)

	server.Use(otelgin.Middleware(
		cfg.AppName,
		otelgin.WithGinFilter(func(c *gin.Context) bool {
			return !isPathBlacklisted(c.Request.URL.Path, blacklistPaths)
		}),
	))

	server.Use(middlewares.SlogMiddleware(logger))
	server.Use(middlewares.CORSMiddleware())

	server.Use(middlewares.HTTPMetricsMiddleware(apmCollector))

	server.GET("/metrics", gin.WrapH(promhttp.Handler()))

	const (
		statusOK       = 200
		statusNotFound = 404
	)

	server.GET("/health", func(c *gin.Context) {
		c.JSON(statusOK, gin.H{"status": "ok"})
	})

	api := server.Group("/api")
	{
		account.RegisterRoutes(api, injector)
	}

	server.NoRoute(func(c *gin.Context) {
		c.String(statusNotFound, "")
	})

	go run(server, logger, cfg)

	<-ctx.Done()
	logger.Info("shutting down server")
}
