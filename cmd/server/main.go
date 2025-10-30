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

	storage := repository.NewMemStorage(appLogger)
	fileDecorator := repository.NewFileStorageDecorator(storage, config.GetConfig().FileStoragePath, appLogger)

	if config.GetConfig().Restore {
		err = fileDecorator.LoadFromFile()
		if err != nil {
			appLogger.Errorf("Error loading file: %v", err)
		} else {
			appLogger.Info("Metrics loaded from file")
		}
	}

	fileDecorator.StartPeriodicSave(ctx)

	metricsService := service.NewMetricsService(fileDecorator)
	handlers := handler.NewHandlers(metricsService, appLogger)
	router := chi.NewRouter()

	router.Use(handler.LoggingMiddleware(appLogger))
	router.Use(handler.GzipDecompressMiddleware(appLogger))

	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByURL)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByURL)
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandlerByURL)
			r.Post("/", handlers.GetValueHandlerByJSON)
		})
	})

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
}
