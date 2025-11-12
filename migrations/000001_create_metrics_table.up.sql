CREATE TABLE IF NOT EXISTS metrics (
    id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, type)
);

CREATE INDEX IF NOT EXISTS idx_metrics_id_type ON metrics(id, type);

