package repository

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/prbllm/go-metrics/internal/logger"
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
	p.logger.Debugf("PostgresRepository.UpdateMetric called for metric: %s", metric.String())
	return nil
}

func (p *PostgresRepository) GetMetric(metric *model.Metrics) (*model.Metrics, error) {
	p.logger.Debugf("PostgresRepository.GetMetric called for metric: %s", metric.String())
	return nil, fmt.Errorf("not implemented")
}

func (p *PostgresRepository) GetAllMetrics() []*model.Metrics {
	p.logger.Debugf("PostgresRepository.GetAllMetrics called")
	return []*model.Metrics{}
}

func (p *PostgresRepository) Ping() error {
	if p.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return p.db.Ping()
}
