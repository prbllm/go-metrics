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
}

var GlobalConfig *Config

func defaultConfig() *Config {
	return &Config{
		ServerHost:          "localhost:8080",
		AgentPollInterval:   2 * time.Second,
		AgentReportInterval: 10 * time.Second,
	}
}

func InitConfig(flagsetName string) error {
	GlobalConfig = ParseFlags(flagsetName, os.Args[1:], flag.ExitOnError)
	return GlobalConfig.Validate()
}

func GetConfig() *Config {
	if GlobalConfig == nil {
		GlobalConfig = defaultConfig()
	}
	return GlobalConfig
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

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{ServerHost: %s, AgentPollInterval: %v, AgentReportInterval: %v}",
		c.ServerHost, c.AgentPollInterval, c.AgentReportInterval)
}
