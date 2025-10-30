package config

import (
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/prbllm/go-metrics/internal/logger"
)

func ParseFlags(flagsetName string, args []string, flagErrorHandling flag.ErrorHandling, logger logger.Logger) *Config {
	config := defaultConfig()

	fs := flag.NewFlagSet(flagsetName, flagErrorHandling)

	fs.StringVar(&config.ServerHost, ServerHostFlag, config.ServerHost, ServerHostDescription)

	switch flagsetName {
	case AgentFlagsSet:
		parseAgentFlags(fs, config, args)
	case ServerFlagsSet:
		parseServerFlags(fs, config, args)
	default:
		logger.Errorf("invalid flagset name: %s", flagsetName)
	}

	return config
}

func parseAgentFlags(fs *flag.FlagSet, config *Config, args []string) {
	var pollIntervalSec int
	var reportIntervalSec int

	fs.IntVar(&reportIntervalSec, ReportIntervalOrRestoreFlag, int(config.AgentReportInterval.Seconds()), ReportIntervalDescription)
	fs.IntVar(&pollIntervalSec, PollIntervalFlag, int(config.AgentPollInterval.Seconds()), PollIntervalDescription)

	fs.Parse(args)

	config.AgentReportInterval = time.Duration(reportIntervalSec) * time.Second
	config.AgentPollInterval = time.Duration(pollIntervalSec) * time.Second
}

func parseServerFlags(fs *flag.FlagSet, config *Config, args []string) {
	var storeIntervalSec int
	var restoreFlag bool
	var restoreStr string

	fs.IntVar(&storeIntervalSec, StoreIntervalFlag, int(config.StoreInterval.Seconds()), StoreIntervalDescription)

	fs.StringVar(&config.FileStoragePath, FileStoragePathFlag, config.FileStoragePath, FileStoragePathDescription)

	fs.BoolVar(&restoreFlag, ReportIntervalOrRestoreFlag, false, RestoreDescription)

	fs.Parse(args)

	config.StoreInterval = time.Duration(storeIntervalSec) * time.Second

	for i, arg := range args {
		if arg == "-"+ReportIntervalOrRestoreFlag && i+1 < len(args) {
			restoreStr = args[i+1]
			break
		}
	}

	if restoreStr != "" {
		config.Restore = parseBoolFlag(restoreStr)
	} else if restoreFlag {
		config.Restore = true
	}
}

func parseBoolFlag(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal != 0
		}
		return false
	}
}
