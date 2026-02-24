package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func writeTempConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	err := os.WriteFile(path, []byte(content), 0o600)
	require.NoError(t, err)

	return path
}

func TestLoadJSONConfig_Server(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	path := writeTempConfigFile(t, `{
		"address": "json-server:8080",
		"restore": true,
		"store_interval": "42s",
		"store_file": "/tmp/json-metrics.db",
		"database_dsn": "postgres://json:pass@localhost/db",
		"crypto_key": "/tmp/json-server-key.pem"
	}`)

	cfg, err := loadJSONConfig(path, ServerFlagsSet, logger)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "json-server:8080", cfg.ServerHost)
	assert.Equal(t, true, cfg.Restore)
	assert.Equal(t, 42*time.Second, cfg.StoreInterval)
	assert.Equal(t, "/tmp/json-metrics.db", cfg.FileStoragePath)
	assert.Equal(t, "postgres://json:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "/tmp/json-server-key.pem", cfg.CryptoKey)
}

func TestLoadJSONConfig_Agent(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	path := writeTempConfigFile(t, `{
		"address": "json-server:8080",
		"report_interval": "5s",
		"poll_interval": "2s",
		"crypto_key": "/tmp/json-agent-key.pem"
	}`)

	cfg, err := loadJSONConfig(path, AgentFlagsSet, logger)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "json-server:8080", cfg.ServerHost)
	assert.Equal(t, 5*time.Second, cfg.AgentReportInterval)
	assert.Equal(t, 2*time.Second, cfg.AgentPollInterval)
	assert.Equal(t, "/tmp/json-agent-key.pem", cfg.CryptoKey)
}

func TestLoadJSONConfig_InvalidDuration(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	path := writeTempConfigFile(t, `{
		"store_interval": "not-a-duration"
	}`)

	_, err := loadJSONConfig(path, ServerFlagsSet, logger)
	require.Error(t, err)
}

func TestDetectConfigFilePath_Priority(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	t.Run("env_only", func(t *testing.T) {
		t.Setenv("CONFIG", "/env/config.json")
		args := []string{}

		path := detectConfigFilePath(ServerFlagsSet, args, logger)
		assert.Equal(t, "/env/config.json", path)
	})

	t.Run("flag_only", func(t *testing.T) {
		os.Unsetenv("CONFIG")
		args := []string{"-c", "/flag/config.json"}

		path := detectConfigFilePath(ServerFlagsSet, args, logger)
		assert.Equal(t, "/flag/config.json", path)
	})

	t.Run("env_overrides_flag", func(t *testing.T) {
		t.Setenv("CONFIG", "/env/config.json")
		args := []string{"-c", "/flag/config.json"}

		path := detectConfigFilePath(ServerFlagsSet, args, logger)
		assert.Equal(t, "/env/config.json", path)
	})
}

func TestInitConfig_JSON_Flags_Env_Priority_Server(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	jsonPath := writeTempConfigFile(t, `{
		"address": "json-server:8080",
		"restore": true,
		"store_interval": "10s",
		"store_file": "/tmp/json-metrics.db",
		"database_dsn": "postgres://json:pass@localhost/db",
		"crypto_key": "/tmp/json-server-key.pem"
	}`)

	// JSON + flags + env: env > flags > JSON > defaults
	t.Setenv("CONFIG", jsonPath)
	t.Setenv(AddressEnvVar, "env-server:9090")
	t.Setenv(StoreIntervalEnvVar, "20")
	t.Setenv(FileStoragePathEnvVar, "/tmp/env-metrics.db")
	t.Setenv(DatabaseDSNEnvVar, "postgres://env:pass@localhost/db")
	t.Setenv(CryptoKeyEnvVar, "/tmp/env-server-key.pem")

	os.Args = []string{
		"server",
		"-a", "flag-server:8080",
		"-i", "5",
		"-f", "/tmp/flag-metrics.db",
		"-d", "postgres://flag:pass@localhost/db",
		"-crypto-key", "/tmp/flag-server-key.pem",
	}

	err := InitConfig(ServerFlagsSet, logger)
	require.NoError(t, err)

	cfg := GetConfig()

	assert.Equal(t, "env-server:9090", cfg.ServerHost)

	assert.Equal(t, 20*time.Second, cfg.StoreInterval)

	assert.Equal(t, "/tmp/env-metrics.db", cfg.FileStoragePath)

	assert.Equal(t, "postgres://env:pass@localhost/db", cfg.DatabaseDSN)

	assert.Equal(t, "/tmp/env-server-key.pem", cfg.CryptoKey)
}

func TestInitConfig_JSON_Flags_Env_Priority_Agent(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	jsonPath := writeTempConfigFile(t, `{
		"address": "json-server:8080",
		"report_interval": "5s",
		"poll_interval": "2s",
		"crypto_key": "/tmp/json-agent-key.pem"
	}`)

	t.Setenv("CONFIG", jsonPath)
	t.Setenv(AddressEnvVar, "env-server:9090")
	t.Setenv(ReportIntervalEnvVar, "15")
	t.Setenv(PollIntervalEnvVar, "7")
	t.Setenv(CryptoKeyEnvVar, "/tmp/env-agent-key.pem")

	os.Args = []string{
		"agent",
		"-a", "flag-server:8080",
		"-r", "10",
		"-p", "3",
		"-crypto-key", "/tmp/flag-agent-key.pem",
	}

	err := InitConfig(AgentFlagsSet, logger)
	require.NoError(t, err)

	cfg := GetConfig()

	assert.Equal(t, "env-server:9090", cfg.ServerHost)

	assert.Equal(t, 15*time.Second, cfg.AgentReportInterval)

	assert.Equal(t, 7*time.Second, cfg.AgentPollInterval)

	assert.Equal(t, "/tmp/env-agent-key.pem", cfg.CryptoKey)
}
