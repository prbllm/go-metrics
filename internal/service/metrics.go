package service

import (
	"fmt"
	"strconv"

	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
)

type MetricsService struct {
	repository repository.MetricsRepository
}

func NewMetricsService(repository repository.MetricsRepository) Service {
	return &MetricsService{repository: repository}
}

func (s *MetricsService) GetMetric(metricType, metricName string) (*model.Metrics, error) {
	metric := &model.Metrics{
		MType: metricType,
		ID:    metricName,
	}
	return s.repository.GetMetric(metric)
}

func (s *MetricsService) UpdateMetric(metricType, metricName, metricValue string) error {
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
	return s.repository.UpdateMetric(metric)
}

func (s *MetricsService) GetAllMetrics() ([]*model.Metrics, error) {
	return s.repository.GetAllMetrics(), nil
}
