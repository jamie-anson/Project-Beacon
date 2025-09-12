package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/jamie-anson/project-beacon-runner/internal/api"
	"github.com/jamie-anson/project-beacon-runner/internal/config"
	"github.com/jamie-anson/project-beacon-runner/internal/db"
	"github.com/jamie-anson/project-beacon-runner/internal/golem"
	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
	"github.com/jamie-anson/project-beacon-runner/internal/hybrid"
	"github.com/jamie-anson/project-beacon-runner/internal/logging"
	"github.com/jamie-anson/project-beacon-runner/internal/metrics"
	"github.com/jamie-anson/project-beacon-runner/internal/queue"
	"github.com/jamie-anson/project-beacon-runner/internal/serverbind"
	"github.com/jamie-anson/project-beacon-runner/internal/service"
	"github.com/jamie-anson/project-beacon-runner/internal/store"
	"github.com/jamie-anson/project-beacon-runner/internal/transparency"
	wsHub "github.com/jamie-anson/project-beacon-runner/internal/websocket"
	"github.com/jamie-anson/project-beacon-runner/internal/worker"

	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
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

	// Materialize trusted keys from environment if provided (helps Fly deploys)
	if path := os.Getenv("TRUSTED_KEYS_FILE"); path != "" {
		if content := os.Getenv("TRUSTED_KEYS_JSON"); content != "" {
			// Ensure parent dir exists
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				logger.Warn().Err(err).Str("path", path).Msg("failed to create parent dir for TRUSTED_KEYS_FILE")
			} else {
				// Basic sanity: must start with '{' or '['
				if len(content) > 0 && (content[0] == '{' || content[0] == '[') {
					if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
						logger.Warn().Err(err).Str("path", path).Msg("failed to write TRUSTED_KEYS_FILE")
					} else {
						logger.Info().Str("path", path).Msg("materialized trusted keys from env")
					}
				} else {
					logger.Warn().Str("path", path).Msg("TRUSTED_KEYS_JSON does not look like JSON; skipping materialization")
				}
			}
		}
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

	// Redis client for security features
	q := queue.MustNewFromEnv()
	var redisClient *redis.Client
	if q != nil && q.GetRedisClient() != nil {
		redisClient = q.GetRedisClient()
	}

	// Setup routes with services and config
	r = api.SetupRoutes(jobsService, cfg, redisClient)

	// Enable OpenTelemetry tracing for Gin routes
	r.Use(otelgin.Middleware("runner-http"))

	// Attach tracing middleware (Gin-aware) and metrics middleware
	r.Use(metrics.GinMiddleware())
	r.GET("/metrics", gin.WrapH(metrics.Handler()))
	// Alias metrics under API namespace for observability tooling compatibility
	r.GET("/api/v1/metrics", gin.WrapH(metrics.Handler()))
	// Support HEAD requests (curl -I) for both metrics endpoints
	r.HEAD("/metrics", gin.WrapH(metrics.Handler()))
	r.HEAD("/api/v1/metrics", gin.WrapH(metrics.Handler()))

	// Expose WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		hub.ServeWS(c.Writer, c.Request)
	})

	// Setup server (handler is the Gin router; tracing via otelgin middleware)
	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: r,
	}

	// Redis client and background processes
	workerCtx, workerCancel := context.WithCancel(context.Background())

	// Start trusted keys hot-reloader if configured
	if cfg.TrustedKeysFile != "" && cfg.TrustedKeysReload > 0 {
		config.StartTrustedKeysReloader(workerCtx, cfg.TrustedKeysFile, cfg.TrustedKeysReload)
	}

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
		// Initialize Hybrid Router client if HYBRID_BASE is set (preferred execution path)
		if base := os.Getenv("HYBRID_BASE"); base != "" {
			jr.Hybrid = hybrid.New(base)
		} else if os.Getenv("ENABLE_HYBRID_DEFAULT") == "1" {
			// Optional: enable default Railway base without env
			jr.Hybrid = hybrid.New("")
		}
		go jr.Start(workerCtx)

		// Start OutboxPublisher (Postgres outbox -> Redis)
		op := worker.NewOutboxPublisher(database.DB, q)
		go op.Start(workerCtx)
	}

	// Start server in goroutine with helper-driven listener-first binding
	go func() {
		desired := cfg.HTTPPort
		strategy := strings.ToLower(strings.TrimSpace(cfg.PortStrategy))

		ln, resolved, err := serverbind.ResolveAndListen(strategy, desired, cfg.PortRangeStart, cfg.PortRangeEnd)
		if err != nil {
			// Strategy-specific messaging
			if strategy == "strict" {
				logger.Fatal().Err(err).Str("addr", desired).Msg("Failed to bind in strict mode")
				return
			}
			if strategy == "ephemeral" {
				logger.Fatal().Err(err).Str("addr", ":0").Msg("Failed to bind in ephemeral mode")
				return
			}
			logger.Fatal().Err(err).Str("addr", desired).Msg("Failed to bind in fallback mode")
			return
		}

		cfg.ResolvedAddr = resolved
		_ = serverbind.WriteAddrFile(cfg.AddrFile, resolved)
		logger.Info().
			Str("addr", resolved).
			Str("strategy", strategy).
			Str("addr_file", cfg.AddrFile).
			Msg("Project Beacon Runner started")
		if _, port, err := net.SplitHostPort(resolved); err == nil && port != "" {
			logger.Info().Msg("Hint: curl http://localhost:" + port + "/health ; curl -H 'Authorization: Bearer $ADMIN_TOKEN' http://localhost:" + port + "/admin/port")
		}

		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server error")
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
