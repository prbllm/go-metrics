package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	err := config.InitConfig("server")
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

	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService)
	router := chi.NewRouter()

	router.Use(handler.LoggingMiddleware())

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

	config.GetLogger().Infof("Server starting on %s", config.GetConfig().ServerHost)
	err = http.ListenAndServe(config.GetConfig().ServerHost, router)
	if err != nil {
		config.GetLogger().Fatalf("Error starting server: %v", err)
	}
}
