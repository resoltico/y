package timing

import (
	"context"
	"sync"
	"time"
)

const timingKey = "timing_start"

type EventPublisher interface {
	Publish(event Event)
}

type Event struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
}

type TimingInfo struct {
	Operation string
	StartTime time.Time
}

type Tracker struct {
	timings  map[string][]time.Duration
	mu       sync.RWMutex
	eventBus EventPublisher
	enabled  bool
}

func NewTracker(eventBus EventPublisher) *Tracker {
	return &Tracker{
		timings:  make(map[string][]time.Duration),
		eventBus: eventBus,
		enabled:  true,
	}
}

func (tt *Tracker) StartTiming(operation string) context.Context {
	if !tt.enabled {
		return context.Background()
	}

	start := time.Now()

	ctx := context.WithValue(context.Background(), timingKey, TimingInfo{
		Operation: operation,
		StartTime: start,
	})

	if tt.eventBus != nil {
		tt.eventBus.Publish(Event{
			Type: "timing_started",
			Data: map[string]interface{}{
				"operation": operation,
				"start":     start,
			},
		})
	}

	return ctx
}

func (tt *Tracker) EndTiming(ctx context.Context) {
	if !tt.enabled {
		return
	}

	timingInfo, ok := ctx.Value(timingKey).(TimingInfo)
	if !ok {
		return
	}

	duration := time.Since(timingInfo.StartTime)

	tt.mu.Lock()
	if tt.timings[timingInfo.Operation] == nil {
		tt.timings[timingInfo.Operation] = make([]time.Duration, 0)
	}
	tt.timings[timingInfo.Operation] = append(tt.timings[timingInfo.Operation], duration)
	tt.mu.Unlock()

	if tt.eventBus != nil {
		tt.eventBus.Publish(Event{
			Type: "timing_completed",
			Data: map[string]interface{}{
				"operation": timingInfo.Operation,
				"duration":  duration,
				"start":     timingInfo.StartTime,
				"end":       time.Now(),
			},
		})
	}
}

func (tt *Tracker) GetTimings(operation string) []time.Duration {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	timings := tt.timings[operation]
	if timings == nil {
		return nil
	}

	result := make([]time.Duration, len(timings))
	copy(result, timings)
	return result
}

func (tt *Tracker) GetAllTimings() map[string][]time.Duration {
	tt.mu.RLock()
	defer tt.mu.RUnlock()

	result := make(map[string][]time.Duration)
	for operation, timings := range tt.timings {
		result[operation] = make([]time.Duration, len(timings))
		copy(result[operation], timings)
	}
	return result
}

func (tt *Tracker) GetAverageTime(operation string) time.Duration {
	timings := tt.GetTimings(operation)
	if len(timings) == 0 {
		return 0
	}

	var total time.Duration
	for _, duration := range timings {
		total += duration
	}

	return total / time.Duration(len(timings))
}

func (tt *Tracker) SetEnabled(enabled bool) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.enabled = enabled
}

func (tt *Tracker) Reset(operation string) {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	if operation == "" {
		tt.timings = make(map[string][]time.Duration)
	} else {
		delete(tt.timings, operation)
	}
}
