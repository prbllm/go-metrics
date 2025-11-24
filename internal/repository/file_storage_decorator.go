package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
)

type FileStorageDecorator struct {
	memStorage *MemStorage
	filePath   string
	logger     logger.Logger
}

func NewFileStorageDecorator(memStorage *MemStorage, filePath string, logger logger.Logger) *FileStorageDecorator {
	return &FileStorageDecorator{
		memStorage: memStorage,
		filePath:   filePath,
		logger:     logger,
	}
}

func (f *FileStorageDecorator) saveIfSyncMode(ctx context.Context) {
	if config.GetConfig().StoreInterval == 0 {
		if saveErr := f.SaveToFile(ctx); saveErr != nil {
			f.logger.Errorf("Failed to save metrics: %v", saveErr)
		}
	}
}

func (f *FileStorageDecorator) UpdateMetric(ctx context.Context, metric *model.Metrics) error {
	err := f.memStorage.UpdateMetric(ctx, metric)
	if err != nil {
		return fmt.Errorf("failed to update metric: %w", err)
	}

	f.saveIfSyncMode(ctx)
	return nil
}

func (f *FileStorageDecorator) UpdateMetricsBatch(ctx context.Context, metrics []*model.Metrics) error {
	err := f.memStorage.UpdateMetricsBatch(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to update metrics batch: %w", err)
	}

	f.saveIfSyncMode(ctx)
	return nil
}

func (f *FileStorageDecorator) GetMetric(ctx context.Context, metric *model.Metrics) (*model.Metrics, error) {
	return f.memStorage.GetMetric(ctx, metric)
}

func (f *FileStorageDecorator) GetAllMetrics(ctx context.Context) []model.Metrics {
	return f.memStorage.GetAllMetrics(ctx)
}

func (f *FileStorageDecorator) Ping(ctx context.Context) error {
	return f.memStorage.Ping(ctx)
}

func (f *FileStorageDecorator) SaveToFile(ctx context.Context) error {
	metrics := f.memStorage.GetAllMetrics(ctx)
	json, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(f.filePath, json, 0644)
}

func (f *FileStorageDecorator) LoadFromFile(ctx context.Context) error {
	file, err := os.ReadFile(f.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	metrics := make([]*model.Metrics, 0)
	err = json.Unmarshal(file, &metrics)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file: %w", err)
	}
	for _, metric := range metrics {
		f.memStorage.UpdateMetric(ctx, metric)
	}
	return nil
}

func (f *FileStorageDecorator) StartPeriodicSave(ctx context.Context) {
	storeInterval := config.GetConfig().StoreInterval
	if storeInterval <= 0 {
		f.logger.Info("StoreInterval is 0 or negative, skipping periodic save")
		return
	}

	f.logger.Infof("Starting periodic save every %v", storeInterval)

	go func() {
		ticker := time.NewTicker(storeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := f.SaveToFile(ctx); err != nil {
					f.logger.Errorf("Failed to save metrics: %v", err)
				} else {
					f.logger.Infof("Metrics saved to file: %s", f.filePath)
				}
			case <-ctx.Done():
				f.logger.Info("Stopping periodic save")
				return
			}
		}
	}()
}
