package migrations

import (
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
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

func TestRunMigrations(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()

	_, err := db.Exec("DROP TABLE IF EXISTS metrics CASCADE")
	require.NoError(t, err, "Failed to drop existing table")

	_, err = db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
	require.NoError(t, err, "Failed to drop schema_migrations table")

	err = RunMigrations(db, logger)
	require.NoError(t, err, "Failed to run migrations")

	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'metrics'
		)
	`).Scan(&exists)
	require.NoError(t, err, "Failed to check table existence")
	require.True(t, exists, "Table 'metrics' should exist")

	rows, err := db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'metrics' 
		ORDER BY column_name
	`)
	require.NoError(t, err, "Failed to query table columns")
	defer rows.Close()

	expectedColumns := map[string]string{
		"id":         "character varying",
		"type":       "character varying",
		"delta":      "bigint",
		"value":      "double precision",
		"updated_at": "timestamp without time zone",
	}

	foundColumns := make(map[string]string)
	for rows.Next() {
		var colName, dataType string
		err := rows.Scan(&colName, &dataType)
		require.NoError(t, err, "Failed to scan column")
		foundColumns[colName] = dataType
	}
	require.NoError(t, rows.Err(), "Error occurred during row iteration")

	require.Equal(t, len(expectedColumns), len(foundColumns), "Column count mismatch")

	for colName, expectedType := range expectedColumns {
		actualType, exists := foundColumns[colName]
		require.True(t, exists, "Column %s should exist", colName)
		require.Equal(t, expectedType, actualType, "Column %s has wrong type", colName)
	}

	var indexExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM pg_indexes 
			WHERE tablename = 'metrics' 
			AND indexname = 'idx_metrics_id_type'
		)
	`).Scan(&indexExists)
	require.NoError(t, err, "Failed to check index existence")
	require.True(t, indexExists, "Index 'idx_metrics_id_type' should exist")
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := getTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	logger := zaptest.NewLogger(t).Sugar()

	_, err := db.Exec("DROP TABLE IF EXISTS metrics CASCADE")
	require.NoError(t, err, "Failed to drop existing table")

	_, err = db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
	require.NoError(t, err, "Failed to drop schema_migrations table")

	err = RunMigrations(db, logger)
	require.NoError(t, err, "Failed to run migrations first time")

	var existsAfterFirst bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'metrics'
		)
	`).Scan(&existsAfterFirst)
	require.NoError(t, err, "Failed to check table existence after first migration")
	require.True(t, existsAfterFirst, "Table 'metrics' should exist after first migration")

	err = RunMigrations(db, logger)
	require.NoError(t, err, "Failed to run migrations second time")

	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'metrics'
		)
	`).Scan(&exists)
	require.NoError(t, err, "Failed to check table existence")
	require.True(t, exists, "Table 'metrics' should still exist after second migration")
}
