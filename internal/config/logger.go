package config

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var sugarLogger *zap.SugaredLogger

func InitLogger() error {
	if sugarLogger != nil {
		return nil
	}

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("02-01-2006 15:04:05.000")
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logLevel, _ := GetEnvironment(LogLevelEnvVar)
	switch strings.ToLower(logLevel) {
	case LogLevelDebug, LogLevelDebugShort:
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case LogLevelInfo:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case LogLevelWarn, LogLevelWarning, LogLevelWarningShort:
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case LogLevelError, LogLevelErrorShort:
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case LogLevelFatal:
		config.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	sugarLogger = logger.Sugar()
	return nil
}

func GetLogger() *zap.SugaredLogger {
	if sugarLogger == nil {
		return nil
	}
	return sugarLogger
}
