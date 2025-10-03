package repository

import (
	"context"

	"github.com/prbllm/go-metrics/internal/model"
)

type MetricsRepository interface {
	UpdateMetric(ctx context.Context, metric *model.Metrics) error
}
