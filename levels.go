package mlog

// Standard levels
var (
	LvlPanic = LogLevel{ID: 0, Name: "panic", Stacktrace: true}
	LvlFatal = LogLevel{ID: 1, Name: "fatal", Stacktrace: true}
	LvlError = LogLevel{ID: 2, Name: "error"}
	LvlWarn  = LogLevel{ID: 3, Name: "warn"}
	LvlInfo  = LogLevel{ID: 4, Name: "info"}
	LvlDebug = LogLevel{ID: 5, Name: "debug"}
	LvlTrace = LogLevel{ID: 6, Name: "trace"}
	// used by redirected standard logger
	LvlStdLog = LogLevel{ID: 10, Name: "stdlog"}
	// used only by the logger
	LvlLogError = LogLevel{ID: 11, Name: "logerror", Stacktrace: true}
)

// Register custom (discrete) levels here.
// !!!!! ID's must not exceed 32,768 !!!!!!
var (
	// used by the audit system
	LvlAuditAPI     = LogLevel{ID: 100, Name: "audit-api"}
	LvlAuditContent = LogLevel{ID: 101, Name: "audit-content"}
	LvlAuditPerms   = LogLevel{ID: 102, Name: "audit-permissions"}
	LvlAuditCLI     = LogLevel{ID: 103, Name: "audit-cli"}

	// used by the TCP log target
	LvlTcpLogTarget = LogLevel{ID: 120, Name: "TcpLogTarget"}

	// used by Remote Cluster Service
	LvlRemoteClusterServiceDebug = LogLevel{ID: 130, Name: "RemoteClusterServiceDebug"}
	LvlRemoteClusterServiceError = LogLevel{ID: 131, Name: "RemoteClusterServiceError"}
	LvlRemoteClusterServiceWarn  = LogLevel{ID: 132, Name: "RemoteClusterServiceWarn"}

	// used by Shared Channel Sync Service
	LvlSharedChannelServiceDebug            = LogLevel{ID: 200, Name: "SharedChannelServiceDebug"}
	LvlSharedChannelServiceError            = LogLevel{ID: 201, Name: "SharedChannelServiceError"}
	LvlSharedChannelServiceWarn             = LogLevel{ID: 202, Name: "SharedChannelServiceWarn"}
	LvlSharedChannelServiceMessagesInbound  = LogLevel{ID: 203, Name: "SharedChannelServiceMsgInbound"}
	LvlSharedChannelServiceMessagesOutbound = LogLevel{ID: 204, Name: "SharedChannelServiceMsgOutbound"}

	// add more here ...

	// NEW-COD Коды событий для дальйшего журналирования
	EventCodeInfo             = 1000
	EventCodeAppStarted       = 1001
	EventCodeServiceStarted   = 1002
	EventCodeNetworkConnected = 1003
	EventCodeUserLoggedIn     = 1004
	EventCodeAppStopped       = 1005

	EventCodeWarning              = 2000
	EventCodeMemoryLimit          = 2001
	EventCodeDbConnectionLost     = 2002
	EventCodeLowBattery           = 2003
	EventCodeSlowResponse         = 2004
	EventCodeApiLimitExceeded     = 2005
	EventCodeDnsResolutionProblem = 2006
	EventCodeReconnectionAttempt  = 2007

	EventCodeError                   = 3000
	EventCodePaymentProcessing       = 3001
	EventCodeFileUploadFailed        = 3002
	EventCodeDataSyncError           = 3003
	EventCodeInsufficientPermissions = 3004
	EventCodeInternalServerError     = 3005
	EventCodeJsonParseError          = 3006
	EventCodeUserAuthError           = 3007
	EventCodeExternalServiceTimeout  = 3008
	EventCodeLogWriteError           = 3009
	EventCodeAppInitializationError  = 3010
	EventCodeFileOperationError      = 3011
	EventCodeSystemError             = 3012
)

// Combinations for LogM (log multi)
var (
	MLvlAuditAll = []LogLevel{LvlAuditAPI, LvlAuditContent, LvlAuditPerms, LvlAuditCLI}
)

// NEW-COD
// Функция для получения кода события по уровню и специфичному коду
func GetEventCode(level LogLevel, specificCode int) int {
	if specificCode != 0 {
		return specificCode
	}

	switch level.ID {
	case LvlInfo.ID:
		return EventCodeInfo
	case LvlWarn.ID:
		return EventCodeWarning
	case LvlError.ID:
		return EventCodeError
	default:
		return EventCodeInfo
	}
}
