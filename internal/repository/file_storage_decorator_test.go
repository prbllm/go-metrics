package repository

import (
	"os"
	"testing"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestFileStorageDecorator_UpdateMetricsBatch(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_metrics_*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	logger := zaptest.NewLogger(t).Sugar()
	memStorage := NewMemStorage(logger)
	decorator := NewFileStorageDecorator(memStorage, tmpFile.Name(), logger)

	metrics := []*model.Metrics{
		{ID: "counter1", MType: model.Counter, Delta: func() *int64 { v := int64(10); return &v }()},
		{ID: "gauge1", MType: model.Gauge, Value: func() *float64 { v := 3.14; return &v }()},
	}

	originalStoreInterval := config.GetConfig().StoreInterval
	defer func() {
		cfg := config.GetConfig()
		cfg.StoreInterval = originalStoreInterval
	}()

	cfg := config.GetConfig()
	cfg.StoreInterval = 0

	err = decorator.UpdateMetricsBatch(metrics)
	require.NoError(t, err, "UpdateMetricsBatch should succeed")

	_, err = os.Stat(tmpFile.Name())
	require.NoError(t, err, "File should exist")

	counter1, err := decorator.GetMetric(&model.Metrics{ID: "counter1", MType: model.Counter})
	require.NoError(t, err)
	require.Equal(t, int64(10), *counter1.Delta)
}

func TestFileStorageDecorator_UpdateMetricsBatch_NilSlice(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_metrics_*.json")
	require.NoError(t, err)
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	logger := zaptest.NewLogger(t).Sugar()
	memStorage := NewMemStorage(logger)
	decorator := NewFileStorageDecorator(memStorage, tmpFile.Name(), logger)

	err = decorator.UpdateMetricsBatch(nil)
	require.Error(t, err, "Should return error for nil slice")
}
