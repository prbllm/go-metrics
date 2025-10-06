package main

import (
	"fmt"
	"net/http"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/handler"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"

	"github.com/go-chi/chi/v5"
)

func main() {
	err := config.InitConfig("server")
	if err != nil {
		panic(err)
	}

	storage := repository.NewMemStorage()
	metricsService := service.NewMetricsService(storage)
	handlers := handler.NewHandlers(metricsService)
	router := chi.NewRouter()
	router.Route(config.CommonPath, func(r chi.Router) {
		r.Get("/", handlers.GetAllMetricsHandler)
		r.Route(config.UpdatePath, func(r chi.Router) {
			r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricHandler)
		})
		r.Route(config.ValuePath, func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", handlers.GetValueHandler)
		})
	})

	fmt.Println("Server starting on ", config.GetConfig().ServerHost)
	err = http.ListenAndServe(config.GetConfig().ServerHost, router)
	if err != nil {
		panic(err)
	}
}
