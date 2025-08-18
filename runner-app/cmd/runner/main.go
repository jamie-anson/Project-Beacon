package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jamie-anson/project-beacon-runner/internal/api"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/db"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/worker"
	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
)

func main() {
	// Initialize structured logging
	logger := logging.Init()
	logger.Info().Msg("logger initialized")
	// Load config
	cfg := config.Load()

	// Initialize Gin router
	r := gin.Default()

	// Metrics middleware and endpoint
	r.Use(metrics.GinMiddleware())
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	// Initialize database (non-fatal if not available; db package handles fallback)
	database, err := db.Initialize(cfg.DatabaseURL)
	if err != nil {
		logger.Warn().Err(err).Msg("database initialization warning")
	}

	// Setup routes with DB dependency
	api.SetupRoutes(r, database)

	// Setup server
	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: r,
	}

	// Redis client and background processes
	workerCtx, workerCancel := context.WithCancel(context.Background())
	q := queue.MustNewFromEnv()

	// Start structured workers if DB is available
	if database != nil && database.DB != nil {
		// Initialize Golem service for worker
		apiKey := os.Getenv("GOLEM_API_KEY")
		if apiKey == "" {
			apiKey = "test-key"
		}
		network := os.Getenv("GOLEM_NETWORK")
		if network == "" {
			network = "testnet"
		}
		gsvc := golem.NewService(apiKey, network)

		// Initialize IPFS and Bundler for worker
		ipfsClient := ipfs.NewFromEnv()
		ipfsRepo := store.NewIPFSRepo(database.DB)
		bundler := ipfs.NewBundler(ipfsClient, ipfsRepo)

		// Start JobRunner (Redis -> execute -> Postgres -> IPFS bundling)
		jr := worker.NewJobRunner(database.DB, q, gsvc, bundler)
		go jr.Start(workerCtx)

		// Start OutboxPublisher (Postgres outbox -> Redis)
		op := worker.NewOutboxPublisher(database.DB, q)
		go op.Start(workerCtx)
	}

	// Start server in goroutine
	go func() {
		logger.Info().Str("addr", cfg.HTTPPort).Msg("Starting Project Beacon Runner")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	workerCancel()
	// Close Redis client
	if err := q.Close(); err != nil {
		logger.Error().Err(err).Msg("error closing redis")
	}

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	// Close DB if open
	if database != nil && database.DB != nil {
		_ = database.Close()
	}

	logger.Info().Msg("Server exited")
}
