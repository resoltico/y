package debug

import (
	"context"
	"os"
	"time"

	"otsu-obliterator/internal/debug/eventbus"
	"otsu-obliterator/internal/debug/filetracker"
	"otsu-obliterator/internal/debug/logger"
	"otsu-obliterator/internal/debug/memtracker"
	"otsu-obliterator/internal/debug/timing"
)

// EventBusImpl wraps eventbus.Bus to implement EventPublisher interface
type EventBusImpl struct {
	*eventbus.Bus
}

func (e *EventBusImpl) Publish(event Event) {
	busEvent := eventbus.Event{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
		Context:   event.Context,
	}
	e.Bus.Publish(busEvent)
}

func (e *EventBusImpl) Subscribe(eventType string, handler EventHandler) {
	busHandler := &eventHandlerAdapter{handler: handler}
	e.Bus.Subscribe(eventType, busHandler)
}

func (e *EventBusImpl) Unsubscribe(eventType string, handler EventHandler) {
	busHandler := &eventHandlerAdapter{handler: handler}
	e.Bus.Unsubscribe(eventType, busHandler)
}

type eventHandlerAdapter struct {
	handler EventHandler
}

func (e *eventHandlerAdapter) Handle(event eventbus.Event) {
	debugEvent := Event{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
		Context:   event.Context,
	}
	e.handler.Handle(debugEvent)
}

func (e *eventHandlerAdapter) GetID() string {
	return e.handler.GetID()
}

// Event bus adapters for different tracker types
type MemTrackerEventBus struct {
	eventBus *EventBusImpl
}

func (m *MemTrackerEventBus) Publish(event memtracker.Event) {
	m.eventBus.Publish(Event{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	})
}

type FileTrackerEventBus struct {
	eventBus *EventBusImpl
}

func (f *FileTrackerEventBus) Publish(event filetracker.Event) {
	f.eventBus.Publish(Event{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	})
}

type TimingTrackerEventBus struct {
	eventBus *EventBusImpl
}

func (t *TimingTrackerEventBus) Publish(event timing.Event) {
	t.eventBus.Publish(Event{
		Type:      event.Type,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	})
}

// Adapter implementations
type TimingTrackerImpl struct {
	tracker *timing.Tracker
}

func (t *TimingTrackerImpl) StartTiming(operation string) context.Context {
	return t.tracker.StartTiming(operation)
}

func (t *TimingTrackerImpl) EndTiming(ctx context.Context) {
	t.tracker.EndTiming(ctx)
}

func (t *TimingTrackerImpl) GetTimings(operation string) []time.Duration {
	return t.tracker.GetTimings(operation)
}

type MemoryTrackerImpl struct {
	tracker *memtracker.Tracker
}

func (m *MemoryTrackerImpl) TrackAllocation(ptr uintptr, size int64, tag string) {
	m.tracker.TrackAllocation(ptr, size, tag)
}

func (m *MemoryTrackerImpl) TrackDeallocation(ptr uintptr, tag string) {
	m.tracker.TrackDeallocation(ptr, tag)
}

func (m *MemoryTrackerImpl) GetAllocations() map[uintptr]AllocationInfo {
	allocations := m.tracker.GetAllocations()
	result := make(map[uintptr]AllocationInfo)
	for k, v := range allocations {
		result[k] = AllocationInfo{
			Size:        v.Size,
			Tag:         v.Tag,
			AllocatedAt: v.AllocatedAt,
			StackTrace:  v.StackTrace,
		}
	}
	return result
}

func (m *MemoryTrackerImpl) GetStats() MemoryStats {
	stats := m.tracker.GetStats()
	return MemoryStats{
		TotalAllocated:   stats.TotalAllocated,
		TotalDeallocated: stats.TotalDeallocated,
		CurrentlyActive:  stats.CurrentlyActive,
		AllocationCount:  stats.AllocationCount,
		LeakCount:        stats.LeakCount,
	}
}

type FileTrackerImpl struct {
	tracker *filetracker.Tracker
}

func (f *FileTrackerImpl) TrackOpen(path string, handle uintptr) {
	f.tracker.TrackOpen(path, handle)
}

func (f *FileTrackerImpl) TrackClose(path string, handle uintptr) {
	f.tracker.TrackClose(path, handle)
}

func (f *FileTrackerImpl) GetOpenFiles() map[string]FileInfo {
	files := f.tracker.GetOpenFiles()
	result := make(map[string]FileInfo)
	for k, v := range files {
		result[k] = FileInfo{
			Path:       v.Path,
			Handle:     v.Handle,
			OpenedAt:   v.OpenedAt,
			StackTrace: v.StackTrace,
		}
	}
	return result
}

func (f *FileTrackerImpl) DetectLeaks() []FileInfo {
	leaks := f.tracker.DetectLeaks()
	result := make([]FileInfo, len(leaks))
	for i, v := range leaks {
		result[i] = FileInfo{
			Path:       v.Path,
			Handle:     v.Handle,
			OpenedAt:   v.OpenedAt,
			StackTrace: v.StackTrace,
		}
	}
	return result
}

type DebugCoordinator struct {
	logger        Logger
	timingTracker TimingTracker
	memoryTracker MemoryTracker
	fileTracker   FileTracker
	eventBus      EventPublisher
}

func NewCoordinator(config Config) *DebugCoordinator {
	bus := eventbus.NewBus(config.EventBufferSize)
	eventBus := &EventBusImpl{Bus: bus}

	var loggerImpl Logger
	if config.EnableLogging {
		loggerImpl = logger.NewStructuredLogger(
			os.Stdout,
			logger.Level(config.LogLevel),
			config.UseJSONLogging,
		)
	} else {
		loggerImpl = logger.NoOpLogger{}
	}

	memEventBus := &MemTrackerEventBus{eventBus: eventBus}
	memTracker := memtracker.NewTracker(memEventBus, config.EnableStackTraces)
	memTracker.SetEnabled(config.EnableMemoryTracking)

	fileEventBus := &FileTrackerEventBus{eventBus: eventBus}
	fileTracker := filetracker.NewTracker(fileEventBus)
	fileTracker.SetEnabled(config.EnableFileTracking)

	timingEventBus := &TimingTrackerEventBus{eventBus: eventBus}
	timingTracker := timing.NewTracker(timingEventBus)
	timingTracker.SetEnabled(config.EnableTimingTracking)

	return &DebugCoordinator{
		logger:        loggerImpl,
		timingTracker: &TimingTrackerImpl{tracker: timingTracker},
		memoryTracker: &MemoryTrackerImpl{tracker: memTracker},
		fileTracker:   &FileTrackerImpl{tracker: fileTracker},
		eventBus:      eventBus,
	}
}

func (dc *DebugCoordinator) Logger() Logger {
	return dc.logger
}

func (dc *DebugCoordinator) TimingTracker() TimingTracker {
	return dc.timingTracker
}

func (dc *DebugCoordinator) MemoryTracker() MemoryTracker {
	return dc.memoryTracker
}

func (dc *DebugCoordinator) FileTracker() FileTracker {
	return dc.fileTracker
}

func (dc *DebugCoordinator) EventPublisher() EventPublisher {
	return dc.eventBus
}

func (dc *DebugCoordinator) Shutdown() {
	if busImpl, ok := dc.eventBus.(*EventBusImpl); ok {
		busImpl.Bus.Shutdown()
	}
}

type Config struct {
	EnableLogging        bool
	EnableMemoryTracking bool
	EnableFileTracking   bool
	EnableTimingTracking bool
	EnableStackTraces    bool
	UseJSONLogging       bool
	LogLevel             int
	EventBufferSize      int
}

func DefaultConfig() Config {
	return Config{
		EnableLogging:        true,
		EnableMemoryTracking: true,
		EnableFileTracking:   true,
		EnableTimingTracking: true,
		EnableStackTraces:    false,
		UseJSONLogging:       false,
		LogLevel:             int(logger.LevelInfo),
		EventBufferSize:      1000,
	}
}

func ProductionConfig() Config {
	return Config{
		EnableLogging:        true,
		EnableMemoryTracking: false,
		EnableFileTracking:   false,
		EnableTimingTracking: false,
		EnableStackTraces:    false,
		UseJSONLogging:       true,
		LogLevel:             int(logger.LevelError),
		EventBufferSize:      100,
	}
}
