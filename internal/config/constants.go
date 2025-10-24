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

const (
	AddressEnvVar        = "ADDRESS"
	ReportIntervalEnvVar = "REPORT_INTERVAL"
	PollIntervalEnvVar   = "POLL_INTERVAL"
	LogLevelEnvVar       = "LOG_LEVEL"
)

const (
	LogLevelDebug        = "debug"
	LogLevelDebugShort   = "dbg"
	LogLevelInfo         = "info"
	LogLevelWarn         = "warn"
	LogLevelWarning      = "warning"
	LogLevelWarningShort = "wrn"
	LogLevelError        = "error"
	LogLevelErrorShort   = "err"
	LogLevelFatal        = "fatal"
)

const (
	ContentTypeHeader     = "Content-Type"
	ContentEncodingHeader = "Content-Encoding"
	AcceptEncodingHeader  = "Accept-Encoding"
	VaryHeader            = "Vary"
)

const (
	ContentTypeJSON      = "application/json"
	ContentTypeTextPlain = "text/plain"
	ContentEncodingGzip  = "gzip"
)
