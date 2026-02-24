package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/prbllm/go-metrics/internal/logger"
)

type serverJSONConfig struct {
	Address      string `json:"address"`
	Restore      bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile    string `json:"store_file"`
	DatabaseDSN  string `json:"database_dsn"`
	CryptoKey    string `json:"crypto_key"`
}

type agentJSONConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

// loadJSONConfig загружает конфигурацию из JSON-файла и маппит только поддерживаемые поля
// в частично заполненную Config. Остальные поля остаются нулевыми и не должны
// переопределять значения по умолчанию.
func loadJSONConfig(path string, flagsetName string, log logger.Logger) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	cfg := &Config{}

	switch flagsetName {
	case ServerFlagsSet:
		var s serverJSONConfig
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, fmt.Errorf("failed to unmarshal server config JSON: %w", err)
		}

		if s.Address != "" {
			cfg.ServerHost = s.Address
		}

		// Restore: поддерживается только значение true, чтобы не ломать
		// поведение по умолчанию (false) при отсутствии поля.
		if s.Restore {
			cfg.Restore = true
		}

		if s.StoreInterval != "" {
			d, err := time.ParseDuration(s.StoreInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid store_interval value %q: %w", s.StoreInterval, err)
			}
			if d > 0 {
				cfg.StoreInterval = d
			}
		}

		if s.StoreFile != "" {
			cfg.FileStoragePath = s.StoreFile
		}

		if s.DatabaseDSN != "" {
			cfg.DatabaseDSN = s.DatabaseDSN
		}

		if s.CryptoKey != "" {
			cfg.CryptoKey = s.CryptoKey
		}

	case AgentFlagsSet:
		var a agentJSONConfig
		if err := json.Unmarshal(data, &a); err != nil {
			return nil, fmt.Errorf("failed to unmarshal agent config JSON: %w", err)
		}

		if a.Address != "" {
			cfg.ServerHost = a.Address
		}

		if a.ReportInterval != "" {
			d, err := time.ParseDuration(a.ReportInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid report_interval value %q: %w", a.ReportInterval, err)
			}
			if d > 0 {
				cfg.AgentReportInterval = d
			}
		}

		if a.PollInterval != "" {
			d, err := time.ParseDuration(a.PollInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid poll_interval value %q: %w", a.PollInterval, err)
			}
			if d > 0 {
				cfg.AgentPollInterval = d
			}
		}

		if a.CryptoKey != "" {
			cfg.CryptoKey = a.CryptoKey
		}
	default:
		log.Errorf("invalid flagset name for JSON config: %s", flagsetName)
		return nil, fmt.Errorf("invalid flagset name: %s", flagsetName)
	}

	log.Infof("Loaded JSON config from %s for %s", path, flagsetName)
	return cfg, nil
}

