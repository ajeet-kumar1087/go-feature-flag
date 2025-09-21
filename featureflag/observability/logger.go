package observability

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]interface{})
	Info(ctx context.Context, msg string, fields map[string]interface{})
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	Error(ctx context.Context, msg string, fields map[string]interface{})
	WithFields(fields map[string]interface{}) Logger
}

// DefaultLogger provides a simple implementation of the Logger interface
type DefaultLogger struct {
	level  LogLevel
	logger *log.Logger
	fields map[string]interface{}
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "[featureflag] ", log.LstdFlags|log.Lshortfile),
		fields: make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.level <= LogLevelDebug {
		l.log(LogLevelDebug, msg, l.mergeFields(fields))
	}
}

// Info logs an info message
func (l *DefaultLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.level <= LogLevelInfo {
		l.log(LogLevelInfo, msg, l.mergeFields(fields))
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.level <= LogLevelWarn {
		l.log(LogLevelWarn, msg, l.mergeFields(fields))
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {
	if l.level <= LogLevelError {
		l.log(LogLevelError, msg, l.mergeFields(fields))
	}
}

// WithFields returns a new logger with additional fields
func (l *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &DefaultLogger{
		level:  l.level,
		logger: l.logger,
		fields: newFields,
	}
}

// log performs the actual logging
func (l *DefaultLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	timestamp := time.Now().Format(time.RFC3339)
	logMsg := fmt.Sprintf("%s [%s] %s", timestamp, level.String(), msg)

	if len(fields) > 0 {
		logMsg += " |"
		for k, v := range fields {
			logMsg += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	l.logger.Println(logMsg)
}

// mergeFields merges the logger's fields with additional fields
func (l *DefaultLogger) mergeFields(fields map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range l.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return merged
}

// NoOpLogger is a logger that does nothing
type NoOpLogger struct{}

// NewNoOpLogger creates a new no-op logger
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{}
}

// Debug does nothing
func (l *NoOpLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {}

// Info does nothing
func (l *NoOpLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {}

// Warn does nothing
func (l *NoOpLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {}

// Error does nothing
func (l *NoOpLogger) Error(ctx context.Context, msg string, fields map[string]interface{}) {}

// WithFields returns the same no-op logger
func (l *NoOpLogger) WithFields(fields map[string]interface{}) Logger {
	return l
}
