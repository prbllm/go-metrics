package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanupEnvironment() {
	envVars := []string{AddressEnvVar, ReportIntervalEnvVar, PollIntervalEnvVar}
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
				AddressEnvVar:        "env-server:9090",
				ReportIntervalEnvVar: "15",
				PollIntervalEnvVar:   "5",
			},
			expectedConfig: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: 15 * time.Second,
				AgentPollInterval:   5 * time.Second,
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
			},
		},
		{
			name:    "no environment variables set",
			envVars: map[string]string{},
			expectedConfig: Config{
				ServerHost:          DefaultServerHost,
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
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
			config.loadFromEnvironment()

			assert.Equal(t, tt.expectedConfig.ServerHost, config.ServerHost, "ServerHost is not equal to expected")
			assert.Equal(t, tt.expectedConfig.AgentReportInterval, config.AgentReportInterval, "AgentReportInterval is not equal to expected")
			assert.Equal(t, tt.expectedConfig.AgentPollInterval, config.AgentPollInterval, "AgentPollInterval is not equal to expected")
		})
	}
}

func TestConfigPriority(t *testing.T) {
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
			},
			description: "Environment variables should override command line flags",
		},
		{
			name:    "flags override defaults",
			envVars: map[string]string{},
			flags:   []string{"-a", "flag-server:8080", "-r", "20", "-p", "10"},
			expected: Config{
				ServerHost:          "flag-server:8080",
				AgentReportInterval: 20 * time.Second,
				AgentPollInterval:   10 * time.Second,
			},
			description: "Command line flags should override defaults when no env vars",
		},
		{
			name:    "defaults when no flags or env",
			envVars: map[string]string{},
			flags:   []string{},
			expected: Config{
				ServerHost:          DefaultServerHost,
				AgentReportInterval: DefaultAgentReportInterval,
				AgentPollInterval:   DefaultAgentPollInterval,
			},
			description: "Default values when no flags or environment variables",
		},
		{
			name: "mixed priority - env for some, flags for others",
			envVars: map[string]string{
				AddressEnvVar: "env-server:9090",
			},
			flags: []string{"-r", "20", "-p", "10"},
			expected: Config{
				ServerHost:          "env-server:9090",
				AgentReportInterval: 20 * time.Second,
				AgentPollInterval:   10 * time.Second,
			},
			description: "Mixed priority - environment for address, flags for intervals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupEnvironment()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			config := ParseFlags("test", tt.flags, flag.ContinueOnError)

			config.loadFromEnvironment()

			assert.Equal(t, tt.expected.ServerHost, config.ServerHost,
				"ServerHost: %s", tt.description)
			assert.Equal(t, tt.expected.AgentReportInterval, config.AgentReportInterval,
				"AgentReportInterval: %s", tt.description)
			assert.Equal(t, tt.expected.AgentPollInterval, config.AgentPollInterval,
				"AgentPollInterval: %s", tt.description)
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
			},
			expectError: false,
		},
		{
			name: "empty server host",
			config: Config{
				ServerHost:          "",
				AgentPollInterval:   2 * time.Second,
				AgentReportInterval: 10 * time.Second,
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
			},
			expectError: true,
			errorMsg:    "agent poll interval must be positive",
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
	}

	expected := "Config{ServerHost: test-server:8080, AgentPollInterval: 5s, AgentReportInterval: 15s}"
	actual := config.String()

	assert.Equal(t, expected, actual)
}
