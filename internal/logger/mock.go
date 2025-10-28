package logger

type MockLogger struct{}

func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Debug(msg string) {
}

func (m *MockLogger) Info(msg string) {
}

func (m *MockLogger) Warn(msg string) {
}

func (m *MockLogger) Error(msg string) {
}

func (m *MockLogger) Fatal(msg string) {
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
}

func (m *MockLogger) Sync() error {
	return nil
}
