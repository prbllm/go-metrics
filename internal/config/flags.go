package config

import (
	"flag"
	"time"
)

func ParseFlags(flagsetName string, args []string, flagErrorHandling flag.ErrorHandling) *Config {
	config := defaultConfig()

	fs := flag.NewFlagSet(flagsetName, flagErrorHandling)

	fs.StringVar(&config.ServerHost, "a", config.ServerHost, "Server address (default: localhost:8080)")

	var reportIntervalSec int
	var pollIntervalSec int
	fs.IntVar(&reportIntervalSec, "r", int(config.AgentReportInterval.Seconds()), "Agent report interval in seconds (default: 10)")
	fs.IntVar(&pollIntervalSec, "p", int(config.AgentPollInterval.Seconds()), "Agent poll interval in seconds (default: 2)")

	fs.Parse(args)

	config.AgentReportInterval = time.Duration(reportIntervalSec) * time.Second
	config.AgentPollInterval = time.Duration(pollIntervalSec) * time.Second

	return config
}
