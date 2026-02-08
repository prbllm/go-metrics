package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prbllm/go-metrics/internal/audit"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
)

// MetricsService реализует интерфейс Service для работы с метриками.
type MetricsService struct {
	repository repository.MetricsRepository
	observers  []audit.MetricsObserver
}

// NewMetricsService создает новый экземпляр MetricsService.
func NewMetricsService(repository repository.MetricsRepository) *MetricsService {
	return &MetricsService{
		repository: repository,
		observers:  make([]audit.MetricsObserver, 0),
	}
}

// RegisterObserver регистрирует наблюдателя для событий аудита.
func (s *MetricsService) RegisterObserver(observer audit.MetricsObserver) {
	if observer != nil {
		s.observers = append(s.observers, observer)
	}
}

// GetMetric возвращает метрику по типу и имени.
func (s *MetricsService) GetMetric(ctx context.Context, metricType, metricName string) (*model.Metrics, error) {
	metric := &model.Metrics{
		MType: metricType,
		ID:    metricName,
	}
	return s.repository.GetMetric(ctx, metric)
}

// UpdateMetric обновляет метрику по типу, имени и значению в виде строки.
func (s *MetricsService) UpdateMetric(ctx context.Context, metricType, metricName, metricValue string) error {
	metric := &model.Metrics{
		MType: metricType,
		ID:    metricName,
	}
	switch metricType {
	case model.Counter:
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid metric value: %w", err)
		}
		metric.Delta = &delta
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("invalid metric value: %w", err)
		}
		metric.Value = &value
	}
	if err := s.repository.UpdateMetric(ctx, metric); err != nil {
		return err
	}
	s.notifyObservers(ctx, []string{metricName})
	return nil
}

// GetAllMetrics возвращает все метрики.
func (s *MetricsService) GetAllMetrics(ctx context.Context) ([]model.Metrics, error) {
	return s.repository.GetAllMetrics(ctx), nil
}

// UpdateMetricByStruct обновляет метрику по структуре Metrics.
func (s *MetricsService) UpdateMetricByStruct(ctx context.Context, metric *model.Metrics) error {
	if err := ValidateMetric(metric); err != nil {
		return err
	}
	if err := s.repository.UpdateMetric(ctx, metric); err != nil {
		return err
	}
	s.notifyObservers(ctx, []string{metric.ID})
	return nil
}

// UpdateMetricsBatchByStruct обновляет пакет метрик.
func (s *MetricsService) UpdateMetricsBatchByStruct(ctx context.Context, metrics []*model.Metrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics are nil")
	}

	for _, metric := range metrics {
		if err := ValidateMetric(metric); err != nil {
			return fmt.Errorf("validation failed for metric %s: %w", metric.ID, err)
		}
	}

	if err := s.repository.UpdateMetricsBatch(ctx, metrics); err != nil {
		return err
	}

	metricsIDs := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		metricsIDs = append(metricsIDs, metric.ID)
	}
	s.notifyObservers(ctx, metricsIDs)
	return nil
}

// Ping проверяет доступность хранилища метрик.
func (s *MetricsService) Ping(ctx context.Context) error {
	return s.repository.Ping(ctx)
}

func (s *MetricsService) notifyObservers(ctx context.Context, metricsIDs []string) {
	if len(s.observers) == 0 || len(metricsIDs) == 0 {
		return
	}

	ipAddress := audit.GetClientIP(ctx)
	event := audit.AuditEvent{
		Timestamp:  time.Now().Unix(),
		MetricsIDs: metricsIDs,
		IPAddress:  ipAddress,
	}

	for _, observer := range s.observers {
		observer.Process(ctx, event)
	}
}
