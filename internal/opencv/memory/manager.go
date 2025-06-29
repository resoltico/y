package memory

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

type Manager struct {
	mu           sync.RWMutex
	logger       logger.Logger
	maxMemory    int64
	usedMemory   int64
	allocCount   int64
	deallocCount int64
	activeMats   map[uint64]*MatInfo

	// Go 1.24 worker pool for memory operations
	workers chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc

	// Performance monitoring
	gcTriggerThreshold int64
	lastGCTime         time.Time
}

type MatInfo struct {
	ID        uint64
	Tag       string
	Size      int64
	Timestamp time.Time
	Type      gocv.MatType
	Rows      int
	Cols      int
}

func NewManager(log logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Use Go 1.24 memory optimization features
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)

	// Modern memory limit calculation for image processing
	systemMemory := int64(memStats.Sys)
	maxMemory := systemMemory * 4 / 10 // 40% of system memory
	if maxMemory < 1024*1024*1024 {
		maxMemory = 1024 * 1024 * 1024 // 1GB minimum
	}
	if maxMemory > 8*1024*1024*1024 {
		maxMemory = 8 * 1024 * 1024 * 1024 // 8GB maximum
	}

	manager := &Manager{
		logger:             log,
		maxMemory:          maxMemory,
		activeMats:         make(map[uint64]*MatInfo),
		workers:            make(chan struct{}, runtime.NumCPU()),
		ctx:                ctx,
		cancel:             cancel,
		gcTriggerThreshold: maxMemory * 75 / 100, // Trigger at 75%
		lastGCTime:         time.Now(),
	}

	// Initialize worker pool
	for i := 0; i < runtime.NumCPU(); i++ {
		manager.workers <- struct{}{}
	}

	go manager.monitorMemoryUsage()

	log.Info("Memory manager initialized", map[string]interface{}{
		"max_memory_gb":    maxMemory / (1024 * 1024 * 1024),
		"gc_trigger_gb":    manager.gcTriggerThreshold / (1024 * 1024 * 1024),
		"system_memory_gb": systemMemory / (1024 * 1024 * 1024),
		"worker_count":     runtime.NumCPU(),
	})

	return manager
}

func (m *Manager) GetMat(rows, cols int, matType gocv.MatType, tag string) (*safe.Mat, error) {
	size := int64(rows * cols * getMatTypeSize(matType))

	m.mu.Lock()
	if m.usedMemory+size > m.maxMemory {
		m.mu.Unlock()
		m.forceGarbageCollection()

		m.mu.Lock()
		if m.usedMemory+size > m.maxMemory {
			m.mu.Unlock()
			return nil, &MemoryExhaustionError{
				Requested: size,
				Available: m.maxMemory - m.usedMemory,
				Total:     m.maxMemory,
			}
		}
	}
	m.mu.Unlock()

	mat, err := safe.NewMatWithTracker(rows, cols, matType, m, tag)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.usedMemory += size
	m.allocCount++
	m.activeMats[mat.ID()] = &MatInfo{
		ID:        mat.ID(),
		Tag:       tag,
		Size:      size,
		Timestamp: time.Now(),
		Type:      matType,
		Rows:      rows,
		Cols:      cols,
	}
	m.mu.Unlock()

	// Async memory pressure check
	if m.usedMemory > m.gcTriggerThreshold {
		go m.asyncGarbageCollection()
	}

	return mat, nil
}

func (m *Manager) ReleaseMat(mat *safe.Mat, tag string) {
	if mat == nil {
		return
	}

	m.mu.Lock()
	if info, exists := m.activeMats[mat.ID()]; exists {
		delete(m.activeMats, mat.ID())
		m.usedMemory -= info.Size
		m.deallocCount++
	}
	m.mu.Unlock()

	mat.Close()
}

func (m *Manager) GetStats() (allocCount, deallocCount int64, usedMemory int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.allocCount, m.deallocCount, m.usedMemory
}

func (m *Manager) monitorMemoryUsage() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performMemoryCheck()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) performMemoryCheck() {
	alloc, dealloc, used := m.GetStats()
	activeCount := len(m.activeMats)
	gocvCount := gocv.MatProfile.Count()

	utilizationRatio := float64(used) / float64(m.maxMemory)

	m.logger.Debug("Memory statistics", map[string]interface{}{
		"allocations":    alloc,
		"deallocations":  dealloc,
		"used_gb":        used / (1024 * 1024 * 1024),
		"max_gb":         m.maxMemory / (1024 * 1024 * 1024),
		"utilization":    utilizationRatio,
		"active_mats":    activeCount,
		"gocv_count":     gocvCount,
		"leak_indicator": alloc - dealloc,
	})

	// Memory pressure warnings
	if utilizationRatio > 0.9 {
		m.logger.Warning("High memory pressure detected", map[string]interface{}{
			"utilization": utilizationRatio,
			"used_gb":     used / (1024 * 1024 * 1024),
		})
		go m.forceGarbageCollection()
	}
}

func (m *Manager) forceGarbageCollection() {
	m.logger.Debug("Forcing garbage collection", nil)

	// Use Go 1.24 memory management features
	runtime.GC()
	runtime.GC() // Double collection for image processing cleanup

	m.mu.Lock()
	m.lastGCTime = time.Now()
	m.mu.Unlock()
}

func (m *Manager) asyncGarbageCollection() {
	select {
	case <-m.workers:
		defer func() { m.workers <- struct{}{} }()

		// Rate limit GC operations
		m.mu.RLock()
		timeSinceLastGC := time.Since(m.lastGCTime)
		m.mu.RUnlock()

		if timeSinceLastGC > 3*time.Second {
			m.forceGarbageCollection()
		}
	default:
		// Worker pool full, skip this GC cycle
	}
}

func (m *Manager) Shutdown() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	matCount := len(m.activeMats)
	totalSize := int64(0)

	for id, info := range m.activeMats {
		totalSize += info.Size
		delete(m.activeMats, id)
	}

	m.logger.Info("Memory manager shutdown", map[string]interface{}{
		"mats_cleaned":        matCount,
		"memory_freed_gb":     totalSize / (1024 * 1024 * 1024),
		"final_gocv_count":    gocv.MatProfile.Count(),
		"total_allocations":   m.allocCount,
		"total_deallocations": m.deallocCount,
	})

	m.usedMemory = 0
	runtime.GC()
}

type MemoryExhaustionError struct {
	Requested int64
	Available int64
	Total     int64
}

func (e *MemoryExhaustionError) Error() string {
	return fmt.Sprintf("memory exhausted: requested %d MB, available %d MB, total %d MB",
		e.Requested/(1024*1024), e.Available/(1024*1024), e.Total/(1024*1024))
}

func getMatTypeSize(matType gocv.MatType) int {
	switch matType {
	case gocv.MatTypeCV8UC1:
		return 1
	case gocv.MatTypeCV8UC3:
		return 3
	case gocv.MatTypeCV8UC4:
		return 4
	case gocv.MatTypeCV16UC1:
		return 2
	case gocv.MatTypeCV16UC3:
		return 6
	case gocv.MatTypeCV32FC1:
		return 4
	case gocv.MatTypeCV32FC3:
		return 12
	case gocv.MatTypeCV64FC1:
		return 8
	case gocv.MatTypeCV64FC3:
		return 24
	default:
		return 4 // Default assumption
	}
}
