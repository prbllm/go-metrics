package repository

import (
	"context"
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

	updateCounterStmt *sql.Stmt
	updateGaugeStmt   *sql.Stmt
	getMetricStmt     *sql.Stmt
	getAllMetricsStmt *sql.Stmt
}

func NewPostgresRepository(ctx context.Context, dsn string, logger logger.Logger) (*PostgresRepository, error) {
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

	repo := &PostgresRepository{
		db:     db,
		logger: logger,
	}

	if err := repo.initPreparedStatements(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize prepared statements: %w", err)
	}

	return repo, nil
}

func (p *PostgresRepository) initPreparedStatements(ctx context.Context) error {
	counterQuery := `
		INSERT INTO metrics (id, type, delta, value, updated_at)
		VALUES ($1, $2, $3, NULL, CURRENT_TIMESTAMP)
		ON CONFLICT (id, type)
		DO UPDATE SET
			delta = metrics.delta + EXCLUDED.delta,
			updated_at = CURRENT_TIMESTAMP
	`
	stmt, err := p.db.PrepareContext(ctx, counterQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare counter statement: %w", err)
	}
	p.updateCounterStmt = stmt

	gaugeQuery := `
		INSERT INTO metrics (id, type, delta, value, updated_at)
		VALUES ($1, $2, NULL, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (id, type)
		DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP
	`
	stmt, err = p.db.PrepareContext(ctx, gaugeQuery)
	if err != nil {
		p.updateCounterStmt.Close()
		return fmt.Errorf("failed to prepare gauge statement: %w", err)
	}
	p.updateGaugeStmt = stmt

	getMetricQuery := `
		SELECT id, type, delta, value
		FROM metrics
		WHERE id = $1 AND type = $2
	`
	stmt, err = p.db.PrepareContext(ctx, getMetricQuery)
	if err != nil {
		p.updateCounterStmt.Close()
		p.updateGaugeStmt.Close()
		return fmt.Errorf("failed to prepare get metric statement: %w", err)
	}
	p.getMetricStmt = stmt

	getAllMetricsQuery := `
		SELECT id, type, delta, value
		FROM metrics
		ORDER BY id, type
	`
	stmt, err = p.db.PrepareContext(ctx, getAllMetricsQuery)
	if err != nil {
		p.updateCounterStmt.Close()
		p.updateGaugeStmt.Close()
		p.getMetricStmt.Close()
		return fmt.Errorf("failed to prepare get all metrics statement: %w", err)
	}
	p.getAllMetricsStmt = stmt

	return nil
}

func (p *PostgresRepository) Close() error {
	if p.updateCounterStmt != nil {
		p.updateCounterStmt.Close()
	}
	if p.updateGaugeStmt != nil {
		p.updateGaugeStmt.Close()
	}
	if p.getMetricStmt != nil {
		p.getMetricStmt.Close()
	}
	if p.getAllMetricsStmt != nil {
		p.getAllMetricsStmt.Close()
	}

	if p.db != nil {
		p.logger.Info("Closing PostgreSQL database connection")
		return p.db.Close()
	}
	return nil
}

func (p *PostgresRepository) UpdateMetric(ctx context.Context, metric *model.Metrics) error {
	if metric == nil {
		return fmt.Errorf("metric is nil")
	}

	p.logger.Debugf("PostgresRepository.UpdateMetric called for metric: %s", metric.String())

	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			return fmt.Errorf("delta is required for counter metric")
		}

		_, err := p.updateCounterStmt.ExecContext(ctx, metric.ID, metric.MType, *metric.Delta)
		if err != nil {
			return fmt.Errorf("failed to update counter metric: %w", err)
		}

	case model.Gauge:
		if metric.Value == nil {
			return fmt.Errorf("value is required for gauge metric")
		}

		_, err := p.updateGaugeStmt.ExecContext(ctx, metric.ID, metric.MType, *metric.Value)
		if err != nil {
			return fmt.Errorf("failed to update gauge metric: %w", err)
		}

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func (p *PostgresRepository) UpdateMetricsBatch(ctx context.Context, metrics []*model.Metrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics are nil")
	}

	if len(metrics) == 0 {
		return nil
	}

	p.logger.Debugf("PostgresRepository.UpdateMetricsBatch called for %d metrics", len(metrics))

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				p.logger.Errorf("Failed to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	counterQuery := `
		INSERT INTO metrics (id, type, delta, value, updated_at)
		VALUES ($1, $2, $3, NULL, CURRENT_TIMESTAMP)
		ON CONFLICT (id, type)
		DO UPDATE SET
			delta = metrics.delta + EXCLUDED.delta,
			updated_at = CURRENT_TIMESTAMP
	`
	counterStmt, err := tx.PrepareContext(ctx, counterQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare counter statement: %w", err)
	}
	defer counterStmt.Close()

	gaugeQuery := `
		INSERT INTO metrics (id, type, delta, value, updated_at)
		VALUES ($1, $2, NULL, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (id, type)
		DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP
	`
	gaugeStmt, err := tx.PrepareContext(ctx, gaugeQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare gauge statement: %w", err)
	}
	defer gaugeStmt.Close()

	for _, metric := range metrics {
		if metric == nil {
			err = fmt.Errorf("metric is nil in batch")
			return err
		}

		switch metric.MType {
		case model.Counter:
			if metric.Delta == nil {
				err = fmt.Errorf("delta is required for counter metric %s", metric.ID)
				return err
			}

			_, execErr := counterStmt.ExecContext(ctx, metric.ID, metric.MType, *metric.Delta)
			if execErr != nil {
				err = fmt.Errorf("failed to update counter metric %s: %w", metric.ID, execErr)
				return err
			}

		case model.Gauge:
			if metric.Value == nil {
				err = fmt.Errorf("value is required for gauge metric %s", metric.ID)
				return err
			}

			_, execErr := gaugeStmt.ExecContext(ctx, metric.ID, metric.MType, *metric.Value)
			if execErr != nil {
				err = fmt.Errorf("failed to update gauge metric %s: %w", metric.ID, execErr)
				return err
			}

		default:
			err = fmt.Errorf("unknown metric type: %s for metric %s", metric.MType, metric.ID)
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.logger.Debugf("Successfully updated batch of %d metrics", len(metrics))
	return nil
}

func (p *PostgresRepository) GetMetric(ctx context.Context, metric *model.Metrics) (*model.Metrics, error) {
	if metric == nil {
		return nil, fmt.Errorf("metric is nil")
	}

	p.logger.Debugf("PostgresRepository.GetMetric called for metric: %s", metric.String())

	var id, mType string
	var delta sql.NullInt64
	var value sql.NullFloat64

	err := p.getMetricStmt.QueryRowContext(ctx, metric.ID, metric.MType).Scan(&id, &mType, &delta, &value)
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

func (p *PostgresRepository) GetAllMetrics(ctx context.Context) []*model.Metrics {
	p.logger.Debugf("PostgresRepository.GetAllMetrics called")

	rows, err := p.getAllMetricsStmt.QueryContext(ctx)
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

func (p *PostgresRepository) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return p.db.PingContext(ctx)
}
