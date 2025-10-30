package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func cleanupEnvironment() {
	envVars := []string{AddressEnvVar, ReportIntervalEnvVar, PollIntervalEnvVar, StoreIntervalEnvVar, FileStoragePathEnvVar, RestoreEnvVar}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}

func TestConfigLoadFromEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedConfig Config
	}{
		{
			name: "all environment variables set",
			envVars: map[string]string{
				AddressEnvVar:         "env-server:9090",
				ReportIntervalEnvVar:  "15",
				PollIntervalEnvVar:    "5",
				StoreIntervalEnvVar:   "60",
				FileStoragePathEnvVar: "/tmp/env-metrics.json",
				RestoreEnvVar:         "true",
			},
			expectedConfig: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: 15 * time.Second,
				AgentPollInterval:   5 * time.Second,
				StoreInterval:       60 * time.Second,
				FileStoragePath:     "/tmp/env-metrics.json",
				Restore:             true,
			},
		},
		{
			name: "partial environment variables set",
			envVars: map[string]string{
				AddressEnvVar: "env-server:9090",
			},
			expectedConfig: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
		},
		{
			name:    "no environment variables set",
			envVars: map[string]string{},
			expectedConfig: Config{
				ServerHost:          DefaultServerHost,
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupEnvironment()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := defaultConfig()
			logger := zaptest.NewLogger(t).Sugar()
			config.loadFromEnvironment(ServerFlagsSet, logger)
			config.loadFromEnvironment(AgentFlagsSet, logger)

			assert.Equal(t, tt.expectedConfig.ServerHost, config.ServerHost, "ServerHost is not equal to expected")
			assert.Equal(t, tt.expectedConfig.AgentReportInterval, config.AgentReportInterval, "AgentReportInterval is not equal to expected")
			assert.Equal(t, tt.expectedConfig.AgentPollInterval, config.AgentPollInterval, "AgentPollInterval is not equal to expected")
			assert.Equal(t, tt.expectedConfig.StoreInterval, config.StoreInterval, "StoreInterval is not equal to expected")
			assert.Equal(t, tt.expectedConfig.FileStoragePath, config.FileStoragePath, "FileStoragePath is not equal to expected")
			assert.Equal(t, tt.expectedConfig.Restore, config.Restore, "Restore is not equal to expected")
		})
	}
}

func TestConfigPriorityAgent(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		flags       []string
		expected    Config
		description string
	}{
		{
			name: "environment overrides flags",
			envVars: map[string]string{
				AddressEnvVar:        "env-server:9090",
				ReportIntervalEnvVar: "15",
				PollIntervalEnvVar:   "5",
			},
			flags: []string{"-a", "flag-server:8080", "-r", "20", "-p", "10"},
			expected: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: 15 * time.Second,
				AgentPollInterval:   5 * time.Second,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
			description: "Environment variables should override command line flags for agent",
		},
		{
			name:    "flags override defaults",
			envVars: map[string]string{},
			flags:   []string{"-a", "flag-server:8080", "-r", "20", "-p", "10"},
			expected: Config{
				ServerHost:          "flag-server:8080",
				AgentReportInterval: 20 * time.Second,
				AgentPollInterval:   10 * time.Second,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
			description: "Command line flags should override defaults when no env vars for agent",
		},
		{
			name:    "defaults when no flags or env",
			envVars: map[string]string{},
			flags:   []string{},
			expected: Config{
				ServerHost:          DefaultServerHost,
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
			description: "Default values when no flags or environment variables for agent",
		},
		{
			name: "mixed priority - env for some, flags for others",
			envVars: map[string]string{
				AddressEnvVar:      "env-server:9090",
				PollIntervalEnvVar: "5",
			},
			flags: []string{"-r", "20"},
			expected: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: 20 * time.Second,
				AgentPollInterval:   5 * time.Second,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
			description: "Mixed priority - environment for address and poll interval, flags for report interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupEnvironment()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			logger := zaptest.NewLogger(t).Sugar()
			config := ParseFlags(AgentFlagsSet, tt.flags, flag.ContinueOnError, logger)

			config.loadFromEnvironment(AgentFlagsSet, logger)

			assert.Equal(t, tt.expected.ServerHost, config.ServerHost,
				"ServerHost: %s", tt.description)
			assert.Equal(t, tt.expected.AgentReportInterval, config.AgentReportInterval,
				"AgentReportInterval: %s", tt.description)
			assert.Equal(t, tt.expected.AgentPollInterval, config.AgentPollInterval,
				"AgentPollInterval: %s", tt.description)
		})
	}
}

func TestConfigPriorityServer(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		flags       []string
		expected    Config
		description string
	}{
		{
			name: "environment overrides flags",
			envVars: map[string]string{
				AddressEnvVar:         "env-server:9090",
				StoreIntervalEnvVar:   "60",
				FileStoragePathEnvVar: "/tmp/env-metrics.json",
				RestoreEnvVar:         "true",
			},
			flags: []string{"-a", "flag-server:8080", "-i", "30", "-f", "/tmp/flag-metrics.json", "-r", "false"},
			expected: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       60 * time.Second,
				FileStoragePath:     "/tmp/env-metrics.json",
				Restore:             true,
			},
			description: "Environment variables should override command line flags for server",
		},
		{
			name:    "flags override defaults",
			envVars: map[string]string{},
			flags:   []string{"-a", "flag-server:8080", "-i", "30", "-f", "/tmp/flag-metrics.json", "-r", "true"},
			expected: Config{
				ServerHost:          "flag-server:8080",
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "/tmp/flag-metrics.json",
				Restore:             true,
			},
			description: "Command line flags should override defaults when no env vars for server",
		},
		{
			name:    "defaults when no flags or env",
			envVars: map[string]string{},
			flags:   []string{},
			expected: Config{
				ServerHost:          DefaultServerHost,
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             DefaultRestore,
			},
			description: "Default values when no flags or environment variables for server",
		},
		{
			name: "mixed priority - env for some, flags for others",
			envVars: map[string]string{
				AddressEnvVar:         "env-server:9090",
				FileStoragePathEnvVar: "/tmp/env-metrics.json",
			},
			flags: []string{"-i", "30", "-r", "true"},
			expected: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "/tmp/env-metrics.json",
				Restore:             true,
			},
			description: "Mixed priority - environment for address and file path, flags for store interval and restore",
		},
		{
			name:    "restore flag without value (should be true)",
			envVars: map[string]string{},
			flags:   []string{"-r"},
			expected: Config{
				ServerHost:          DefaultServerHost,
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
				StoreInterval:       DefaultStoreInterval,
				FileStoragePath:     DefaultFileStoragePath,
				Restore:             true,
			},
			description: "Restore flag without value should be interpreted as true for server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupEnvironment()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			logger := zaptest.NewLogger(t).Sugar()
			config := ParseFlags(ServerFlagsSet, tt.flags, flag.ContinueOnError, logger)

			config.loadFromEnvironment(ServerFlagsSet, logger)

			assert.Equal(t, tt.expected.ServerHost, config.ServerHost,
				"ServerHost: %s", tt.description)
			assert.Equal(t, tt.expected.StoreInterval, config.StoreInterval,
				"StoreInterval: %s", tt.description)
			assert.Equal(t, tt.expected.FileStoragePath, config.FileStoragePath,
				"FileStoragePath: %s", tt.description)
			assert.Equal(t, tt.expected.Restore, config.Restore,
				"Restore: %s", tt.description)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				ServerHost:          "localhost:8080",
				AgentPollInterval:   2 * time.Second,
				AgentReportInterval: 10 * time.Second,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "metrics.json",
				Restore:             false,
			},
			expectError: false,
		},
		{
			name: "empty server host",
			config: Config{
				ServerHost:          "",
				AgentPollInterval:   2 * time.Second,
				AgentReportInterval: 10 * time.Second,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "metrics.json",
				Restore:             false,
			},
			expectError: true,
			errorMsg:    "server host cannot be empty",
		},
		{
			name: "negative poll interval",
			config: Config{
				ServerHost:          "localhost:8080",
				AgentPollInterval:   -1 * time.Second,
				AgentReportInterval: 10 * time.Second,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "metrics.json",
				Restore:             false,
			},
			expectError: true,
			errorMsg:    "agent poll interval must be positive",
		},
		{
			name: "negative report interval",
			config: Config{
				ServerHost:          "localhost:8080",
				AgentPollInterval:   2 * time.Second,
				AgentReportInterval: -1 * time.Second,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "metrics.json",
				Restore:             false,
			},
			expectError: true,
			errorMsg:    "agent report interval must be positive",
		},
		{
			name: "zero poll interval",
			config: Config{
				ServerHost:          "localhost:8080",
				AgentPollInterval:   0,
				AgentReportInterval: 10 * time.Second,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "metrics.json",
				Restore:             false,
			},
			expectError: true,
			errorMsg:    "agent poll interval must be positive",
		},
		{
			name: "negative store interval",
			config: Config{
				ServerHost:          "localhost:8080",
				AgentPollInterval:   2 * time.Second,
				AgentReportInterval: 10 * time.Second,
				StoreInterval:       -1 * time.Second,
				FileStoragePath:     "metrics.json",
				Restore:             false,
			},
			expectError: true,
			errorMsg:    "store interval must be non-negative",
		},
		{
			name: "empty file storage path",
			config: Config{
				ServerHost:          "localhost:8080",
				AgentPollInterval:   2 * time.Second,
				AgentReportInterval: 10 * time.Second,
				StoreInterval:       30 * time.Second,
				FileStoragePath:     "",
				Restore:             false,
			},
			expectError: true,
			errorMsg:    "file storage path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	config := Config{
		ServerHost:          "test-server:8080",
		AgentPollInterval:   5 * time.Second,
		AgentReportInterval: 15 * time.Second,
		StoreInterval:       60 * time.Second,
		FileStoragePath:     "/tmp/test-metrics.json",
		Restore:             true,
	}

	expected := "Config{ServerHost: test-server:8080, AgentPollInterval: 5s, AgentReportInterval: 15s, StoreInterval: 1m0s, FileStoragePath: /tmp/test-metrics.json, Restore: true}"
	actual := config.String()

	assert.Equal(t, expected, actual)
}
