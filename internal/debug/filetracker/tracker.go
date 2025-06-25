package filetracker

import (
	"runtime"
	"sync"
	"time"
)

type FileInfo struct {
	Path       string
	Handle     uintptr
	OpenedAt   time.Time
	StackTrace []uintptr
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
	openFiles map[string]FileInfo
	mu        sync.RWMutex
	eventBus  EventPublisher
	enabled   bool
}

func NewTracker(eventBus EventPublisher) *Tracker {
	return &Tracker{
		openFiles: make(map[string]FileInfo),
		eventBus:  eventBus,
		enabled:   true,
	}
}

func (ft *Tracker) TrackOpen(path string, handle uintptr) {
	if !ft.enabled {
		return
	}

	ft.mu.Lock()
	defer ft.mu.Unlock()

	var pcs [16]uintptr
	n := runtime.Callers(2, pcs[:])

	info := FileInfo{
		Path:       path,
		Handle:     handle,
		OpenedAt:   time.Now(),
		StackTrace: pcs[:n],
	}

	ft.openFiles[path] = info

	if ft.eventBus != nil {
		ft.eventBus.Publish(Event{
			Type: "file_opened",
			Data: map[string]interface{}{
				"path":   path,
				"handle": handle,
				"time":   info.OpenedAt,
			},
		})
	}
}

func (ft *Tracker) TrackClose(path string, handle uintptr) {
	if !ft.enabled {
		return
	}

	ft.mu.Lock()
	defer ft.mu.Unlock()

	if info, exists := ft.openFiles[path]; exists && info.Handle == handle {
		delete(ft.openFiles, path)

		if ft.eventBus != nil {
			ft.eventBus.Publish(Event{
				Type: "file_closed",
				Data: map[string]interface{}{
					"path":     path,
					"handle":   handle,
					"duration": time.Since(info.OpenedAt),
				},
			})
		}
	}
}

func (ft *Tracker) GetOpenFiles() map[string]FileInfo {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	result := make(map[string]FileInfo)
	for k, v := range ft.openFiles {
		result[k] = v
	}
	return result
}

func (ft *Tracker) DetectLeaks() []FileInfo {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	threshold := time.Now().Add(-5 * time.Minute)
	var leaks []FileInfo

	for _, info := range ft.openFiles {
		if info.OpenedAt.Before(threshold) {
			leaks = append(leaks, info)
		}
	}

	return leaks
}

func (ft *Tracker) SetEnabled(enabled bool) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.enabled = enabled
}
