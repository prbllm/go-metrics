package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	logger *zap.SugaredLogger
}

func NewZapLogger() (*ZapLogger, error) {
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

	return &ZapLogger{
		logger: zapLogger.Sugar(),
	}, nil
}

func (z *ZapLogger) Debug(msg string) {
	z.logger.Debug(msg)
}

func (z *ZapLogger) Info(msg string) {
	z.logger.Info(msg)
}

func (z *ZapLogger) Warn(msg string) {
	z.logger.Warn(msg)
}

func (z *ZapLogger) Error(msg string) {
	z.logger.Error(msg)
}

func (z *ZapLogger) Fatal(msg string) {
	z.logger.Fatal(msg)
}

func (z *ZapLogger) Debugf(format string, args ...interface{}) {
	z.logger.Debugf(format, args...)
}

func (z *ZapLogger) Infof(format string, args ...interface{}) {
	z.logger.Infof(format, args...)
}

func (z *ZapLogger) Warnf(format string, args ...interface{}) {
	z.logger.Warnf(format, args...)
}

func (z *ZapLogger) Errorf(format string, args ...interface{}) {
	z.logger.Errorf(format, args...)
}

func (z *ZapLogger) Fatalf(format string, args ...interface{}) {
	z.logger.Fatalf(format, args...)
}

func (z *ZapLogger) Sync() error {
	return z.logger.Sync()
}
