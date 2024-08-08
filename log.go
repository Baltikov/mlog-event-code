package mlog

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattermost/logr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// Very verbose messages for debugging specific issues
	LevelDebug = "debug"
	// Default log level, informational
	LevelInfo = "info"
	// Warnings are messages about possible issues
	LevelWarn = "warn"
	// Errors are messages about things we know are problems
	LevelError = "error"

	// DefaultFlushTimeout is the default amount of time mlog.Flush will wait
	// before timing out.
	DefaultFlushTimeout = time.Second * 5
)

var (
	// disableZap is set when Zap should be disabled and Logr used instead.
	// This is needed for unit testing as Zap has no shutdown capabilities
	// and holds file handles until process exit. Currently unit test create
	// many server instances, and thus many Zap log files.
	// This flag will be removed when Zap is permanently replaced.
	disableZap int32
)

// Type and function aliases from zap to limit the libraries scope into MM code
type Field = zapcore.Field

var Int64 = zap.Int64
var Int32 = zap.Int32
var Int = zap.Int
var Uint32 = zap.Uint32
var String = zap.String
var Any = zap.Any
var Err = zap.Error
var NamedErr = zap.NamedError
var Bool = zap.Bool
var Duration = zap.Duration

type LoggerIFace interface {
	IsLevelEnabled(LogLevel) bool
	Debug(string, int, ...Field)            // Добавлен параметр int для кода события
	Info(string, int, ...Field)             // Добавлен параметр int для кода события
	Warn(string, int, ...Field)             // Добавлен параметр int для кода события
	Error(string, int, ...Field)            // Добавлен параметр int для кода события
	Critical(string, int, ...Field)         // Добавлен параметр int для кода события
	Log(LogLevel, string, int, ...Field)    // Добавлен параметр int для кода события
	LogM([]LogLevel, string, int, ...Field) // Добавлен параметр int для кода события
}

type TargetInfo logr.TargetInfo

type LoggerConfiguration struct {
	EnableConsole bool
	ConsoleJson   bool
	EnableColor   bool
	ConsoleLevel  string
	EnableFile    bool
	FileJson      bool
	FileLevel     string
	FileLocation  string
	FileMaxSizeMB int
	FileCompress  bool
}

type Logger struct {
	zap          *zap.Logger
	consoleLevel zap.AtomicLevel
	fileLevel    zap.AtomicLevel
	logrLogger   *logr.Logger
	mutex        *sync.RWMutex
}

func getZapLevel(level string) zapcore.Level {
	switch level {
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func makeEncoder(json, color bool) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	if json {
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	if color {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func NewLogger(config *LoggerConfiguration) *Logger {
	cores := []zapcore.Core{}
	logger := &Logger{
		consoleLevel: zap.NewAtomicLevelAt(getZapLevel(config.ConsoleLevel)),
		fileLevel:    zap.NewAtomicLevelAt(getZapLevel(config.FileLevel)),
		logrLogger:   newLogr(),
		mutex:        &sync.RWMutex{},
	}

	if config.EnableConsole {
		writer := zapcore.Lock(os.Stderr)
		core := zapcore.NewCore(makeEncoder(config.ConsoleJson, config.EnableColor), writer, logger.consoleLevel)
		cores = append(cores, core)
	}

	if config.EnableFile {
		if atomic.LoadInt32(&disableZap) != 0 {
			t := &LogTarget{
				Type:         "file",
				Format:       "json",
				Levels:       mlogLevelToLogrLevels(config.FileLevel),
				MaxQueueSize: DefaultMaxTargetQueue,
				Options: []byte(fmt.Sprintf(`{"Filename":"%s", "MaxSizeMB":%d, "Compress":%t}`,
					config.FileLocation, config.FileMaxSizeMB, config.FileCompress)),
			}
			if !config.FileJson {
				t.Format = "plain"
			}
			if tgt, err := NewLogrTarget("mlogFile", t); err == nil {
				_ = logger.logrLogger.Logr().AddTarget(tgt)
			} else {
				Error("error creating mlogFile", Err(err))
			}
		} else {
			writer := zapcore.AddSync(&lumberjack.Logger{
				Filename: config.FileLocation,
				MaxSize:  config.FileMaxSizeMB,
				Compress: config.FileCompress,
			})

			core := zapcore.NewCore(makeEncoder(config.FileJson, false), writer, logger.fileLevel)
			cores = append(cores, core)
		}
	}

	combinedCore := zapcore.NewTee(cores...)

	logger.zap = zap.New(combinedCore,
		zap.AddCaller(),
	)
	return logger
}

func (l *Logger) ChangeLevels(config *LoggerConfiguration) {
	l.consoleLevel.SetLevel(getZapLevel(config.ConsoleLevel))
	l.fileLevel.SetLevel(getZapLevel(config.FileLevel))
}

func (l *Logger) SetConsoleLevel(level string) {
	l.consoleLevel.SetLevel(getZapLevel(level))
}

func (l *Logger) With(fields ...Field) *Logger {
	newLogger := *l
	newLogger.zap = newLogger.zap.With(fields...)
	if newLogger.getLogger() != nil {
		ll := newLogger.getLogger().WithFields(zapToLogr(fields))
		newLogger.logrLogger = &ll
	}
	return &newLogger
}

func (l *Logger) StdLog(fields ...Field) *log.Logger {
	return zap.NewStdLog(l.With(fields...).zap.WithOptions(getStdLogOption()))
}

// StdLogAt returns *log.Logger which writes to supplied zap logger at required level.
func (l *Logger) StdLogAt(level string, fields ...Field) (*log.Logger, error) {
	return zap.NewStdLogAt(l.With(fields...).zap.WithOptions(getStdLogOption()), getZapLevel(level))
}

// StdLogWriter returns a writer that can be hooked up to the output of a golang standard logger
// anything written will be interpreted as log entries accordingly
func (l *Logger) StdLogWriter() io.Writer {
	newLogger := *l
	newLogger.zap = newLogger.zap.WithOptions(zap.AddCallerSkip(4), getStdLogOption())
	f := newLogger.Info
	return &loggerWriter{f}
}

func (l *Logger) WithCallerSkip(skip int) *Logger {
	newLogger := *l
	newLogger.zap = newLogger.zap.WithOptions(zap.AddCallerSkip(skip))
	return &newLogger
}

// Made for the plugin interface, wraps mlog in a simpler interface
// at the cost of performance
func (l *Logger) Sugar() *SugarLogger {
	return &SugarLogger{
		wrappedLogger: l,
		zapSugar:      l.zap.Sugar(),
	}
}

func (l *Logger) IsLevelEnabled(level LogLevel) bool {
	return isLevelEnabled(l.getLogger(), logr.Level(level))
}

func (l *Logger) Debug(message string, specificCode int, fields ...Field) { // Добавлен параметр int для кода события
	l.Log(LvlDebug, message, specificCode, fields...)
}

func (l *Logger) Info(message string, specificCode int, fields ...Field) { // Добавлен параметр int для кода события
	l.Log(LvlInfo, message, specificCode, fields...)
}

func (l *Logger) Warn(message string, specificCode int, fields ...Field) { // Добавлен параметр int для кода события
	l.Log(LvlWarn, message, specificCode, fields...)
}

func (l *Logger) Error(message string, specificCode int, fields ...Field) { // Добавлен параметр int для кода события
	l.Log(LvlError, message, specificCode, fields...)
}

func (l *Logger) Critical(message string, specificCode int, fields ...Field) { // Добавлен параметр int для кода события
	l.Log(LvlFatal, message, specificCode, fields...)
}

func (l *Logger) Log(level LogLevel, message string, specificCode int, fields ...Field) { // Добавлен параметр int для кода события
	eventCode := GetEventCode(level, specificCode) // Использование кода события
	logMessage := LogMessage{
		Timestamp: time.Now(),
		Level:     level.Name,
		Code:      eventCode, // Запись кода события
		Message:   message,
	}

	// Логирование с использованием zap
	l.zap.With(fields...).Sugar().Infof("%+v", logMessage)

	// Логирование с использованием logr
	if isLevelEnabled(l.getLogger(), logr.Level(level.ID)) {
		l.getLogger().WithFields(zapToLogr(fields)).Info(message)
	}
}

func (l *Logger) LogM(levels []LogLevel, specificCode int, message string, fields ...Field) { // Добавлен параметр int для кода события
	var logger *logr.Logger
	for _, lvl := range levels {
		if isLevelEnabled(l.getLogger(), logr.Level(lvl)) {
			// don't create logger with fields unless at least one level is active.
			if logger == nil {
				l := l.getLogger().WithFields(zapToLogr(fields))
				logger = &l
			}
			logger.Log(logr.Level(lvl), message)
		}
	}
}

func (l *Logger) Flush(cxt context.Context) error {
	return l.getLogger().Logr().FlushWithTimeout(cxt)
}

// ShutdownAdvancedLogging stops the logger from accepting new log records and tries to
// flush queues within the context timeout. Once complete all targets are shutdown
// and any resources released.
func (l *Logger) ShutdownAdvancedLogging(cxt context.Context) error {
	err := l.getLogger().Logr().ShutdownWithTimeout(cxt)
	l.setLogger(newLogr())
	return err
}

// ConfigAdvancedLoggingConfig (re)configures advanced logging based on the
// specified log targets. This is the easiest way to get the advanced logger
// configured via a config source such as file.
func (l *Logger) ConfigAdvancedLogging(targets LogTargetCfg) error {
	if err := l.ShutdownAdvancedLogging(context.Background()); err != nil {
		Error("error shutting down previous logger", Err(err))
	}

	err := logrAddTargets(l.getLogger(), targets)
	return err
}

// AddTarget adds one or more logr.Target to the advanced logger. This is the preferred method
// to add custom targets or provide configuration that cannot be expressed via a
// config source.
func (l *Logger) AddTarget(targets ...logr.Target) error {
	return l.getLogger().Logr().AddTarget(targets...)
}

// RemoveTargets selectively removes targets that were previously added to this logger instance
// using the passed in filter function. The filter function should return true to remove the target
// and false to keep it.
func (l *Logger) RemoveTargets(ctx context.Context, f func(ti TargetInfo) bool) error {
	// Use locally defined TargetInfo type so we don't spread Logr dependencies.
	fc := func(tic logr.TargetInfo) bool {
		return f(TargetInfo(tic))
	}
	return l.getLogger().Logr().RemoveTargets(ctx, fc)
}

// EnableMetrics enables metrics collection by supplying a MetricsCollector.
// The MetricsCollector provides counters and gauges that are updated by log targets.
func (l *Logger) EnableMetrics(collector logr.MetricsCollector) error {
	return l.getLogger().Logr().SetMetricsCollector(collector)
}

// getLogger is a concurrent safe getter of the logr logger
func (l *Logger) getLogger() *logr.Logger {
	defer l.mutex.RUnlock()
	l.mutex.RLock()
	return l.logrLogger
}

// setLogger is a concurrent safe setter of the logr logger
func (l *Logger) setLogger(logger *logr.Logger) {
	defer l.mutex.Unlock()
	l.mutex.Lock()
	l.logrLogger = logger
}

// DisableZap is called to disable Zap, and Logr will be used instead. Any Logger
// instances created after this call will only use Logr.
//
// This is needed for unit testing as Zap has no shutdown capabilities
// and holds file handles until process exit. Currently unit tests create
// many server instances, and thus many Zap log file handles.
//
// This method will be removed when Zap is permanently replaced.
func DisableZap() {
	atomic.StoreInt32(&disableZap, 1)
}

// EnableZap re-enables Zap such that any Logger instances created after this
// call will allow Zap targets.
func EnableZap() {
	atomic.StoreInt32(&disableZap, 0)
}
