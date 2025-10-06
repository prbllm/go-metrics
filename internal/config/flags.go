package config

import (
	"flag"
)

func ParseFlags(flagsetName string, args []string, flagErrorHandling flag.ErrorHandling) *Config {
	config := defaultConfig()

	fs := flag.NewFlagSet(flagsetName, flagErrorHandling)

	fs.StringVar(&config.ServerHost, "a", config.ServerHost, "Server address (default: localhost:8080)")
	fs.DurationVar(&config.AgentReportInterval, "r", config.AgentReportInterval, "Agent report interval (default: 10s)")
	fs.DurationVar(&config.AgentPollInterval, "p", config.AgentPollInterval, "Agent poll interval (default: 2s)")

	fs.Parse(args)

	return config
}
