package logger

import (
	"io"
	"log/slog"
	"os"
)

type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Warning(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
}

type StructuredLogger struct {
	logger *slog.Logger
	level  LogLevel
}

func NewStructuredLogger(level LogLevel) *StructuredLogger {
	var slogLevel slog.Level
	switch level {
	case DebugLevel:
		slogLevel = slog.LevelDebug
	case InfoLevel:
		slogLevel = slog.LevelInfo
	case WarnLevel:
		slogLevel = slog.LevelWarn
	case ErrorLevel:
		slogLevel = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &StructuredLogger{
		logger: logger,
		level:  level,
	}
}

func NewFileLogger(level LogLevel, writer io.Writer) *StructuredLogger {
	var slogLevel slog.Level
	switch level {
	case DebugLevel:
		slogLevel = slog.LevelDebug
	case InfoLevel:
		slogLevel = slog.LevelInfo
	case WarnLevel:
		slogLevel = slog.LevelWarn
	case ErrorLevel:
		slogLevel = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewJSONHandler(writer, opts)
	logger := slog.New(handler)

	return &StructuredLogger{
		logger: logger,
		level:  level,
	}
}

func (l *StructuredLogger) Debug(msg string, fields map[string]interface{}) {
	if l.level > DebugLevel {
		return
	}
	l.logWithFields(slog.LevelDebug, msg, fields)
}

func (l *StructuredLogger) Info(msg string, fields map[string]interface{}) {
	if l.level > InfoLevel {
		return
	}
	l.logWithFields(slog.LevelInfo, msg, fields)
}

func (l *StructuredLogger) Warning(msg string, fields map[string]interface{}) {
	if l.level > WarnLevel {
		return
	}
	l.logWithFields(slog.LevelWarn, msg, fields)
}

func (l *StructuredLogger) Error(msg string, err error, fields map[string]interface{}) {
	if l.level > ErrorLevel {
		return
	}
	
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	
	l.logWithFields(slog.LevelError, msg, fields)
}

func (l *StructuredLogger) logWithFields(level slog.Level, msg string, fields map[string]interface{}) {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	l.logger.Log(nil, level, msg, args...)
}