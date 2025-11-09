package service

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_service.go -package=mocks

import (
	"context"

	"github.com/prbllm/go-metrics/internal/model"
)

type Service interface {
	UpdateMetric(ctx context.Context, metricType, metricName, metricValue string) error
	UpdateMetricByStruct(ctx context.Context, metric *model.Metrics) error
	UpdateMetricsBatchByStruct(ctx context.Context, metrics []*model.Metrics) error
	GetMetric(ctx context.Context, metricType, metricName string) (*model.Metrics, error)
	GetAllMetrics(ctx context.Context) ([]*model.Metrics, error)
	Ping(ctx context.Context) error
}
