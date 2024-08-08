package mlog

import (
	"context"
	"log"
	"sync/atomic"

	"github.com/mattermost/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var globalLogger *Logger

var IsLevelEnabled IsLevelEnabledFunc = defaultIsLevelEnabled
var Debug LogFunc = defaultDebugLog
var Info LogFunc = defaultInfoLog
var Warn LogFunc = defaultWarnLog
var Error LogFunc = defaultErrorLog
var Critical LogFunc = defaultCriticalLog
var Log LogFuncCustom = defaultCustomLog
var LogM LogFuncCustomMulti = defaultCustomMultiLog
var Flush FlushFunc = defaultFlush

var ConfigAdvancedLogging ConfigFunc = defaultAdvancedConfig
var ShutdownAdvancedLogging ShutdownFunc = defaultAdvancedShutdown
var AddTarget AddTargetFunc = defaultAddTarget
var RemoveTargets RemoveTargetsFunc = defaultRemoveTargets
var EnableMetrics EnableMetricsFunc = defaultEnableMetrics

type logWriterFunc func([]byte) (int, error)

type IsLevelEnabledFunc func(LogLevel) bool
type LogFunc func(string, ...Field)
type LogFuncCustom func(LogLevel, string, ...Field)
type LogFuncCustomMulti func([]LogLevel, string, ...Field)
type FlushFunc func(context.Context) error
type ConfigFunc func(cfg LogTargetCfg) error
type ShutdownFunc func(context.Context) error
type AddTargetFunc func(...logr.Target) error
type RemoveTargetsFunc func(context.Context, func(TargetInfo) bool) error
type EnableMetricsFunc func(logr.MetricsCollector) error

// Инициализация глобального логгера
func InitGlobalLogger(logger *Logger) {
	// Завершение работы предыдущего экземпляра логгера, если он существует
	if globalLogger != nil && globalLogger.logrLogger != nil {
		globalLogger.logrLogger.Logr().Shutdown()
	}
	glob := *logger
	glob.zap = glob.zap.WithOptions(zap.AddCallerSkip(1))
	globalLogger = &glob

	// Присвоение функций-оберток глобальным переменным
	Debug = debugWrapper
	Info = infoWrapper
	Warn = warnWrapper
	Error = errorWrapper
	Critical = criticalWrapper
	Log = logWrapper
	LogM = globalLogger.LogM
	Flush = globalLogger.Flush
	ConfigAdvancedLogging = globalLogger.ConfigAdvancedLogging
	ShutdownAdvancedLogging = globalLogger.ShutdownAdvancedLogging
	AddTarget = globalLogger.AddTarget
	RemoveTargets = globalLogger.RemoveTargets
	EnableMetrics = globalLogger.EnableMetrics
}

func (lw logWriterFunc) Write(p []byte) (int, error) {
	return lw(p)
}

func RedirectStdLog(logger *Logger) {
	if atomic.LoadInt32(&disableZap) == 0 {
		zap.RedirectStdLogAt(logger.zap.With(zap.String("source", "stdlog")).WithOptions(zap.AddCallerSkip(-2)), zapcore.ErrorLevel)
		return
	}

	writer := func(p []byte) (int, error) {
		Log(LvlStdLog, string(p))
		return len(p), nil
	}
	log.SetOutput(logWriterFunc(writer))
}

func GloballyDisableDebugLogForTest() {
	globalLogger.consoleLevel.SetLevel(zapcore.ErrorLevel)
}

func GloballyEnableDebugLogForTest() {
	globalLogger.consoleLevel.SetLevel(zapcore.DebugLevel)
}

// Функции-обертки для глобальных переменных логгирования
func debugWrapper(message string, fields ...Field) {
	globalLogger.Debug(message, 0, fields...) // 0 - значение specificCode по умолчанию
}

func infoWrapper(message string, fields ...Field) {
	globalLogger.Info(message, 0, fields...) // 0 - значение specificCode по умолчанию
}

func warnWrapper(message string, fields ...Field) {
	globalLogger.Warn(message, 0, fields...) // 0 - значение specificCode по умолчанию
}

func errorWrapper(message string, fields ...Field) {
	globalLogger.Error(message, 0, fields...) // 0 - значение specificCode по умолчанию
}

func criticalWrapper(message string, fields ...Field) {
	globalLogger.Critical(message, 0, fields...) // 0 - значение specificCode по умолчанию
}

func logWrapper(level LogLevel, message string, fields ...Field) {
	globalLogger.Log(level, message, 0, fields...) // 0 - значение specificCode по умолчанию
}
