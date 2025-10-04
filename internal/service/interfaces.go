package service

import "github.com/prbllm/go-metrics/internal/model"

type MetricsServiceInterface interface {
	UpdateMetric(metricType, metricName, metricValue string) error
	GetMetric(metricType, metricName string) (*model.Metrics, error)
}
