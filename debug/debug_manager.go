package debug

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

type Manager struct {
	mu                sync.RWMutex
	timings          map[string][]time.Duration
	memoryStats      runtime.MemStats
	lastMemoryUpdate time.Time
}

func NewManager() *Manager {
	return &Manager{
		timings: make(map[string][]time.Duration),
	}
}

func (dm *Manager) StartTiming(operation string) time.Time {
	return time.Now()
}

func (dm *Manager) EndTiming(operation string, startTime time.Time) {
	duration := time.Since(startTime)
	
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	if dm.timings[operation] == nil {
		dm.timings[operation] = make([]time.Duration, 0)
	}
	dm.timings[operation] = append(dm.timings[operation], duration)
	
	LogPerformance(operation, duration)
}

func (dm *Manager) LogInfo(component string, message string) {
	log.Printf("[INFO] %s: %s", component, message)
}

func (dm *Manager) LogError(component string, err error) {
	log.Printf("[ERROR] %s: %v", component, err)
}

func (dm *Manager) LogWarning(component string, message string) {
	log.Printf("[WARN] %s: %s", component, message)
}

func (dm *Manager) GetMemoryStats() string {
	dm.updateMemoryStats()
	
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	return fmt.Sprintf(`Memory Statistics:
- Allocated: %.2f MB
- Total Allocated: %.2f MB
- System Memory: %.2f MB
- Garbage Collections: %d
- Goroutines: %d`,
		float64(dm.memoryStats.Alloc)/1024/1024,
		float64(dm.memoryStats.TotalAlloc)/1024/1024,
		float64(dm.memoryStats.Sys)/1024/1024,
		dm.memoryStats.NumGC,
		runtime.NumGoroutine())
}

func (dm *Manager) GetPerformanceReport() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	
	report := "Performance Report:\n"
	
	for operation, timings := range dm.timings {
		if len(timings) == 0 {
			continue
		}
		
		var total time.Duration
		min := timings[0]
		max := timings[0]
		
		for _, timing := range timings {
			total += timing
			if timing < min {
				min = timing
			}
			if timing > max {
				max = timing
			}
		}
		
		avg := total / time.Duration(len(timings))
		
		report += fmt.Sprintf("- %s: count=%d, avg=%v, min=%v, max=%v\n",
			operation, len(timings), avg, min, max)
	}
	
	return report
}

func (dm *Manager) updateMemoryStats() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	// Update memory stats at most once per second
	if time.Since(dm.lastMemoryUpdate) > time.Second {
		runtime.ReadMemStats(&dm.memoryStats)
		dm.lastMemoryUpdate = time.Now()
	}
}

func (dm *Manager) Cleanup() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	
	dm.timings = make(map[string][]time.Duration)
	LogInfo("Debug", "Debug manager cleaned up")
}