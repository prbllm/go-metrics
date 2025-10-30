package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger() (*zap.SugaredLogger, error) {
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("02-01-2006 15:04:05.000")
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	zapConfig.DisableStacktrace = false
	zapConfig.DisableCaller = false

	logLevel, _ := os.LookupEnv("LOG_LEVEL")
	switch strings.ToLower(logLevel) {
	case "debug", "d":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info", "i":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn", "warning", "w":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error", "e":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "fatal", "f":
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	zapLogger, err := zapConfig.Build(
		zap.AddStacktrace(zapcore.PanicLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	return zapLogger.Sugar(), nil
}
