package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prbllm/go-metrics/internal/audit"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/service"
)

type Handlers struct {
	service service.Service
	logger  logger.Logger
}

func NewHandlers(service service.Service, logger logger.Logger) *Handlers {
	return &Handlers{
		service: service,
		logger:  logger,
	}
}

func (h *Handlers) UpdateMetricHandlerByURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	ctx = audit.WithClientIP(ctx, r.RemoteAddr)

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")
	metricValue := chi.URLParam(r, "metricValue")

	if metricType == "" || metricName == "" || metricValue == "" {
		http.NotFound(w, r)
		return
	}

	if err := service.ValidateMetricType(metricType); err != nil {
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if err := service.ValidateMetricValue(metricType, metricValue); err != nil {
		http.Error(w, "Invalid metric value", http.StatusBadRequest)
		return
	}

	h.logger.Infof("Received metric: Type=%s, Name=%s, Value=%s", metricType, metricName, metricValue)

	if h.service != nil {
		if err := h.service.UpdateMetric(ctx, metricType, metricName, metricValue); err != nil {
			h.logger.Errorf("Error updating metric: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) UpdateMetricHandlerByJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	ctx = audit.WithClientIP(ctx, r.RemoteAddr)

	if contentType := r.Header.Get(config.ContentTypeHeader); contentType != config.ContentTypeJSON {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var metric model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := service.ValidateMetric(&metric); err != nil {
		http.Error(w, "Invalid metric", http.StatusBadRequest)
		return
	}

	h.logger.Infof("Received metric: %s", metric.String())

	if err := h.service.UpdateMetricByStruct(ctx, &metric); err != nil {
		h.logger.Errorf("Error updating metric: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) UpdateMetricsBatchHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Debugf("UpdateMetricsBatchHandler called: Method=%s, URL=%s, Path=%s", r.Method, r.URL.String(), r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	ctx = audit.WithClientIP(ctx, r.RemoteAddr)

	contentType := r.Header.Get(config.ContentTypeHeader)
	h.logger.Debugf("Content-Type header: %s", contentType)
	if contentType != config.ContentTypeJSON {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var metrics []*model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	h.logger.Debugf("Successfully decoded %d metrics from request body", len(metrics))

	if len(metrics) == 0 {
		http.Error(w, "Empty metrics batch", http.StatusBadRequest)
		return
	}

	h.logger.Infof("Received batch of %d metrics", len(metrics))

	if err := h.service.UpdateMetricsBatchByStruct(ctx, metrics); err != nil {
		h.logger.Errorf("Error updating metrics batch: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (h *Handlers) GetAllMetricsHandlerByURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	metrics, err := h.service.GetAllMetrics(ctx)
	if err != nil {
		h.logger.Errorf("Error getting metrics: %v", err)
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

func (h *Handlers) GetValueHandlerByURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	if metricType == "" || metricName == "" {
		http.NotFound(w, r)
		return
	}

	metric, err := h.service.GetMetric(ctx, metricType, metricName)
	if metric == nil || err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set(config.ContentTypeHeader, config.ContentTypeTextPlain)
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

func (h *Handlers) GetValueHandlerByJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	if contentType := r.Header.Get(config.ContentTypeHeader); contentType != config.ContentTypeJSON {
		http.Error(w, "Invalid content type", http.StatusBadRequest)
		return
	}

	var metric model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := service.ValidateMetricType(metric.MType); err != nil {
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	metricLoaded, err := h.service.GetMetric(ctx, metric.MType, metric.ID)
	if metricLoaded == nil || err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set(config.ContentTypeHeader, config.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metricLoaded)
}

func (h *Handlers) PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	if h.service == nil {
		h.logger.Error("Service is nil")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.service.Ping(ctx); err != nil {
		h.logger.Errorf("Database ping failed: %v", err)
		http.Error(w, "Database unavailable", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
