// Package logger provides logging utilities for the application.
package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	instance *zap.Logger
	sugar    *zap.SugaredLogger
	once     sync.Once
)

// Initialize sets up the zap logger based on verbose level and log file
// verboseLevel can be: debug, info, warn, error, or format:level (e.g., "json:debug", "console:info")
func Initialize(verboseLevel string, logFile string) error {
	var err error
	once.Do(func() {
		// If verbose is not set, create a no-op logger
		if verboseLevel == "" {
			instance = zap.NewNop()
			sugar = instance.Sugar()
			return
		}

		// Parse format and level from verbose flag
		format, level := parseVerboseFlag(verboseLevel)

		// Determine log file path
		if logFile == "" {
			// Default to .gonzo.log in current directory
			pwd, _ := os.Getwd()
			logFile = filepath.Join(pwd, ".gonzo.log")
		}

		// Create encoder config
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Create encoder based on format preference
		var encoder zapcore.Encoder
		if format == "json" {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}

		// Open log file
		file, fileErr := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if fileErr != nil {
			// If we can't open the file, just use stdout
			core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
			instance = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
			sugar = instance.Sugar()
			err = fileErr
			return
		}

		// Create multi-output core (same format for both stdout and file)
		core := zapcore.NewTee(
			// Output to stdout
			zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
			// Output to file
			zapcore.NewCore(encoder, zapcore.AddSync(file), level),
		)

		// Create logger with caller information
		instance = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
		sugar = instance.Sugar()
	})

	return err
}

// parseVerboseFlag parses the verbose flag to extract format and level
// Examples: "debug" -> ("console", DebugLevel)
//           "json:info" -> ("json", InfoLevel)
//           "console:warn" -> ("console", WarnLevel)
func parseVerboseFlag(verboseFlag string) (format string, level zapcore.Level) {
	parts := strings.Split(verboseFlag, ":")

	if len(parts) == 2 {
		// Format explicitly specified
		format = strings.ToLower(parts[0])
		level = parseLevel(parts[1])
	} else {
		// Only level specified, default to console format
		format = "console"
		level = parseLevel(verboseFlag)
	}

	// Validate format
	if format != "json" && format != "console" {
		format = "console"
	}

	return format, level
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		// If any verbose value is provided, default to INFO
		return zapcore.InfoLevel
	}
}

// Get returns the zap logger instance
func Get() *zap.Logger {
	if instance == nil {
		// Initialize with no-op logger if not already initialized
		_ = Initialize("", "")
	}
	return instance
}

// Sugar returns the sugared logger for more convenient logging
func Sugar() *zap.SugaredLogger {
	if sugar == nil {
		// Initialize with no-op logger if not already initialized
		_ = Initialize("", "")
	}
	return sugar
}

// Sync flushes any buffered log entries
func Sync() error {
	if instance != nil {
		return instance.Sync()
	}
	return nil
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Debugf logs a formatted debug message (using sugared logger)
func Debugf(format string, args ...interface{}) {
	Sugar().Debugf(format, args...)
}

// Infof logs a formatted info message (using sugared logger)
func Infof(format string, args ...interface{}) {
	Sugar().Infof(format, args...)
}

// Warnf logs a formatted warning message (using sugared logger)
func Warnf(format string, args ...interface{}) {
	Sugar().Warnf(format, args...)
}

// Errorf logs a formatted error message (using sugared logger)
func Errorf(format string, args ...interface{}) {
	Sugar().Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits (using sugared logger)
func Fatalf(format string, args ...interface{}) {
	Sugar().Fatalf(format, args...)
}

// Printf is a compatibility wrapper for standard log.Printf
func Printf(format string, args ...interface{}) {
	Sugar().Infof(format, args...)
}