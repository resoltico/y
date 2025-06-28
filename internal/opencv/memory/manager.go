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
	matPool      sync.Pool
	ctx          context.Context
	cancel       context.CancelFunc

	// Performance monitoring
	gcTriggerThreshold int64
	lastGCTime         time.Time
	gcForceInterval    time.Duration
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

	// Calculate memory limits based on system memory
	var memStats runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStats)

	// Use 30% of system memory as limit, minimum 512MB, maximum 4GB
	systemMemory := int64(memStats.Sys)
	maxMemory := systemMemory * 3 / 10
	if maxMemory < 512*1024*1024 {
		maxMemory = 512 * 1024 * 1024 // 512MB minimum
	}
	if maxMemory > 4*1024*1024*1024 {
		maxMemory = 4 * 1024 * 1024 * 1024 // 4GB maximum
	}

	manager := &Manager{
		logger:             log,
		maxMemory:          maxMemory,
		activeMats:         make(map[uint64]*MatInfo),
		ctx:                ctx,
		cancel:             cancel,
		gcTriggerThreshold: maxMemory * 7 / 10, // Trigger GC at 70% memory usage
		gcForceInterval:    30 * time.Second,   // Force GC every 30 seconds if needed
		lastGCTime:         time.Now(),
		matPool: sync.Pool{
			New: func() interface{} {
				return &safe.Mat{}
			},
		},
	}

	go manager.monitorMemory()

	log.Info("MemoryManager", "initialized with adaptive limits", map[string]interface{}{
		"max_memory_mb":      maxMemory / (1024 * 1024),
		"gc_trigger_mb":      manager.gcTriggerThreshold / (1024 * 1024),
		"system_memory_mb":   systemMemory / (1024 * 1024),
		"gocv_initial_count": gocv.MatProfile.Count(),
	})

	return manager
}

func (m *Manager) GetMat(rows, cols int, matType gocv.MatType, tag string) (*safe.Mat, error) {
	size := int64(rows * cols * getMatTypeSize(matType))

	m.mu.Lock()

	// Check memory limit with aggressive cleanup if needed
	if m.usedMemory+size > m.maxMemory {
		m.mu.Unlock()

		// Force garbage collection and retry
		m.forceGarbageCollection()

		m.mu.Lock()
		if m.usedMemory+size > m.maxMemory {
			m.mu.Unlock()
			return nil, fmt.Errorf("memory limit exceeded: would use %d MB, limit is %d MB, current usage %d MB",
				(m.usedMemory+size)/(1024*1024), m.maxMemory/(1024*1024), m.usedMemory/(1024*1024))
		}
	}
	m.mu.Unlock()

	mat, err := safe.NewMatWithTracker(rows, cols, matType, m, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to create Mat: %w", err)
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

	// Trigger proactive garbage collection if memory usage is high
	if m.usedMemory > m.gcTriggerThreshold {
		go m.proactiveGarbageCollection()
	}

	return mat, nil
}

func (m *Manager) GetPooledMat() *safe.Mat {
	return m.matPool.Get().(*safe.Mat)
}

func (m *Manager) ReturnPooledMat(mat *safe.Mat) {
	if mat != nil {
		mat.Reset()
		m.matPool.Put(mat)
	}
}

func (m *Manager) TrackAllocation(ptr uintptr, size int64, tag string) {
	// Placeholder for future allocation tracking
}

func (m *Manager) TrackDeallocation(ptr uintptr, tag string) {
	m.mu.Lock()
	m.deallocCount++
	m.mu.Unlock()
}

func (m *Manager) ReleaseMat(mat *safe.Mat, tag string) {
	if mat == nil {
		return
	}

	m.mu.Lock()
	if info, exists := m.activeMats[mat.ID()]; exists {
		delete(m.activeMats, mat.ID())
		m.usedMemory -= info.Size

		m.logger.Debug("MemoryManager", "Mat released", map[string]interface{}{
			"mat_id":          mat.ID(),
			"tag":             tag,
			"size_mb":         info.Size / (1024 * 1024),
			"remaining_usage": m.usedMemory / (1024 * 1024),
			"active_count":    len(m.activeMats),
		})
	}
	m.mu.Unlock()

	mat.Close()
}

func (m *Manager) GetUsedMemory() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usedMemory
}

func (m *Manager) GetStats() (allocCount, deallocCount int64, usedMemory int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.allocCount, m.deallocCount, m.usedMemory
}

func (m *Manager) GetActiveMatCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.activeMats)
}

// GetDetailedStats returns comprehensive memory statistics
func (m *Manager) GetDetailedStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate memory usage by type
	memoryByType := make(map[string]int64)
	memoryByTag := make(map[string]int64)
	oldestMat := time.Now()
	newestMat := time.Time{}

	for _, info := range m.activeMats {
		typeName := getMatTypeName(info.Type)
		memoryByType[typeName] += info.Size
		memoryByTag[info.Tag] += info.Size

		if info.Timestamp.Before(oldestMat) {
			oldestMat = info.Timestamp
		}
		if info.Timestamp.After(newestMat) {
			newestMat = info.Timestamp
		}
	}

	return map[string]interface{}{
		"used_memory_mb":       m.usedMemory / (1024 * 1024),
		"max_memory_mb":        m.maxMemory / (1024 * 1024),
		"memory_utilization":   float64(m.usedMemory) / float64(m.maxMemory),
		"total_allocations":    m.allocCount,
		"total_deallocations":  m.deallocCount,
		"active_mats":          len(m.activeMats),
		"gocv_mat_count":       gocv.MatProfile.Count(),
		"memory_by_type":       memoryByType,
		"memory_by_tag":        memoryByTag,
		"oldest_mat_age":       time.Since(oldestMat).Seconds(),
		"gc_trigger_threshold": m.gcTriggerThreshold / (1024 * 1024),
	}
}

func (m *Manager) monitorMemory() {
	ticker := time.NewTicker(15 * time.Second) // More frequent monitoring
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performMonitoringCheck()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) performMonitoringCheck() {
	alloc, dealloc, used := m.GetStats()
	activeCount := m.GetActiveMatCount()
	gocvCount := gocv.MatProfile.Count()

	m.logger.Debug("MemoryManager", "memory statistics", map[string]interface{}{
		"allocations":    alloc,
		"deallocations":  dealloc,
		"used_mb":        used / (1024 * 1024),
		"max_mb":         m.maxMemory / (1024 * 1024),
		"utilization":    float64(used) / float64(m.maxMemory),
		"active_mats":    activeCount,
		"gocv_mat_count": gocvCount,
		"leak_indicator": alloc - dealloc,
	})

	// Warning thresholds
	if gocvCount > 200 {
		m.logger.Warning("MemoryManager", "high GoCV Mat count detected", map[string]interface{}{
			"gocv_mat_count": gocvCount,
			"active_mats":    activeCount,
		})
	}

	if activeCount > 100 {
		m.logOldestMats(5)
	}

	// Memory pressure handling
	utilizationRatio := float64(used) / float64(m.maxMemory)
	if utilizationRatio > 0.8 {
		m.logger.Warning("MemoryManager", "high memory pressure", map[string]interface{}{
			"utilization": utilizationRatio,
			"used_mb":     used / (1024 * 1024),
			"max_mb":      m.maxMemory / (1024 * 1024),
		})

		go m.forceGarbageCollection()
	}

	// Force GC if too much time has passed
	if time.Since(m.lastGCTime) > m.gcForceInterval && utilizationRatio > 0.5 {
		go m.proactiveGarbageCollection()
	}
}

func (m *Manager) logOldestMats(count int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.activeMats) == 0 {
		return
	}

	type matAge struct {
		info *MatInfo
		age  time.Duration
	}

	ages := make([]matAge, 0, len(m.activeMats))
	now := time.Now()

	for _, info := range m.activeMats {
		ages = append(ages, matAge{
			info: info,
			age:  now.Sub(info.Timestamp),
		})
	}

	// Sort by age (descending)
	for i := 0; i < len(ages)-1; i++ {
		for j := i + 1; j < len(ages); j++ {
			if ages[i].age < ages[j].age {
				ages[i], ages[j] = ages[j], ages[i]
			}
		}
	}

	limit := count
	if len(ages) < limit {
		limit = len(ages)
	}

	for i := 0; i < limit; i++ {
		mat := ages[i]
		m.logger.Warning("MemoryManager", "long-lived Mat detected", map[string]interface{}{
			"mat_id":     mat.info.ID,
			"tag":        mat.info.Tag,
			"size_mb":    mat.info.Size / (1024 * 1024),
			"age":        mat.age.String(),
			"dimensions": fmt.Sprintf("%dx%d", mat.info.Cols, mat.info.Rows),
			"type":       getMatTypeName(mat.info.Type),
		})
	}
}

func (m *Manager) forceGarbageCollection() {
	m.logger.Debug("MemoryManager", "forcing garbage collection", nil)

	runtime.GC()
	runtime.GC() // Double GC to ensure cleanup

	m.mu.Lock()
	m.lastGCTime = time.Now()
	m.mu.Unlock()

	// Allow time for GC to complete
	runtime.Gosched()
}

func (m *Manager) proactiveGarbageCollection() {
	// Don't run GC too frequently
	m.mu.RLock()
	timeSinceLastGC := time.Since(m.lastGCTime)
	m.mu.RUnlock()

	if timeSinceLastGC < 5*time.Second {
		return
	}

	m.forceGarbageCollection()
}

func (m *Manager) Shutdown() {
	m.cancel()
	m.Cleanup()
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	matCount := len(m.activeMats)
	totalSize := int64(0)

	for id, info := range m.activeMats {
		totalSize += info.Size
		m.logger.Warning("MemoryManager", "cleaning up unreleased Mat", map[string]interface{}{
			"mat_id":     info.ID,
			"tag":        info.Tag,
			"size_mb":    info.Size / (1024 * 1024),
			"age":        time.Since(info.Timestamp).String(),
			"dimensions": fmt.Sprintf("%dx%d", info.Cols, info.Rows),
			"type":       getMatTypeName(info.Type),
		})
		delete(m.activeMats, id)
	}

	m.logger.Info("MemoryManager", "cleanup completed", map[string]interface{}{
		"mats_cleaned":        matCount,
		"memory_freed_mb":     totalSize / (1024 * 1024),
		"final_gocv_count":    gocv.MatProfile.Count(),
		"final_allocations":   m.allocCount,
		"final_deallocations": m.deallocCount,
	})

	m.usedMemory = 0

	// Final aggressive garbage collection
	runtime.GC()
	runtime.GC()
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
	case gocv.MatTypeCV16UC4:
		return 8
	case gocv.MatTypeCV32FC1:
		return 4
	case gocv.MatTypeCV32FC3:
		return 12
	case gocv.MatTypeCV32FC4:
		return 16
	case gocv.MatTypeCV64FC1:
		return 8
	case gocv.MatTypeCV64FC3:
		return 24
	case gocv.MatTypeCV64FC4:
		return 32
	default:
		return 1
	}
}

func getMatTypeName(matType gocv.MatType) string {
	switch matType {
	case gocv.MatTypeCV8UC1:
		return "CV_8UC1"
	case gocv.MatTypeCV8UC3:
		return "CV_8UC3"
	case gocv.MatTypeCV8UC4:
		return "CV_8UC4"
	case gocv.MatTypeCV16UC1:
		return "CV_16UC1"
	case gocv.MatTypeCV16UC3:
		return "CV_16UC3"
	case gocv.MatTypeCV16UC4:
		return "CV_16UC4"
	case gocv.MatTypeCV32FC1:
		return "CV_32FC1"
	case gocv.MatTypeCV32FC3:
		return "CV_32FC3"
	case gocv.MatTypeCV32FC4:
		return "CV_32FC4"
	case gocv.MatTypeCV64FC1:
		return "CV_64FC1"
	case gocv.MatTypeCV64FC3:
		return "CV_64FC3"
	case gocv.MatTypeCV64FC4:
		return "CV_64FC4"
	default:
		return "UNKNOWN"
	}
}
