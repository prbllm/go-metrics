package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Config struct {
	ServerHost string

	AgentPollInterval   time.Duration
	AgentReportInterval time.Duration

	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
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
	}
}

func InitConfig(flagsetName string) error {
	globalConfig = ParseFlags(flagsetName, os.Args[1:], flag.ExitOnError)
	globalConfig.loadFromEnvironment()
	return globalConfig.Validate()
}

func GetConfig() *Config {
	if globalConfig == nil {
		globalConfig = defaultConfig()
	}
	return globalConfig
}

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

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{ServerHost: %s, AgentPollInterval: %v, AgentReportInterval: %v, StoreInterval: %v, FileStoragePath: %s, Restore: %v}",
		c.ServerHost, c.AgentPollInterval, c.AgentReportInterval, c.StoreInterval, c.FileStoragePath, c.Restore)
}

func (c *Config) loadFromEnvironment() {
	address, err := GetEnvironment(AddressEnvVar)
	if err != nil {
		GetLogger().Warnf("failed to get server host from environment: %v", err)
	} else {
		c.ServerHost = address
	}

	reportInterval, err := GetEnvironmentInt(ReportIntervalEnvVar)
	if err != nil {
		GetLogger().Warnf("failed to get report interval from environment: %v", err)
	} else {
		c.AgentReportInterval = time.Duration(reportInterval) * time.Second
	}

	pollInterval, err := GetEnvironmentInt(PollIntervalEnvVar)
	if err != nil {
		GetLogger().Warnf("failed to get poll interval from environment: %v", err)
	} else {
		c.AgentPollInterval = time.Duration(pollInterval) * time.Second
	}

	storeInterval, err := GetEnvironmentInt(StoreIntervalEnvVar)
	if err != nil {
		GetLogger().Warnf("failed to get store interval from environment: %v", err)
	} else {
		c.StoreInterval = time.Duration(storeInterval) * time.Second
	}

	fileStoragePath, err := GetEnvironment(FileStoragePathEnvVar)
	if err != nil {
		GetLogger().Warnf("failed to get file storage path from environment: %v", err)
	} else {
		c.FileStoragePath = fileStoragePath
	}

	restore, err := GetEnvironment(RestoreEnvVar)
	if err != nil {
		GetLogger().Warnf("failed to get restore from environment: %v", err)
	} else {
		if restore == "true" {
			c.Restore = true
		} else {
			c.Restore = false
		}
	}
}
