package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/model"
)

type FileStorageDecorator struct {
	memStorage *MemStorage
	filePath   string
}

func NewFileStorageDecorator(memStorage *MemStorage, filePath string) *FileStorageDecorator {
	return &FileStorageDecorator{memStorage: memStorage, filePath: filePath}
}

func (f *FileStorageDecorator) UpdateMetric(metric *model.Metrics) error {
	err := f.memStorage.UpdateMetric(metric)
	if err != nil {
		return fmt.Errorf("failed to update metric: %w", err)
	}

	if config.GetConfig().StoreInterval == 0 {
		if saveErr := f.SaveToFile(); saveErr != nil {
			config.GetLogger().Errorf("Failed to save metrics: %v", saveErr)
		}
	}

	return nil
}

func (f *FileStorageDecorator) GetMetric(metric *model.Metrics) (*model.Metrics, error) {
	return f.memStorage.GetMetric(metric)
}

func (f *FileStorageDecorator) GetAllMetrics() []*model.Metrics {
	return f.memStorage.GetAllMetrics()
}

func (f *FileStorageDecorator) SaveToFile() error {
	metrics := f.memStorage.GetAllMetrics()
	json, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(f.filePath, json, 0644)
}

func (f *FileStorageDecorator) LoadFromFile() error {
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
		f.memStorage.UpdateMetric(metric)
	}
	return nil
}

func (f *FileStorageDecorator) StartPeriodicSave(ctx context.Context) {
	storeInterval := config.GetConfig().StoreInterval
	if storeInterval <= 0 {
		config.GetLogger().Info("StoreInterval is 0 or negative, skipping periodic save")
		return
	}

	config.GetLogger().Infof("Starting periodic save every %v", storeInterval)

	go func() {
		ticker := time.NewTicker(storeInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := f.SaveToFile(); err != nil {
					config.GetLogger().Errorf("Failed to save metrics: %v", err)
				} else {
					config.GetLogger().Infof("Metrics saved to file: %s", f.filePath)
				}
			case <-ctx.Done():
				config.GetLogger().Info("Stopping periodic save")
				return
			}
		}
	}()
}
