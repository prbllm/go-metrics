package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/service"
)

type Handlers struct {
	metricsService service.MetricsServiceInterface
}

func NewHandlers(metricsService service.MetricsServiceInterface) *Handlers {
	return &Handlers{metricsService: metricsService}
}

func (h *Handlers) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("method=%s uri=%s\n", r.Method, r.RequestURI)
	if r.Method != http.MethodPost {
		fmt.Printf("Method %s not allowed\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	if metricType == "" || metricName == "" || metricValue == "" {
		fmt.Println("Invalid path")
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

func (h *Handlers) GetAllMetricsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("method=%s uri=%s\n", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		fmt.Printf("Method %s not allowed\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics, err := h.metricsService.GetAllMetrics()
	if err != nil {
		fmt.Printf("Error getting metrics: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Metrics Dashboard</title>
</head>
<body>
<ul>`

	for _, metric := range metrics {
		if metric.MType == model.Counter && metric.Delta != nil {
			html += fmt.Sprintf(`<li>%s: %d</li>`, metric.ID, *metric.Delta)
		} else if metric.MType == model.Gauge && metric.Value != nil {
			html += fmt.Sprintf(`<li>%s: %f</li>`, metric.ID, *metric.Value)
		} else {
			html += fmt.Sprintf(`<li>%s: N/A</li>`, metric.ID)
		}
	}

	html += `</ul>
</body>
</html>`

	w.Write([]byte(html))
}

func (h *Handlers) GetValueHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("method=%s uri=%s\n", r.Method, r.RequestURI)
	if r.Method != http.MethodGet {
		fmt.Printf("Method %s not allowed\n", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricType == "" || metricName == "" {
		fmt.Printf("Invalid path: Type=%s, Name=%s\n", metricType, metricName)
		http.NotFound(w, r)
		return
	}

	metric, err := h.metricsService.GetMetric(metricType, metricName)
	if metric == nil || err != nil {
		fmt.Printf("Error getting metric: %v\n", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	fmt.Printf("metric=%s\n", metric.String())
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if metric.MType == model.Counter && metric.Delta != nil {
		fmt.Fprintf(w, "%d", *metric.Delta)
	} else if metric.MType == model.Gauge && metric.Value != nil {
		fmt.Fprintf(w, "%g", *metric.Value)
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
