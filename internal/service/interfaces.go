package service

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_service.go -package=mocks

import "github.com/prbllm/go-metrics/internal/model"

type Service interface {
	UpdateMetric(metricType, metricName, metricValue string) error
	UpdateMetricByStruct(metric *model.Metrics) error
	GetMetric(metricType, metricName string) (*model.Metrics, error)
	GetAllMetrics() ([]*model.Metrics, error)
	Ping() error
}
