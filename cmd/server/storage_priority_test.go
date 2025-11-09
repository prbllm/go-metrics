package main

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/prbllm/go-metrics/internal/config"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/prbllm/go-metrics/internal/repository"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func cleanupEnv() {
	os.Unsetenv(config.DatabaseDSNEnvVar)
	os.Unsetenv(config.FileStoragePathEnvVar)
}

func TestStoragePriority_DatabaseDSN(t *testing.T) {
	cleanupEnv()
	defer cleanupEnv()

	dsn := os.Getenv(config.DatabaseDSNEnvVar)
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	}
	os.Setenv(config.DatabaseDSNEnvVar, dsn)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := config.GetConfig()
	cfg.DatabaseDSN = dsn

	repo := selectStorage(cfg, logger)
	require.NotNil(t, repo, "Repository should not be nil")

	_, isPostgres := repo.(*repository.PostgresRepository)
	require.True(t, isPostgres, "Should use PostgreSQL repository when DATABASE_DSN is set")

	if pgRepo, ok := repo.(*repository.PostgresRepository); ok {
		defer func() {
			if closeErr := pgRepo.Close(); closeErr != nil {
				t.Logf("Error closing PostgreSQL connection: %v", closeErr)
			}
		}()
	}
}

func TestStoragePriority_FileStorage(t *testing.T) {
	cleanupEnv()
	defer cleanupEnv()

	tempFile := t.TempDir() + "/test_metrics.json"
	os.Setenv(config.FileStoragePathEnvVar, tempFile)

	logger := zaptest.NewLogger(t).Sugar()
	cfg := config.GetConfig()
	cfg.DatabaseDSN = ""
	cfg.FileStoragePath = tempFile

	repo := selectStorage(cfg, logger)
	require.NotNil(t, repo, "Repository should not be nil")

	_, isFileStorage := repo.(*repository.FileStorageDecorator)
	require.True(t, isFileStorage, "Should use file storage when DATABASE_DSN is not set but FileStoragePath is set")
}

func TestStoragePriority_Memory(t *testing.T) {
	cleanupEnv()
	defer cleanupEnv()

	logger := zaptest.NewLogger(t).Sugar()
	cfg := config.GetConfig()
	cfg.DatabaseDSN = ""
	cfg.FileStoragePath = ""

	repo := selectStorage(cfg, logger)
	require.NotNil(t, repo, "Repository should not be nil")

	_, isMemStorage := repo.(*repository.MemStorage)
	require.True(t, isMemStorage, "Should use memory storage when both DATABASE_DSN and FileStoragePath are not set")
}

func TestStoragePriority_CounterAccumulation_PostgreSQL(t *testing.T) {
	cleanupEnv()
	defer cleanupEnv()

	dsn := os.Getenv(config.DatabaseDSNEnvVar)
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	}

	logger := zaptest.NewLogger(t).Sugar()
	cfg := config.GetConfig()
	cfg.DatabaseDSN = dsn

	repo := selectStorage(cfg, logger)
	if repo == nil {
		t.Skip("Skipping test: database not available")
		return
	}

	if pgRepo, ok := repo.(*repository.PostgresRepository); ok {
		db := getTestDBForCleanup(t, dsn)
		if db != nil {
			_, _ = db.Exec("TRUNCATE TABLE metrics")
			db.Close()
		}
		defer func() {
			if closeErr := pgRepo.Close(); closeErr != nil {
				t.Logf("Error closing PostgreSQL connection: %v", closeErr)
			}
		}()
	}

	delta1 := int64(5)
	metric1 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta1,
	}
	err := repo.UpdateMetric(context.Background(), metric1)
	require.NoError(t, err)

	delta2 := int64(7)
	metric2 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta2,
	}
	err = repo.UpdateMetric(context.Background(), metric2)
	require.NoError(t, err)

	retrieved, err := repo.GetMetric(context.Background(), metric1)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Delta)
	require.Equal(t, int64(12), *retrieved.Delta, "Counter should accumulate values (5+7=12)")
}

func getTestDBForCleanup(t *testing.T, dsn string) *sql.DB {
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil
	}
	db := stdlib.OpenDB(*config)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil
	}
	return db
}

func TestStoragePriority_CounterAccumulation_FileStorage(t *testing.T) {
	cleanupEnv()
	defer cleanupEnv()

	tempFile := t.TempDir() + "/test_metrics.json"
	logger := zaptest.NewLogger(t).Sugar()
	cfg := config.GetConfig()
	cfg.DatabaseDSN = ""
	cfg.FileStoragePath = tempFile

	repo := selectStorage(cfg, logger)
	require.NotNil(t, repo)

	delta1 := int64(5)
	metric1 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta1,
	}
	err := repo.UpdateMetric(context.Background(), metric1)
	require.NoError(t, err)

	delta2 := int64(7)
	metric2 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta2,
	}
	err = repo.UpdateMetric(context.Background(), metric2)
	require.NoError(t, err)

	retrieved, err := repo.GetMetric(context.Background(), metric1)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Delta)
	require.Equal(t, int64(12), *retrieved.Delta, "Counter should accumulate values (5+7=12)")
}

func TestStoragePriority_CounterAccumulation_Memory(t *testing.T) {
	cleanupEnv()
	defer cleanupEnv()

	logger := zaptest.NewLogger(t).Sugar()
	cfg := config.GetConfig()
	cfg.DatabaseDSN = ""
	cfg.FileStoragePath = ""

	repo := selectStorage(cfg, logger)
	require.NotNil(t, repo)

	delta1 := int64(5)
	metric1 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta1,
	}
	err := repo.UpdateMetric(context.Background(), metric1)
	require.NoError(t, err)

	delta2 := int64(7)
	metric2 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta2,
	}
	err = repo.UpdateMetric(context.Background(), metric2)
	require.NoError(t, err)

	retrieved, err := repo.GetMetric(context.Background(), metric1)
	require.NoError(t, err)
	require.NotNil(t, retrieved.Delta)
	require.Equal(t, int64(12), *retrieved.Delta, "Counter should accumulate values (5+7=12)")
}

func selectStorage(cfg *config.Config, logger logger.Logger) repository.MetricsRepository {
	if cfg.DatabaseDSN != "" {
		postgresRepo, err := repository.NewPostgresRepository(context.Background(), cfg.DatabaseDSN, logger)
		if err != nil {
			logger.Errorf("Error creating PostgreSQL repository: %v", err)
			logger.Warn("Falling back to file storage")
			if cfg.FileStoragePath != "" {
				storage := repository.NewMemStorage(logger)
				fileDecorator := repository.NewFileStorageDecorator(storage, cfg.FileStoragePath, logger)
				return fileDecorator
			}
			return repository.NewMemStorage(logger)
		}
		return postgresRepo
	} else if cfg.FileStoragePath != "" {
		storage := repository.NewMemStorage(logger)
		fileDecorator := repository.NewFileStorageDecorator(storage, cfg.FileStoragePath, logger)
		return fileDecorator
	}
	return repository.NewMemStorage(logger)
}
