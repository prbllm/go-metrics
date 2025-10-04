package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prbllm/go-metrics/internal/service"
)

type Handlers struct {
	metricsService service.MetricsServiceInterface
}

func NewHandlers(metricsService service.MetricsServiceInterface) *Handlers {
	return &Handlers{metricsService: metricsService}
}

func (h *Handlers) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Println("Method not allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/update/")
	parts := strings.Split(path, "/")

	if len(parts) != 3 {
		fmt.Println("Invalid path")
		http.NotFound(w, r)
		return
	}

	metricType := parts[0]
	metricName := parts[1]
	metricValue := parts[2]

	if metricName == "" {
		fmt.Printf("Invalid metric name: Type=%s, Name=%s, Value=%s\n", metricType, metricName, metricValue)
		http.NotFound(w, r)
		return
	}

	if err := service.ValidateMetricType(metricType); err != nil {
		fmt.Printf("Invalid metric type: Type=%s, Name=%s, Value=%s\n", metricType, metricName, metricValue)
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if err := service.ValidateMetricValue(metricType, metricValue); err != nil {
		fmt.Printf("Invalid metric value: Type=%s, Name=%s, Value=%s\n", metricType, metricName, metricValue)
		http.Error(w, "Invalid metric value", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received metric: Type=%s, Name=%s, Value=%s\n", metricType, metricName, metricValue)

	if h.metricsService != nil {
		if err := h.metricsService.UpdateMetric(metricType, metricName, metricValue); err != nil {
			fmt.Printf("Error updating metric: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
