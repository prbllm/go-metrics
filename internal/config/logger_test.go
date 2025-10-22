package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	sugarLogger = nil

	err := InitLogger()
	require.NoError(t, err)

	logger := GetLogger()
	require.NotNil(t, logger)

	logger.Info("Test message for time format")
	logger.Debug("Debug message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	assert.NotNil(t, logger)
}

func TestLoggerLevelFromEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"debug level", LogLevelDebug},
		{"debug short", LogLevelDebugShort},
		{"info level", LogLevelInfo},
		{"warn level", LogLevelWarn},
		{"warning level", LogLevelWarning},
		{"warning short", LogLevelWarningShort},
		{"error level", LogLevelError},
		{"error short", LogLevelErrorShort},
		{"fatal level", LogLevelFatal},
		{"case insensitive", "DEBUG"},
		{"mixed case", "Info"},
		{"unknown level", "unknown"},
		{"empty value", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sugarLogger = nil

			if tt.envValue != "" {
				os.Setenv(LogLevelEnvVar, tt.envValue)
				defer os.Unsetenv(LogLevelEnvVar)
			} else {
				os.Unsetenv(LogLevelEnvVar)
			}

			err := InitLogger()
			require.NoError(t, err)

			logger := GetLogger()
			require.NotNil(t, logger)

			logger.Debug("Debug message")
			logger.Info("Info message")
			logger.Warn("Warning message")
			logger.Error("Error message")
		})
	}
}

func TestInitLoggerMultipleCalls(t *testing.T) {
	sugarLogger = nil

	err := InitLogger()
	require.NoError(t, err)
	logger1 := GetLogger()
	require.NotNil(t, logger1)

	err = InitLogger()
	require.NoError(t, err)
	logger2 := GetLogger()
	require.NotNil(t, logger2)

	assert.Equal(t, logger1, logger2)
}

func TestGetLoggerWithoutInit(t *testing.T) {
	sugarLogger = nil

	logger := GetLogger()
	assert.Nil(t, logger)
}
