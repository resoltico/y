package pipeline

import (
	"context"
)

// Common interfaces used across pipeline components
type Logger interface {
	Debug(component string, message string, fields map[string]interface{})
	Info(component string, message string, fields map[string]interface{})
	Warning(component string, message string, fields map[string]interface{})
	Error(component string, err error, fields map[string]interface{})
}

type TimingTracker interface {
	StartTiming(operation string) context.Context
	EndTiming(ctx context.Context)
}
