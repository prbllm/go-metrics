// Package config предоставляет конфигурацию приложения.
package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/prbllm/go-metrics/internal/logger"
)

// Config содержит конфигурацию сервера и агента метрик.
type Config struct {
	ServerHost string // Адрес сервера

	AgentPollInterval   time.Duration // Интервал сбора метрик агентом
	AgentReportInterval time.Duration // Интервал отправки метрик агентом

	StoreInterval   time.Duration // Интервал сохранения метрик на диск
	FileStoragePath string        // Путь к файлу хранения метрик
	Restore         bool          // Флаг восстановления метрик из файла
	DatabaseDSN     string        // Строка подключения к базе данных
	Key             string        // Ключ для вычисления хеша
	RateLimit       int           // Лимит запросов в секунду
	AuditFile       string        // Путь к файлу аудита
	AuditURL        string        // URL для отправки событий аудита
	PprofEnabled    bool          // Флаг включения pprof эндпоинтов
}

var globalConfig *Config

func defaultConfig() *Config {
	return &Config{
		ServerHost:          DefaultServerHost,
		AgentPollInterval:   DefaultAgentPollInterval,
		AgentReportInterval: DefaultAgentReportInterval,
		StoreInterval:       DefaultStoreInterval,
		FileStoragePath:     DefaultFileStoragePath,
		Restore:             DefaultRestore,
		DatabaseDSN:         DefaultDatabaseDSN,
		Key:                 DefaultKey,
		RateLimit:           DefaultRateLimit,
		AuditFile:           DefaultAuditFile,
		AuditURL:            DefaultAuditURL,
		PprofEnabled:        false,
	}
}

// InitConfig инициализирует глобальную конфигурацию из флагов и переменных окружения.
func InitConfig(flagsetName string, logger logger.Logger) error {
	globalConfig = ParseFlags(flagsetName, os.Args[1:], flag.ExitOnError, logger)
	globalConfig.loadFromEnvironment(flagsetName, logger)
	return globalConfig.Validate()
}

// GetConfig возвращает глобальную конфигурацию.
func GetConfig() *Config {
	if globalConfig == nil {
		globalConfig = defaultConfig()
	}
	return globalConfig
}

// SetConfig устанавливает глобальную конфигурацию.
func SetConfig(config *Config, logger logger.Logger) {
	logger.Infof("Setting config: %v", config.String())
	globalConfig = config
}

// Validate проверяет корректность конфигурации.
func (c *Config) Validate() error {
	if c.ServerHost == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if c.AgentPollInterval <= 0 {
		return fmt.Errorf("agent poll interval must be positive")
	}

	if c.AgentReportInterval <= 0 {
		return fmt.Errorf("agent report interval must be positive")
	}

	if c.StoreInterval < 0 {
		return fmt.Errorf("store interval must be non-negative")
	}

	if c.FileStoragePath == "" {
		return fmt.Errorf("file storage path cannot be empty")
	}

	if c.RateLimit <= 0 {
		return fmt.Errorf("rate limit must be positive")
	}

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{ServerHost: %s, AgentPollInterval: %v, AgentReportInterval: %v, StoreInterval: %v, FileStoragePath: %s, Restore: %v, DatabaseDSN: %s, Key: %s}",
		c.ServerHost, c.AgentPollInterval, c.AgentReportInterval, c.StoreInterval, c.FileStoragePath, c.Restore, c.DatabaseDSN, c.Key)
}

func (c *Config) loadFromEnvironment(flagsetName string, logger logger.Logger) {
	address, err := GetEnvironment(AddressEnvVar)
	if err != nil {
		logger.Warnf("failed to get server host from environment: %v", err)
	} else {
		c.ServerHost = address
	}

	key, err := GetEnvironment(KeyEnvVar)
	if err != nil {
		logger.Warnf("failed to get key from environment: %v", err)
	} else {
		c.Key = key
	}

	switch flagsetName {
	case AgentFlagsSet:
		c.loadAgentEnvironmets(logger)
	case ServerFlagsSet:
		c.loadServerEnvironmets(logger)
	default:
		logger.Errorf("invalid flagset name: %s", flagsetName)
	}
}

func (c *Config) loadAgentEnvironmets(logger logger.Logger) {
	reportInterval, err := GetEnvironmentInt(ReportIntervalEnvVar)
	if err != nil {
		logger.Warnf("failed to get report interval from environment: %v", err)
	} else {
		c.AgentReportInterval = time.Duration(reportInterval) * time.Second
	}

	pollInterval, err := GetEnvironmentInt(PollIntervalEnvVar)
	if err != nil {
		logger.Warnf("failed to get poll interval from environment: %v", err)
	} else {
		c.AgentPollInterval = time.Duration(pollInterval) * time.Second
	}

	rateLimit, err := GetEnvironmentInt(RateLimitEnvVar)
	if err != nil {
		logger.Warnf("failed to get rate limit from environment: %v", err)
	} else {
		c.RateLimit = rateLimit
	}
}

func (c *Config) loadServerEnvironmets(logger logger.Logger) {
	storeInterval, err := GetEnvironmentInt(StoreIntervalEnvVar)
	if err != nil {
		logger.Warnf("failed to get store interval from environment: %v", err)
	} else {
		c.StoreInterval = time.Duration(storeInterval) * time.Second
	}

	fileStoragePath, err := GetEnvironment(FileStoragePathEnvVar)
	if err != nil {
		logger.Warnf("failed to get file storage path from environment: %v", err)
	} else {
		c.FileStoragePath = fileStoragePath
	}

	restore, err := GetEnvironment(RestoreEnvVar)
	if err != nil {
		logger.Warnf("failed to get restore from environment: %v", err)
	} else {
		if restore == "true" {
			c.Restore = true
		} else {
			c.Restore = false
		}
	}

	databaseDSN, err := GetEnvironment(DatabaseDSNEnvVar)
	if err != nil {
		logger.Warnf("failed to get database DSN from environment: %v", err)
	} else {
		c.DatabaseDSN = databaseDSN
	}

	auditFile, err := GetEnvironment(AuditFileEnvVar)
	if err != nil {
		logger.Warnf("failed to get audit file from environment: %v", err)
	} else {
		c.AuditFile = auditFile
	}

	auditURL, err := GetEnvironment(AuditURLEnvVar)
	if err != nil {
		logger.Warnf("failed to get audit URL from environment: %v", err)
	} else {
		c.AuditURL = auditURL
	}
}
