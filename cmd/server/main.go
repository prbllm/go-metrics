package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"

	"github.com/go-chi/chi/v5"
)

func createFileStorage(ctx context.Context, filePath string, restore bool, logger logger.Logger) repository.MetricsRepository {
	storage := repository.NewMemStorage(logger)
	fileDecorator := repository.NewFileStorageDecorator(storage, filePath, logger)

	if restore {
		if loadErr := fileDecorator.LoadFromFile(ctx); loadErr != nil {
			logger.Warnf("Error loading file: %v", loadErr)
		} else {
			logger.Info("Metrics loaded from file")
		}
	}

	fileDecorator.StartPeriodicSave(ctx)
	logger.Info("Using file storage")
	return fileDecorator
}

func createMetricsRepository(ctx context.Context, cfg *config.Config, appLogger logger.Logger) repository.MetricsRepository {
	if cfg.DatabaseDSN != "" {
		postgresRepo, err := repository.NewPostgresRepository(ctx, cfg.DatabaseDSN, appLogger)
		if err != nil {
			appLogger.Errorf("Error creating PostgreSQL repository: %v", err)
			appLogger.Warn("Falling back to file storage")
			if cfg.FileStoragePath != "" {
				return createFileStorage(ctx, cfg.FileStoragePath, cfg.Restore, appLogger)
			}
			appLogger.Info("Using in-memory storage")
			return repository.NewMemStorage(appLogger)
		}
		appLogger.Info("Using PostgreSQL storage")
		return postgresRepo
	}

	if cfg.FileStoragePath != "" {
		return createFileStorage(ctx, cfg.FileStoragePath, cfg.Restore, appLogger)
	}

	appLogger.Info("Using in-memory storage")
	return repository.NewMemStorage(appLogger)
}

func main() {
	appLogger, err := logger.NewZapLogger()
	if err != nil {
		fmt.Println("Error creating logger: ", err)
		os.Exit(1)
	}
	defer appLogger.Sync()

	err = config.InitConfig(config.ServerFlagsSet, appLogger)
	if err != nil {
		fmt.Println("Error initializing config: ", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.GetConfig()
	metricsRepository := createMetricsRepository(ctx, cfg, appLogger)

	metricsService := service.NewMetricsService(metricsRepository)

	handlers := handler.NewHandlers(metricsService, appLogger)
	router := chi.NewRouter()

	router.Use(handler.LoggingMiddleware(appLogger), handler.GzipDecompressMiddleware(appLogger))

	router.Get(config.PingPath, handlers.PingHandler)

	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByURL)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.UpdatesPath, func(r chi.Router) {
			r.Post("/", handlers.UpdateMetricsBatchHandler)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandlerByURL)
			r.Post("/", handlers.GetValueHandlerByJSON)
		})
	})

	router.NotFound(handlers.NotFoundHandler)

	server := &http.Server{
		Addr:    config.GetConfig().ServerHost,
		Handler: router,
	}

	serverErr := make(chan error, 1)

	go func() {
		appLogger.Infof("Server starting on %s", config.GetConfig().ServerHost)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		appLogger.Info("Received shutdown signal, shutting down server...")
	case err := <-serverErr:
		appLogger.Errorf("Server error: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Errorf("Server forced to shutdown: %v", err)
	} else {
		appLogger.Info("Server exited gracefully")
	}

	if pgRepo, ok := metricsRepository.(*repository.PostgresRepository); ok {
		if closeErr := pgRepo.Close(); closeErr != nil {
			appLogger.Errorf("Error closing PostgreSQL connection: %v", closeErr)
		} else {
			appLogger.Info("PostgreSQL connection closed")
		}
	}
}
