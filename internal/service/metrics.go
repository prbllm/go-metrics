package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
)

type MetricsService struct {
	repository repository.MetricsRepository
}

func NewMetricsService(repository repository.MetricsRepository) Service {
	return &MetricsService{
		repository: repository,
	}
}

func (s *MetricsService) GetMetric(ctx context.Context, metricType, metricName string) (*model.Metrics, error) {
	metric := &model.Metrics{
		MType: metricType,
		ID:    metricName,
	}
	return s.repository.GetMetric(ctx, metric)
}

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
	return s.repository.UpdateMetric(ctx, metric)
}

func (s *MetricsService) GetAllMetrics(ctx context.Context) ([]model.Metrics, error) {
	return s.repository.GetAllMetrics(ctx), nil
}

func (s *MetricsService) UpdateMetricByStruct(ctx context.Context, metric *model.Metrics) error {
	if err := ValidateMetric(metric); err != nil {
		return err
	}
	return s.repository.UpdateMetric(ctx, metric)
}

func (s *MetricsService) UpdateMetricsBatchByStruct(ctx context.Context, metrics []*model.Metrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics are nil")
	}

	for _, metric := range metrics {
		if err := ValidateMetric(metric); err != nil {
			return fmt.Errorf("validation failed for metric %s: %w", metric.ID, err)
		}
	}

	return s.repository.UpdateMetricsBatch(ctx, metrics)
}

func (s *MetricsService) Ping(ctx context.Context) error {
	return s.repository.Ping(ctx)
}
