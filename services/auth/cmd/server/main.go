package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/TarasDemchenkooo/backend/pkg/observability/admin"
	"github.com/TarasDemchenkooo/backend/pkg/observability/logger"
	"github.com/TarasDemchenkooo/backend/pkg/observability/metrics"
	"github.com/TarasDemchenkooo/backend/pkg/observability/tracing"
	"github.com/TarasDemchenkooo/backend/services/auth/internal/application"
	"github.com/TarasDemchenkooo/backend/services/auth/internal/config"
	httpdelivery "github.com/TarasDemchenkooo/backend/services/auth/internal/delivery/http"
	"github.com/TarasDemchenkooo/backend/services/auth/internal/infrastructure/crypto"
	"github.com/TarasDemchenkooo/backend/services/auth/internal/infrastructure/kafka"
	"github.com/TarasDemchenkooo/backend/services/auth/internal/infrastructure/postgres"
	redisadapter "github.com/TarasDemchenkooo/backend/services/auth/internal/infrastructure/redis"
)

const traceFlushLimit = 5 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.New(logger.Config{}).Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	})

	if err := run(cfg, log); err != nil {
		log.Error("service stopped with error", "error", err)
		os.Exit(1)
	}

	log.Info("service stopped")
}

func run(cfg config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tracing.SetErrorHandler(log)

	tracer, err := tracing.New(ctx, tracing.Config{
		Enabled:        cfg.Tracing.Enabled,
		ServiceName:    cfg.App.Name,
		ServiceVersion: cfg.App.Version,
		Environment:    cfg.App.Environment,
		Endpoint:       cfg.Tracing.Endpoint,
		Insecure:       cfg.Tracing.Insecure,
		SampleRatio:    cfg.Tracing.SampleRatio,
	})
	if err != nil {
		return err
	}
	defer shutdownTracer(tracer, log)

	log.InfoContext(ctx, "tracing initialized",
		"enabled", cfg.Tracing.Enabled,
		"endpoint", cfg.Tracing.Endpoint,
		"sample_ratio", cfg.Tracing.SampleRatio,
	)

	serviceMetrics, err := metrics.New(metrics.Config{Namespace: cfg.Metrics.Namespace})
	if err != nil {
		return err
	}

	adminSrv := admin.New(admin.Config{
		Addr:     cfg.Admin.Addr,
		Gatherer: serviceMetrics.Registry(),
		Logger:   log,
	})

	errCh := make(chan error, 2)

	go func() {
		log.Info("admin server started", "addr", adminSrv.Addr(), "metrics_path", admin.MetricsPath)
		if err := adminSrv.Start(); err != nil {
			errCh <- err
		}
	}()

	pool, redisClient, err := connectDependencies(ctx, cfg, log)
	if err != nil {
		closeAdmin(adminSrv, log)

		if ctx.Err() != nil {
			log.Info("shutdown signal received during startup")

			return nil
		}

		return err
	}
	defer pool.Close()
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("close redis client", "error", err)
		}
	}()

	adminSrv.AddReadinessCheck("postgres", pool.Ping)
	adminSrv.AddReadinessCheck("redis", func(ctx context.Context) error {
		return redisClient.Ping(ctx).Err()
	})

	auth := application.NewRegisterUseCase(
		postgres.NewUserRepository(pool),
		redisadapter.NewCodeStore(redisClient),
		kafka.NewNoopPublisher(log),
		crypto.NewBcryptHasher(cfg.Auth.BcryptCost),
		cfg.Auth.VerificationCodeTTL,
		log,
	)

	router := httpdelivery.NewRouter(httpdelivery.NewAuthHandler(auth, log), log, serviceMetrics)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("http server started", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	adminSrv.MarkReady()
	log.Info("service ready")

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		stop()
		log.Info("shutdown signal received")
	}

	return shutdown(srv, adminSrv, cfg.Shutdown, log)
}

func connectDependencies(ctx context.Context, cfg config.Config, log *slog.Logger) (*pgxpool.Pool, *redis.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, cfg.Startup.Timeout)
	defer cancel()

	pool, err := postgres.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		return nil, nil, fmt.Errorf("connect dependencies within %s: %w", cfg.Startup.Timeout, err)
	}

	log.InfoContext(ctx, "connected to postgres")

	redisClient, err := redisadapter.New(ctx, redisadapter.Config{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		pool.Close()

		return nil, nil, fmt.Errorf("connect dependencies within %s: %w", cfg.Startup.Timeout, err)
	}

	log.InfoContext(ctx, "connected to redis", "addr", cfg.Redis.Addr, "db", cfg.Redis.DB)

	return pool, redisClient, nil
}

func closeAdmin(adminSrv *admin.Server, log *slog.Logger) {
	if err := adminSrv.Close(); err != nil {
		log.Error("close admin server", "error", err)
	}
}

func shutdown(srv *http.Server, adminSrv *admin.Server, cfg config.Shutdown, log *slog.Logger) error {
	adminSrv.MarkShuttingDown()

	log.Info("readiness turned off, draining", "delay", cfg.DrainDelay)
	time.Sleep(cfg.DrainDelay)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var errs []error

	if err := srv.Shutdown(ctx); err != nil {
		errs = append(errs, fmt.Errorf("shutdown http server: %w", err))

		if err := srv.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close http server: %w", err))
		}
	}

	log.Info("http server stopped")

	if err := adminSrv.Shutdown(ctx); err != nil {
		errs = append(errs, err)

		if err := adminSrv.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	log.Info("admin server stopped")

	return errors.Join(errs...)
}

func shutdownTracer(tracer *tracing.Provider, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), traceFlushLimit)
	defer cancel()

	if err := tracer.Shutdown(ctx); err != nil {
		log.Error("shutdown tracing", "error", err)
	}
}
