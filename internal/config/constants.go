package config

import "time"

const (
	DefaultServerHost = "localhost:8080"
)

const (
	DefaultAgentPollInterval   = 2 * time.Second
	DefaultAgentReportInterval = 10 * time.Second
)

const (
	ValuePath  = "/value"
	UpdatePath = "/update"
	CommonPath = "/"
)

const (
	ServerHostFlag     = "a"
	ReportIntervalFlag = "r"
	PollIntervalFlag   = "p"
)

const (
	ServerHostDescription     = "Server address (default: localhost:8080)"
	ReportIntervalDescription = "Agent report interval in seconds (default: 10)"
	PollIntervalDescription   = "Agent poll interval in seconds (default: 2)"
)
