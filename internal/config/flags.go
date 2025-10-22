package config

import (
	"flag"
	"time"
)

func ParseFlags(flagsetName string, args []string, flagErrorHandling flag.ErrorHandling) *Config {
	config := defaultConfig()

	fs := flag.NewFlagSet(flagsetName, flagErrorHandling)

	fs.StringVar(&config.ServerHost, ServerHostFlag, config.ServerHost, ServerHostDescription)

	var reportIntervalSec int
	var pollIntervalSec int
	fs.IntVar(&reportIntervalSec, ReportIntervalFlag, int(config.AgentReportInterval.Seconds()), ReportIntervalDescription)
	fs.IntVar(&pollIntervalSec, PollIntervalFlag, int(config.AgentPollInterval.Seconds()), PollIntervalDescription)

	fs.Parse(args)

	config.AgentReportInterval = time.Duration(reportIntervalSec) * time.Second
	config.AgentPollInterval = time.Duration(pollIntervalSec) * time.Second

	return config
}
