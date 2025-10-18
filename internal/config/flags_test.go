package config

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
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
			name: "Server and Agent flags",
			args: []string{"-a", "localhost:8081", "-p", "3", "-r", "12"},
			expected: func() Config {
				cfg := *defaultConfig()
				cfg.ServerHost = "localhost:8081"
				cfg.AgentPollInterval = 3 * time.Second
				cfg.AgentReportInterval = 12 * time.Second
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
			got := ParseFlags("test", tc.args, flag.ContinueOnError)
			expected := tc.expected()
			require.Equal(t, expected.ServerHost, got.ServerHost, "ServerHost is not equal to expected")
			require.Equal(t, expected.AgentPollInterval, got.AgentPollInterval, "AgentPollInterval is not equal to expected")
			require.Equal(t, expected.AgentReportInterval, got.AgentReportInterval, "AgentReportInterval is not equal to expected")
		})
	}
}
