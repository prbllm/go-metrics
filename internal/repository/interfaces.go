package repository

//go:generate mockgen -source=interfaces.go -destination=../mocks/mock_repository.go -package=mocks

import (
	"context"

	"github.com/prbllm/go-metrics/internal/model"
)

type MetricsRepository interface {
	UpdateMetric(ctx context.Context, metric *model.Metrics) error
	UpdateMetricsBatch(ctx context.Context, metrics []*model.Metrics) error
	GetMetric(ctx context.Context, metric *model.Metrics) (*model.Metrics, error)
	GetAllMetrics(ctx context.Context) []model.Metrics
	Ping(ctx context.Context) error
}
