package versions

import "github.com/prbllm/go-metrics/internal/logger"

func defaultNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func LogBuildInfo(version string, date string, commit string, logger logger.Logger) {
	if logger == nil {
		return
	}
	logger.Infof("Build version: %s", defaultNA(version))
	logger.Infof("Build date: %s", defaultNA(date))
	logger.Infof("Build commit: %s", defaultNA(commit))
}
