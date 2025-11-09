package config

import "time"

const (
	DefaultServerHost = "localhost:8080"
)

const (
	DefaultAgentPollInterval   = 2 * time.Second
	DefaultAgentReportInterval = 10 * time.Second
	DefaultStoreInterval       = 30 * time.Second
	DefaultFileStoragePath     = "metrics.json"
	DefaultRestore             = false
	DefaultDatabaseDSN         = ""
)

const (
	ValuePath   = "/value"
	UpdatePath  = "/update"
	UpdatesPath = "/updates"
	CommonPath  = "/"
	PingPath    = "/ping"
)

const (
	AgentFlagsSet  = "agent"
	ServerFlagsSet = "server"
)

const (
	ServerHostFlag              = "a"
	ReportIntervalOrRestoreFlag = "r"
	PollIntervalFlag            = "p"
	StoreIntervalFlag           = "i"
	FileStoragePathFlag         = "f"
	DatabaseDSNFlag             = "d"
)

const (
	ServerHostDescription      = "Server address (default: localhost:8080)"
	ReportIntervalDescription  = "Agent report interval in seconds (default: 10)"
	PollIntervalDescription    = "Agent poll interval in seconds (default: 2)"
	StoreIntervalDescription   = "Store interval in seconds (default: 30)"
	FileStoragePathDescription = "File storage path (default: metrics.json)"
	RestoreDescription         = "Restore from file storage (true/false, default: false)"
	DatabaseDSNDescription     = "Database connection string (DSN)"
)

const (
	AddressEnvVar         = "ADDRESS"
	ReportIntervalEnvVar  = "REPORT_INTERVAL"
	PollIntervalEnvVar    = "POLL_INTERVAL"
	LogLevelEnvVar        = "LOG_LEVEL"
	StoreIntervalEnvVar   = "STORE_INTERVAL"
	FileStoragePathEnvVar = "FILE_STORAGE_PATH"
	RestoreEnvVar         = "RESTORE"
	DatabaseDSNEnvVar     = "DATABASE_DSN"
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
