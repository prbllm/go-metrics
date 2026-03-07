package grpcserver

import (
	"context"
	"testing"

	metricsv1 "github.com/prbllm/go-metrics/internal/proto/metrics/v1"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/prbllm/go-metrics/internal/service"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestMetricsServer_UpdateMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	repo := repository.NewMemStorage(logger)
	svc := service.NewMetricsService(repo)
	srv := NewMetricsServer(svc)
	ctx := context.Background()

	t.Run("empty request returns ok", func(t *testing.T) {
		resp, err := srv.UpdateMetrics(ctx, &metricsv1.UpdateMetricsRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("nil request returns ok", func(t *testing.T) {
		resp, err := srv.UpdateMetrics(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	t.Run("batch is stored and retrievable", func(t *testing.T) {
		delta := int64(42)
		value := 3.14
		req := &metricsv1.UpdateMetricsRequest{
			Metrics: []*metricsv1.Metric{
				{Id: "counter1", Type: metricsv1.Metric_COUNTER, Delta: delta},
				{Id: "gauge1", Type: metricsv1.Metric_GAUGE, Value: value},
			},
		}
		resp, err := srv.UpdateMetrics(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		all, err := svc.GetAllMetrics(ctx)
		require.NoError(t, err)
		require.Len(t, all, 2)
		var gotCounter, gotGauge *model.Metrics
		for i := range all {
			if all[i].ID == "counter1" {
				gotCounter = &all[i]
			}
			if all[i].ID == "gauge1" {
				gotGauge = &all[i]
			}
		}
		require.NotNil(t, gotCounter)
		require.NotNil(t, gotGauge)
		require.Equal(t, model.Counter, gotCounter.MType)
		require.NotNil(t, gotCounter.Delta)
		require.Equal(t, delta, *gotCounter.Delta)
		require.Equal(t, model.Gauge, gotGauge.MType)
		require.NotNil(t, gotGauge.Value)
		require.Equal(t, value, *gotGauge.Value)
	})
}
