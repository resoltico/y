package debug

import (
	"context"
	"time"
)

// EventPublisher distributes debug events to subscribers without blocking
type EventPublisher interface {
	Publish(event Event)
	Subscribe(eventType string, handler EventHandler)
	Unsubscribe(eventType string, handler EventHandler)
}

// EventHandler processes debug events asynchronously
type EventHandler interface {
	Handle(event Event)
	GetID() string
}

// Event represents a debug event with contextual data
type Event struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
	Context   context.Context
}

// Logger provides structured logging with context
type Logger interface {
	Info(component string, message string, fields map[string]interface{})
	Error(component string, err error, fields map[string]interface{})
	Warning(component string, message string, fields map[string]interface{})
	Debug(component string, message string, fields map[string]interface{})
}

// TimingTracker measures operation performance
type TimingTracker interface {
	StartTiming(operation string) context.Context
	EndTiming(ctx context.Context)
	GetTimings(operation string) []time.Duration
}

// MemoryTracker monitors allocation and deallocation
type MemoryTracker interface {
	TrackAllocation(ptr uintptr, size int64, tag string)
	TrackDeallocation(ptr uintptr, tag string)
	GetAllocations() map[uintptr]AllocationInfo
	GetStats() MemoryStats
}

// FileTracker monitors file handle lifecycle
type FileTracker interface {
	TrackOpen(path string, handle uintptr)
	TrackClose(path string, handle uintptr)
	GetOpenFiles() map[string]FileInfo
	DetectLeaks() []FileInfo
}

// AllocationInfo contains memory allocation details
type AllocationInfo struct {
	Size        int64
	Tag         string
	AllocatedAt time.Time
	StackTrace  []uintptr
}

// MemoryStats provides memory usage summary
type MemoryStats struct {
	TotalAllocated   int64
	TotalDeallocated int64
	CurrentlyActive  int64
	AllocationCount  int64
	LeakCount        int64
}

// FileInfo contains file handle information
type FileInfo struct {
	Path       string
	Handle     uintptr
	OpenedAt   time.Time
	StackTrace []uintptr
}

// Coordinator combines all debug capabilities
type Coordinator interface {
	Logger() Logger
	TimingTracker() TimingTracker
	MemoryTracker() MemoryTracker
	FileTracker() FileTracker
	EventPublisher() EventPublisher
	Shutdown()
}
