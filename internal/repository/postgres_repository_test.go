package repository

import (
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/prbllm/go-metrics/internal/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func getTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	}

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Skipf("Skipping test: failed to parse DSN: %v", err)
		return nil
	}

	db := stdlib.OpenDB(*config)
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		db.Close()
		return nil
	}

	return db
}

func setupTestRepository(t *testing.T) (*PostgresRepository, func()) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"
	}

	logger := zaptest.NewLogger(t).Sugar()
	repo, err := NewPostgresRepository(dsn, logger)
	if err != nil {
		t.Skipf("Skipping test: failed to create repository: %v", err)
		return nil, func() {}
	}

	db := getTestDB(t)
	if db != nil {
		_, _ = db.Exec("TRUNCATE TABLE metrics")
		db.Close()
	}

	cleanup := func() {
		db := getTestDB(t)
		if db != nil {
			_, _ = db.Exec("TRUNCATE TABLE metrics")
			db.Close()
		}
		repo.Close()
	}

	return repo, cleanup
}

func TestPostgresRepository_UpdateMetric_Counter(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	delta := int64(10)
	metric := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta,
	}

	err := repo.UpdateMetric(metric)
	require.NoError(t, err, "Failed to update counter metric")

	retrieved, err := repo.GetMetric(metric)
	require.NoError(t, err, "Failed to get metric")
	require.NotNil(t, retrieved.Delta, "Delta should not be nil")
	require.Equal(t, int64(10), *retrieved.Delta, "Delta should be 10")
}

func TestPostgresRepository_UpdateMetric_CounterAccumulation(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	delta1 := int64(5)
	metric1 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta1,
	}

	err := repo.UpdateMetric(metric1)
	require.NoError(t, err, "Failed to update counter metric first time")

	delta2 := int64(7)
	metric2 := &model.Metrics{
		ID:    "test_counter",
		MType: model.Counter,
		Delta: &delta2,
	}

	err = repo.UpdateMetric(metric2)
	require.NoError(t, err, "Failed to update counter metric second time")

	retrieved, err := repo.GetMetric(metric1)
	require.NoError(t, err, "Failed to get metric")
	require.NotNil(t, retrieved.Delta, "Delta should not be nil")
	require.Equal(t, int64(12), *retrieved.Delta, "Delta should be accumulated (5+7=12)")
}

func TestPostgresRepository_UpdateMetric_Gauge(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	value := 3.14
	metric := &model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: &value,
	}

	err := repo.UpdateMetric(metric)
	require.NoError(t, err, "Failed to update gauge metric")

	retrieved, err := repo.GetMetric(metric)
	require.NoError(t, err, "Failed to get metric")
	require.NotNil(t, retrieved.Value, "Value should not be nil")
	require.Equal(t, 3.14, *retrieved.Value, "Value should be 3.14")
}

func TestPostgresRepository_UpdateMetric_GaugeReplacement(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	value1 := 10.5
	metric1 := &model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: &value1,
	}

	err := repo.UpdateMetric(metric1)
	require.NoError(t, err, "Failed to update gauge metric first time")

	value2 := 20.7
	metric2 := &model.Metrics{
		ID:    "test_gauge",
		MType: model.Gauge,
		Value: &value2,
	}

	err = repo.UpdateMetric(metric2)
	require.NoError(t, err, "Failed to update gauge metric second time")

	retrieved, err := repo.GetMetric(metric1)
	require.NoError(t, err, "Failed to get metric")
	require.NotNil(t, retrieved.Value, "Value should not be nil")
	require.Equal(t, 20.7, *retrieved.Value, "Value should be replaced (20.7, not accumulated)")
}

func TestPostgresRepository_GetMetric_NotFound(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	metric := &model.Metrics{
		ID:    "nonexistent",
		MType: model.Counter,
	}

	_, err := repo.GetMetric(metric)
	require.Error(t, err, "Should return error for nonexistent metric")
	require.Contains(t, err.Error(), "not found", "Error should indicate metric not found")
}

func TestPostgresRepository_GetAllMetrics_Empty(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	metrics := repo.GetAllMetrics()
	require.NotNil(t, metrics, "Metrics should not be nil")
	require.Empty(t, metrics, "Metrics should be empty")
}

func TestPostgresRepository_GetAllMetrics_WithData(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	delta1 := int64(10)
	metric1 := &model.Metrics{
		ID:    "counter1",
		MType: model.Counter,
		Delta: &delta1,
	}
	err := repo.UpdateMetric(metric1)
	require.NoError(t, err)

	delta2 := int64(20)
	metric2 := &model.Metrics{
		ID:    "counter2",
		MType: model.Counter,
		Delta: &delta2,
	}
	err = repo.UpdateMetric(metric2)
	require.NoError(t, err)

	value1 := 1.5
	metric3 := &model.Metrics{
		ID:    "gauge1",
		MType: model.Gauge,
		Value: &value1,
	}
	err = repo.UpdateMetric(metric3)
	require.NoError(t, err)

	metrics := repo.GetAllMetrics()
	require.NotNil(t, metrics, "Metrics should not be nil")
	require.Len(t, metrics, 3, "Should have 3 metrics")

	metricMap := make(map[string]*model.Metrics)
	for _, m := range metrics {
		key := m.ID + ":" + m.MType
		metricMap[key] = m
	}

	require.Contains(t, metricMap, "counter1:counter", "Should contain counter1")
	require.Contains(t, metricMap, "counter2:counter", "Should contain counter2")
	require.Contains(t, metricMap, "gauge1:gauge", "Should contain gauge1")
}

func TestPostgresRepository_Ping(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	err := repo.Ping()
	require.NoError(t, err, "Ping should succeed")
}

func TestPostgresRepository_UpdateMetric_InvalidInput(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	err := repo.UpdateMetric(nil)
	require.Error(t, err, "Should return error for nil metric")

	metric := &model.Metrics{
		ID:    "test",
		MType: model.Counter,
		Delta: nil,
	}
	err = repo.UpdateMetric(metric)
	require.Error(t, err, "Should return error for counter without delta")

	metric2 := &model.Metrics{
		ID:    "test",
		MType: model.Gauge,
		Value: nil,
	}
	err = repo.UpdateMetric(metric2)
	require.Error(t, err, "Should return error for gauge without value")

	metric3 := &model.Metrics{
		ID:    "test",
		MType: "unknown",
	}
	err = repo.UpdateMetric(metric3)
	require.Error(t, err, "Should return error for unknown metric type")
}

func TestPostgresRepository_GetMetric_NilInput(t *testing.T) {
	repo, cleanup := setupTestRepository(t)
	if repo == nil {
		return
	}
	defer cleanup()

	_, err := repo.GetMetric(nil)
	require.Error(t, err, "Should return error for nil metric")
}
