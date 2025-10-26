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
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	err := config.InitConfig(config.ServerFlagsSet)
	if err != nil {
		fmt.Println("Error initializing config: ", err)
		os.Exit(1)
	}

	err = config.InitLogger()
	if err != nil {
		fmt.Println("Error initializing logger: ", err)
		os.Exit(1)
	}
	defer config.GetLogger().Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		config.GetLogger().Info("Received shutdown signal")
		cancel()
	}()

	storage := repository.NewMemStorage()
	fileDecorator := repository.NewFileStorageDecorator(storage, config.GetConfig().FileStoragePath)

	if config.GetConfig().Restore {
		err = fileDecorator.LoadFromFile()
		if err != nil {
			config.GetLogger().Errorf("Error loading file: %v", err)
		} else {
			config.GetLogger().Info("Metrics loaded from file")
		}
	}

	fileDecorator.StartPeriodicSave(ctx)

	metricsService := service.NewMetricsService(fileDecorator)
	handlers := handler.NewHandlers(metricsService)
	router := chi.NewRouter()

	router.Use(handler.LoggingMiddleware())
	router.Use(handler.GzipDecompressMiddleware())

	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandlerByUrl)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandlerByUrl)
			r.Post("/", handlers.UpdateMetricHandlerByJSON)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandlerByUrl)
			r.Post("/", handlers.GetValueHandlerByJSON)
		})
	})

	server := &http.Server{
		Addr:    config.GetConfig().ServerHost,
		Handler: router,
	}

	go func() {
		config.GetLogger().Infof("Server starting on %s", config.GetConfig().ServerHost)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			config.GetLogger().Fatalf("Error starting server: %v", err)
		}
	}()

	<-ctx.Done()
	config.GetLogger().Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		config.GetLogger().Errorf("Server forced to shutdown: %v", err)
	} else {
		config.GetLogger().Info("Server exited gracefully")
	}
}
