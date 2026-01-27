package config

import "time"

const (
	defaultServerHost = "localhost:8080"
)

const (
	defaultAgentPollInterval   = 2 * time.Second
	defaultAgentReportInterval = 10 * time.Second
	defaultStoreInterval       = 30 * time.Second
	defaultFileStoragePath     = "metrics.json"
	defaultRestore             = false
	defaultDatabaseDSN         = ""
	defaultKey                 = ""
	defaultRateLimit           = 10
	HTTPRequestTimeout         = 10 * time.Second
	defaultAuditFile           = ""
	defaultAuditURL            = ""
	AuditEventChannelBuffer    = 100
	AuditWorkerPoolSize        = 5
	PprofPort                  = "6060"
)

const (
	ValuePath   = "/value"
	UpdatePath  = "/update"
	UpdatesPath = "/updates"
	CommonPath  = "/"
	PingPath    = "/ping"
	DebugPath   = "/debug"
	PprofPath   = "/debug/pprof"
)

const (
	AgentFlagsSet  = "agent"
	ServerFlagsSet = "server"
)

const (
	serverHostFlag              = "a"
	reportIntervalOrRestoreFlag = "r"
	pollIntervalFlag            = "p"
	storeIntervalFlag           = "i"
	fileStoragePathFlag         = "f"
	databaseDSNFlag             = "d"
	keyFlag                     = "k"
	rateLimitFlag               = "l"
	auditFileFlag               = "audit-file"
	auditURLFlag                = "audit-url"
	pprofFlag                   = "pprof"
)

const (
	serverHostDescription      = "Server address (default: localhost:8080)"
	reportIntervalDescription  = "Agent report interval in seconds (default: 10)"
	pollIntervalDescription    = "Agent poll interval in seconds (default: 2)"
	storeIntervalDescription   = "Store interval in seconds (default: 30)"
	fileStoragePathDescription = "File storage path (default: metrics.json)"
	restoreDescription         = "Restore from file storage (true/false, default: false)"
	databaseDSNDescription     = "Database connection string (DSN)"
	keyDescription             = "Key for hash"
	rateLimitDescription       = "Rate limit in requests per second (default: 10)"
	auditFileDescription       = "Audit file path (default: empty)"
	auditURLDescription        = "Audit URL (default: empty)"
	pprofDescription           = "Enable pprof endpoints (default: false)"
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
	KeyEnvVar             = "KEY"
	RateLimitEnvVar       = "RATE_LIMIT"
	AuditFileEnvVar       = "AUDIT_FILE"
	AuditURLEnvVar        = "AUDIT_URL"
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
	HashSHA256Header      = "HashSHA256"
)

const (
	ContentTypeJSON      = "application/json"
	ContentTypeTextPlain = "text/plain"
	ContentEncodingGzip  = "gzip"
)

const (
	MaxRetries = 3
	Delay1     = 1 * time.Second
	Delay3     = 3 * time.Second
	Delay5     = 5 * time.Second
)
