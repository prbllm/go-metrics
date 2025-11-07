package config

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestParseAgentFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected func() Config
	}{
		{
			name: "Agent flags",
			args: []string{"-p", "3", "-r", "12"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.AgentPollInterval = 3 * time.Second
				cfg.AgentReportInterval = 12 * time.Second
				return cfg
			},
		},
		{
			name: "Agent flags combined",
			args: []string{"-a", "localhost:8081", "-p", "3", "-r", "12"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.ServerHost = "localhost:8081"
				cfg.AgentPollInterval = 3 * time.Second
				cfg.AgentReportInterval = 12 * time.Second
				return cfg
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseFlags(AgentFlagsSet, tc.args, flag.ContinueOnError, zaptest.NewLogger(t).Sugar())
			expected := tc.expected()
			require.Equal(t, expected.ServerHost, got.ServerHost, "ServerHost is not equal to expected")
			require.Equal(t, expected.AgentPollInterval, got.AgentPollInterval, "AgentPollInterval is not equal to expected")
			require.Equal(t, expected.AgentReportInterval, got.AgentReportInterval, "AgentReportInterval is not equal to expected")
		})
	}
}

func TestParseServerFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected func() Config
	}{
		{
			name: "Server flags",
			args: []string{"-a", "localhost:8081"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.ServerHost = "localhost:8081"
				return cfg
			},
		},
		{
			name: "Store interval flag",
			args: []string{"-i", "60"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.StoreInterval = 60 * time.Second
				return cfg
			},
		},
		{
			name: "File storage path flag",
			args: []string{"-f", "/tmp/metrics.json"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.FileStoragePath = "/tmp/metrics.json"
				return cfg
			},
		},
		{
			name: "Restore flag without value",
			args: []string{"-r"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.Restore = true
				return cfg
			},
		},
		{
			name: "Restore flag false",
			args: []string{"-r", "false"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.Restore = false
				return cfg
			},
		},
		{
			name: "Database DSN flag",
			args: []string{"-d", "postgres://user:pass@localhost/db"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.DatabaseDSN = "postgres://user:pass@localhost/db"
				return cfg
			},
		},
		{
			name: "Server flags combined",
			args: []string{"-a", "localhost:8081", "-i", "60", "-f", "/tmp/metrics.json", "-r", "true", "-d", "postgres://user:pass@localhost/db"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.ServerHost = "localhost:8081"
				cfg.StoreInterval = 60 * time.Second
				cfg.FileStoragePath = "/tmp/metrics.json"
				cfg.Restore = true
				cfg.DatabaseDSN = "postgres://user:pass@localhost/db"
				return cfg
			},
		},
		{
			name: "unknown_flag_rejected",
			args: []string{"-foo"},
			expected: func() Config {
				return *defaultConfig()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseFlags(ServerFlagsSet, tc.args, flag.ContinueOnError, zaptest.NewLogger(t).Sugar())
			expected := tc.expected()
			require.Equal(t, expected.ServerHost, got.ServerHost, "ServerHost is not equal to expected")
			require.Equal(t, expected.StoreInterval, got.StoreInterval, "StoreInterval is not equal to expected")
			require.Equal(t, expected.FileStoragePath, got.FileStoragePath, "FileStoragePath is not equal to expected")
			require.Equal(t, expected.Restore, got.Restore, "Restore is not equal to expected")
			require.Equal(t, expected.DatabaseDSN, got.DatabaseDSN, "DatabaseDSN is not equal to expected")
		})
	}
}
