package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarning
	LevelError
)

type StructuredLogger struct {
	writer    io.Writer
	level     Level
	mu        sync.Mutex
	formatter Formatter
}

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

type Formatter interface {
	Format(entry LogEntry) ([]byte, error)
}

type JSONFormatter struct{}

func (f JSONFormatter) Format(entry LogEntry) ([]byte, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

type TextFormatter struct{}

func (f TextFormatter) Format(entry LogEntry) ([]byte, error) {
	line := fmt.Sprintf("%s [%s] %s: %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Level,
		entry.Component,
		entry.Message)

	if entry.Error != "" {
		line += fmt.Sprintf(" error=%s", entry.Error)
	}

	if len(entry.Fields) > 0 {
		fieldsJson, _ := json.Marshal(entry.Fields)
		line += fmt.Sprintf(" fields=%s", string(fieldsJson))
	}

	return []byte(line + "\n"), nil
}

func NewStructuredLogger(writer io.Writer, level Level, useJSON bool) *StructuredLogger {
	var formatter Formatter
	if useJSON {
		formatter = JSONFormatter{}
	} else {
		formatter = TextFormatter{}
	}

	return &StructuredLogger{
		writer:    writer,
		level:     level,
		formatter: formatter,
	}
}

func (sl *StructuredLogger) Info(component string, message string, fields map[string]interface{}) {
	if sl.level <= LevelInfo {
		sl.log(LevelInfo, component, message, "", fields)
	}
}

func (sl *StructuredLogger) Error(component string, err error, fields map[string]interface{}) {
	if sl.level <= LevelError {
		sl.log(LevelError, component, "operation failed", err.Error(), fields)
	}
}

func (sl *StructuredLogger) Warning(component string, message string, fields map[string]interface{}) {
	if sl.level <= LevelWarning {
		sl.log(LevelWarning, component, message, "", fields)
	}
}

func (sl *StructuredLogger) Debug(component string, message string, fields map[string]interface{}) {
	if sl.level <= LevelDebug {
		sl.log(LevelDebug, component, message, "", fields)
	}
}

func (sl *StructuredLogger) log(level Level, component, message, errorMsg string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     levelToString(level),
		Component: component,
		Message:   message,
		Fields:    fields,
		Error:     errorMsg,
	}

	data, err := sl.formatter.Format(entry)
	if err != nil {
		return
	}

	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.writer.Write(data)
}

func levelToString(level Level) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// NoOpLogger provides no-operation implementation for production
type NoOpLogger struct{}

func (n NoOpLogger) Info(component string, message string, fields map[string]interface{})    {}
func (n NoOpLogger) Error(component string, err error, fields map[string]interface{})        {}
func (n NoOpLogger) Warning(component string, message string, fields map[string]interface{}) {}
func (n NoOpLogger) Debug(component string, message string, fields map[string]interface{})   {}
