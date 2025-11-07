package repository

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/prbllm/go-metrics/internal/logger"
	"github.com/prbllm/go-metrics/internal/migrations"
	"github.com/prbllm/go-metrics/internal/model"
)

type PostgresRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewPostgresRepository(dsn string, logger logger.Logger) (*PostgresRepository, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN cannot be empty")
	}

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	db := stdlib.OpenDB(*config)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Infof("Connected to PostgreSQL database: %s", dsn)

	if err := migrations.RunMigrations(db, logger); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &PostgresRepository{
		db:     db,
		logger: logger,
	}, nil
}

func (p *PostgresRepository) Close() error {
	if p.db != nil {
		p.logger.Info("Closing PostgreSQL database connection")
		return p.db.Close()
	}
	return nil
}

func (p *PostgresRepository) UpdateMetric(metric *model.Metrics) error {
	if metric == nil {
		return fmt.Errorf("metric is nil")
	}

	p.logger.Debugf("PostgresRepository.UpdateMetric called for metric: %s", metric.String())

	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("delta is required for counter metric")
		}

		query := `
			INSERT INTO metrics (id, type, delta, value, updated_at)
			VALUES ($1, $2, $3, NULL, CURRENT_TIMESTAMP)
			ON CONFLICT (id, type)
			DO UPDATE SET
				delta = metrics.delta + EXCLUDED.delta,
				updated_at = CURRENT_TIMESTAMP
		`
		_, err := p.db.Exec(query, metric.ID, metric.MType, *metric.Delta)
		if err != nil {
			return fmt.Errorf("failed to update counter metric: %w", err)
		}

	case model.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("value is required for gauge metric")
		}

		query := `
			INSERT INTO metrics (id, type, delta, value, updated_at)
			VALUES ($1, $2, NULL, $3, CURRENT_TIMESTAMP)
			ON CONFLICT (id, type)
			DO UPDATE SET
				value = EXCLUDED.value,
				updated_at = CURRENT_TIMESTAMP
		`
		_, err := p.db.Exec(query, metric.ID, metric.MType, *metric.Value)
		if err != nil {
			return fmt.Errorf("failed to update gauge metric: %w", err)
		}

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func (p *PostgresRepository) GetMetric(metric *model.Metrics) (*model.Metrics, error) {
	if metric == nil {
		return nil, fmt.Errorf("metric is nil")
	}

	p.logger.Debugf("PostgresRepository.GetMetric called for metric: %s", metric.String())

	query := `
		SELECT id, type, delta, value
		FROM metrics
		WHERE id = $1 AND type = $2
	`

	var id, mType string
	var delta sql.NullInt64
	var value sql.NullFloat64

	err := p.db.QueryRow(query, metric.ID, metric.MType).Scan(&id, &mType, &delta, &value)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("metric %s:%s not found", metric.MType, metric.ID)
		}
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	result := &model.Metrics{
		ID:    id,
		MType: mType,
	}

	if delta.Valid {
		result.Delta = &delta.Int64
	}

	if value.Valid {
		result.Value = &value.Float64
	}

	return result, nil
}

func (p *PostgresRepository) GetAllMetrics() []*model.Metrics {
	p.logger.Debugf("PostgresRepository.GetAllMetrics called")

	query := `
		SELECT id, type, delta, value
		FROM metrics
		ORDER BY id, type
	`

	rows, err := p.db.Query(query)
	if err != nil {
		p.logger.Errorf("Failed to query all metrics: %v", err)
		return []*model.Metrics{}
	}
	defer rows.Close()

	metrics := make([]*model.Metrics, 0)

	for rows.Next() {
		var id, mType string
		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&id, &mType, &delta, &value); err != nil {
			p.logger.Errorf("Failed to scan metric row: %v", err)
			continue
		}

		metric := &model.Metrics{
			ID:    id,
			MType: mType,
		}

		if delta.Valid {
			metric.Delta = &delta.Int64
		}

		if value.Valid {
			metric.Value = &value.Float64
		}

		metrics = append(metrics, metric)
	}

	if err := rows.Err(); err != nil {
		p.logger.Errorf("Error iterating over metrics rows: %v", err)
		return []*model.Metrics{}
	}

	p.logger.Debugf("Retrieved %d metrics from database", len(metrics))
	return metrics
}

func (p *PostgresRepository) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return p.db.Ping()
}
