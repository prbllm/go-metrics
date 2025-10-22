package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/service"
)

type Handlers struct {
	service service.Service
}

func NewHandlers(service service.Service) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) UpdateMetricHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		config.GetLogger().Errorf("Method %s not allowed", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	if metricType == "" || metricName == "" || metricValue == "" {
		config.GetLogger().Errorf("Invalid path")
		http.NotFound(w, r)
		return
	}

	if err := service.ValidateMetricType(metricType); err != nil {
		config.GetLogger().Errorf("Invalid metric type: Type=%s, Name=%s, Value=%s", metricType, metricName, metricValue)
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if err := service.ValidateMetricValue(metricType, metricValue); err != nil {
		config.GetLogger().Errorf("Invalid metric value: Type=%s, Name=%s, Value=%s", metricType, metricName, metricValue)
		http.Error(w, "Invalid metric value", http.StatusBadRequest)
		return
	}

	config.GetLogger().Infof("Received metric: Type=%s, Name=%s, Value=%s", metricType, metricName, metricValue)

	if h.service != nil {
		if err := h.service.UpdateMetric(metricType, metricName, metricValue); err != nil {
			config.GetLogger().Errorf("Error updating metric: %v", err)
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
	if r.Method != http.MethodGet {
		config.GetLogger().Errorf("Method %s not allowed", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics, err := h.service.GetAllMetrics()
	if err != nil {
		config.GetLogger().Errorf("Error getting metrics: %v", err)
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
	if r.Method != http.MethodGet {
		config.GetLogger().Errorf("Method %s not allowed", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricType == "" || metricName == "" {
		config.GetLogger().Errorf("Invalid path: Type=%s, Name=%s", metricType, metricName)
		http.NotFound(w, r)
		return
	}

	metric, err := h.service.GetMetric(metricType, metricName)
	if metric == nil || err != nil {
		config.GetLogger().Errorf("Error getting metric: %v", err)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

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
