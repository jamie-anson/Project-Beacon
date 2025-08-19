package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strings"

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
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/transparency"
	wsHub "github.com/jamie-anson/project-beacon-runner/internal/websocket"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	// Initialize structured logging
	logger := logging.Init()
	logger.Info().Msg("logger initialized")
	// Load config
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("invalid configuration")
	}

	// Initialize OpenTelemetry (optional, enabled when OTEL_EXPORTER_OTLP_ENDPOINT is set)
	tp, tpClose := initOpenTelemetry(context.Background(), "project-beacon-runner")
	if tp != nil {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = tp.Shutdown(ctx)
			if tpClose != nil {
				tpClose()
			}
		}()
		logger.Info().Msg("OpenTelemetry initialized")
	}

	// Placeholder router; final router will come from api.SetupRoutes
	r := gin.Default()

	// Initialize database (non-fatal if not available; db package handles fallback)
	database, err := db.Initialize(cfg.DatabaseURL)
	if err != nil {
		logger.Warn().Err(err).Msg("database initialization warning")
	}

	// Initialize JobsService if DB is available
	var jobsService *service.JobsService
	if database != nil && database.DB != nil {
		jobsService = service.NewJobsServiceWithQueue(database.DB, cfg.JobsQueueName)
	} else {
		jobsService = &service.JobsService{}
	}

	// Initialize WebSocket hub and transparency sinks
	hub := wsHub.NewHub()
	go hub.Run()
	if database != nil && database.DB != nil {
		transparencyRepo := store.NewTransparencyRepo(database.DB)
		transparency.SetRepo(transparencyRepo)
	}
	transparency.RegisterBroadcaster(hub.BroadcastMessage)

	// Setup routes with services and config
	r = api.SetupRoutes(jobsService, cfg)

	// Attach metrics middleware and endpoint to the final router
	r.Use(metrics.GinMiddleware())
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	// Expose WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

	// Setup server (wrap handler with otelhttp for tracing)
	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: otelhttp.NewHandler(r, "runner-http"),
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
		jr := worker.NewJobRunnerWithQueue(database.DB, q, gsvc, bundler, cfg.JobsQueueName)
		go jr.Start(workerCtx)

		// Start OutboxPublisher (Postgres outbox -> Redis)
		op := worker.NewOutboxPublisher(database.DB, q)
		go op.Start(workerCtx)
	}

	// Start server in goroutine
	go func() {
		addr1 := cfg.HTTPPort
		logger.Info().Str("addr", addr1).Msg("Starting Project Beacon Runner")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// If primary port is busy and it's the default :8090, retry on :8091
			if addr1 == ":8090" && (strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "bind")) {
				addr2 := ":8091"
				logger.Warn().Err(err).Str("from", addr1).Str("to", addr2).Msg("Port busy, retrying on fallback port")
				srv.Addr = addr2
				logger.Info().Str("addr", addr2).Msg("Starting Project Beacon Runner")
				if err2 := srv.ListenAndServe(); err2 != nil && err2 != http.ErrServerClosed {
					logger.Fatal().Err(err2).Msg("Failed to start server on fallback port")
				}
				return
			}
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

// initOpenTelemetry configures a tracer provider with OTLP exporter if endpoint env is set.
func initOpenTelemetry(ctx context.Context, serviceName string) (*trace.TracerProvider, func()) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// No exporter configured; still set global propagators for inbound context
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		return nil, nil
	}

	insecure := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE") == "true"
	clientOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if insecure {
		clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, clientOpts...)
	if err != nil {
		return nil, nil
	}

	resEnv, _ := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	res, _ := resource.Merge(resource.Default(), resEnv)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, func() {}
}
