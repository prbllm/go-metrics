package config

import (
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/prbllm/go-metrics/internal/logger"
)

func ParseFlags(flagsetName string, args []string, flagErrorHandling flag.ErrorHandling, logger logger.Logger) *Config {
	logger.Infof("Parsing flags for %s, flags: %v", flagsetName, args)

	config := defaultConfig()
	fs := flag.NewFlagSet(flagsetName, flagErrorHandling)

	fs.StringVar(&config.ServerHost, serverHostFlag, config.ServerHost, serverHostDescription)
	fs.StringVar(&config.Key, keyFlag, config.Key, keyDescription)

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

	fs.IntVar(&reportIntervalSec, reportIntervalOrRestoreFlag, int(config.AgentReportInterval.Seconds()), reportIntervalDescription)
	fs.IntVar(&pollIntervalSec, pollIntervalFlag, int(config.AgentPollInterval.Seconds()), pollIntervalDescription)
	fs.IntVar(&config.RateLimit, rateLimitFlag, config.RateLimit, rateLimitDescription)

	fs.Parse(args)

	config.AgentReportInterval = time.Duration(reportIntervalSec) * time.Second
	config.AgentPollInterval = time.Duration(pollIntervalSec) * time.Second
}

func parseServerFlags(fs *flag.FlagSet, config *Config, args []string) {
	var storeIntervalSec int
	var restoreFlag bool
	var restoreStr string

	processedArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-"+reportIntervalOrRestoreFlag && i+1 < len(args) {
			nextArg := args[i+1]
			if nextArg == "true" || nextArg == "false" || nextArg == "1" || nextArg == "0" {
				restoreStr = nextArg
				processedArgs = append(processedArgs, args[i])
				i++
				continue
			}
		}
		processedArgs = append(processedArgs, args[i])
	}

	fs.IntVar(&storeIntervalSec, storeIntervalFlag, int(config.StoreInterval.Seconds()), storeIntervalDescription)

	fs.StringVar(&config.FileStoragePath, fileStoragePathFlag, config.FileStoragePath, fileStoragePathDescription)

	fs.BoolVar(&restoreFlag, reportIntervalOrRestoreFlag, false, restoreDescription)

	fs.StringVar(&config.DatabaseDSN, databaseDSNFlag, config.DatabaseDSN, databaseDSNDescription)

	fs.StringVar(&config.AuditFile, auditFileFlag, config.AuditFile, auditFileDescription)

	fs.StringVar(&config.AuditURL, auditURLFlag, config.AuditURL, auditURLDescription)

	fs.BoolVar(&config.PprofEnabled, pprofFlag, false, pprofDescription)

	fs.Parse(processedArgs)

	config.StoreInterval = time.Duration(storeIntervalSec) * time.Second

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
