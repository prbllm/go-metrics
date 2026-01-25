// Package logger предоставляет интерфейс для логирования.
package logger

// Logger определяет интерфейс для логирования сообщений различных уровней.
type Logger interface {
	// Debug логирует отладочное сообщение.
	Debug(args ...interface{})

	// Info логирует информационное сообщение.
	Info(args ...interface{})

	// Warn логирует предупреждение.
	Warn(args ...interface{})

	// Error логирует ошибку.
	Error(args ...interface{})

	// Fatal логирует критическую ошибку и завершает программу.
	Fatal(args ...interface{})

	// Debugf логирует отформатированное отладочное сообщение.
	Debugf(format string, args ...interface{})

	// Infof логирует отформатированное информационное сообщение.
	Infof(format string, args ...interface{})

	// Warnf логирует отформатированное предупреждение.
	Warnf(format string, args ...interface{})

	// Errorf логирует отформатированную ошибку.
	Errorf(format string, args ...interface{})

	// Fatalf логирует отформатированную критическую ошибку и завершает программу.
	Fatalf(format string, args ...interface{})

	// Sync синхронизирует буферы логгера.
	Sync() error
}
