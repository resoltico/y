package memtracker

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type AllocationInfo struct {
	Size        int64
	Tag         string
	AllocatedAt time.Time
	StackTrace  []uintptr
}

type MemoryStats struct {
	TotalAllocated   int64
	TotalDeallocated int64
	CurrentlyActive  int64
	AllocationCount  int64
	LeakCount        int64
}

type EventPublisher interface {
	Publish(event Event)
}

type Event struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
}

type Tracker struct {
	allocations  map[uintptr]AllocationInfo
	mu           sync.RWMutex
	eventBus     EventPublisher
	enabled      bool
	stackTraces  bool
	totalAlloc   int64
	totalDealloc int64
	allocCount   int64
	leakCount    int64
}

func NewTracker(eventBus EventPublisher, enableStackTraces bool) *Tracker {
	return &Tracker{
		allocations: make(map[uintptr]AllocationInfo),
		eventBus:    eventBus,
		enabled:     true,
		stackTraces: enableStackTraces,
	}
}

func (mt *Tracker) TrackAllocation(ptr uintptr, size int64, tag string) {
	if !mt.enabled {
		return
	}

	atomic.AddInt64(&mt.totalAlloc, size)
	atomic.AddInt64(&mt.allocCount, 1)

	info := AllocationInfo{
		Size:        size,
		Tag:         tag,
		AllocatedAt: time.Now(),
	}

	if mt.stackTraces {
		var pcs [32]uintptr
		n := runtime.Callers(3, pcs[:])
		info.StackTrace = pcs[:n]
	}

	mt.mu.Lock()
	mt.allocations[ptr] = info
	mt.mu.Unlock()

	if mt.eventBus != nil {
		mt.eventBus.Publish(Event{
			Type: "memory_allocated",
			Data: map[string]interface{}{
				"ptr":  ptr,
				"size": size,
				"tag":  tag,
				"time": info.AllocatedAt,
			},
		})
	}
}

func (mt *Tracker) TrackDeallocation(ptr uintptr, tag string) {
	if !mt.enabled {
		return
	}

	mt.mu.Lock()
	info, exists := mt.allocations[ptr]
	if exists {
		delete(mt.allocations, ptr)
		atomic.AddInt64(&mt.totalDealloc, info.Size)
	} else {
		atomic.AddInt64(&mt.leakCount, 1)
	}
	mt.mu.Unlock()

	if mt.eventBus != nil {
		eventData := map[string]interface{}{
			"ptr": ptr,
			"tag": tag,
		}

		if exists {
			eventData["size"] = info.Size
			eventData["lifetime"] = time.Since(info.AllocatedAt)
			mt.eventBus.Publish(Event{
				Type: "memory_deallocated",
				Data: eventData,
			})
		} else {
			mt.eventBus.Publish(Event{
				Type: "memory_untracked_deallocation",
				Data: eventData,
			})
		}
	}
}

func (mt *Tracker) GetAllocations() map[uintptr]AllocationInfo {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	result := make(map[uintptr]AllocationInfo)
	for k, v := range mt.allocations {
		result[k] = v
	}
	return result
}

func (mt *Tracker) GetStats() MemoryStats {
	mt.mu.RLock()
	currentlyActive := int64(len(mt.allocations))
	mt.mu.RUnlock()

	return MemoryStats{
		TotalAllocated:   atomic.LoadInt64(&mt.totalAlloc),
		TotalDeallocated: atomic.LoadInt64(&mt.totalDealloc),
		CurrentlyActive:  currentlyActive,
		AllocationCount:  atomic.LoadInt64(&mt.allocCount),
		LeakCount:        atomic.LoadInt64(&mt.leakCount),
	}
}

func (mt *Tracker) SetEnabled(enabled bool) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.enabled = enabled
}

func (mt *Tracker) SetStackTracingEnabled(enabled bool) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.stackTraces = enabled
}

func (mt *Tracker) DetectLeaks(olderThan time.Duration) []AllocationInfo {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	threshold := time.Now().Add(-olderThan)
	var leaks []AllocationInfo

	for _, info := range mt.allocations {
		if info.AllocatedAt.Before(threshold) {
			leaks = append(leaks, info)
		}
	}

	return leaks
}

func (mt *Tracker) GetAllocationsByTag(tag string) []AllocationInfo {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	var result []AllocationInfo
	for _, info := range mt.allocations {
		if info.Tag == tag {
			result = append(result, info)
		}
	}

	return result
}
