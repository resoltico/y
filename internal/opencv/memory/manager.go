package memory

import (
	"fmt"
	"sync"
	"time"

	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

type Manager struct {
	pools        map[PoolKey]*Pool
	mu           sync.RWMutex
	logger       logger.Logger
	maxMemory    int64
	usedMemory   int64
	allocCount   int64
	deallocCount int64
}

type PoolKey struct {
	Rows    int
	Cols    int
	MatType gocv.MatType
}

func NewManager(log logger.Logger) *Manager {
	return &Manager{
		pools:     make(map[PoolKey]*Pool),
		logger:    log,
		maxMemory: 2 * 1024 * 1024 * 1024, // 2GB limit
	}
}

func (m *Manager) GetMat(rows, cols int, matType gocv.MatType, tag string) (*safe.Mat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	size := int64(rows * cols * m.getMatTypeSize(matType))

	if m.usedMemory+size > m.maxMemory {
		return nil, fmt.Errorf("memory limit exceeded: would use %d bytes, limit is %d",
			m.usedMemory+size, m.maxMemory)
	}

	key := PoolKey{Rows: rows, Cols: cols, MatType: matType}

	if pool, exists := m.pools[key]; exists {
		if mat := pool.Get(); mat != nil {
			m.logger.Debug("MemoryManager", "reused Mat from pool", map[string]interface{}{
				"tag":  tag,
				"size": size,
			})
			return mat, nil
		}
	}

	mat, err := safe.NewMatWithTracker(rows, cols, matType, m, tag)
	if err != nil {
		return nil, err
	}

	m.usedMemory += size
	m.allocCount++

	m.logger.Debug("MemoryManager", "created new Mat", map[string]interface{}{
		"tag":          tag,
		"size":         size,
		"total_allocs": m.allocCount,
	})

	return mat, nil
}

func (m *Manager) TrackAllocation(ptr uintptr, size int64, tag string) {
	// Tracking is handled in Mat allocation
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
	defer m.mu.Unlock()

	size := int64(mat.Rows() * mat.Cols() * m.getMatTypeSize(mat.Type()))

	key := PoolKey{
		Rows:    mat.Rows(),
		Cols:    mat.Cols(),
		MatType: mat.Type(),
	}

	if pool, exists := m.pools[key]; exists {
		if pool.Put(mat) {
			m.logger.Debug("MemoryManager", "returned Mat to pool", map[string]interface{}{
				"tag":  tag,
				"size": size,
			})
			m.usedMemory -= size
			return
		}
	} else {
		pool = NewPool(5) // Max 5 mats per pool
		m.pools[key] = pool
		if pool.Put(mat) {
			m.logger.Debug("MemoryManager", "created new pool and stored Mat", map[string]interface{}{
				"tag":  tag,
				"size": size,
			})
			m.usedMemory -= size
			return
		}
	}

	mat.Close()
	m.usedMemory -= size

	m.logger.Debug("MemoryManager", "closed Mat directly", map[string]interface{}{
		"tag":    tag,
		"size":   size,
		"reason": "pool_full",
	})
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

func (m *Manager) MonitorMemory() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			alloc, dealloc, used := m.GetStats()

			m.logger.Debug("MemoryManager", "memory stats", map[string]interface{}{
				"allocations":   alloc,
				"deallocations": dealloc,
				"used_bytes":    used,
				"gocv_count":    gocv.MatProfile.Count(),
			})

			// Warn if potential leak detected
			if gocv.MatProfile.Count() > 100 {
				m.logger.Warning("MemoryManager", "potential memory leak detected", map[string]interface{}{
					"mat_count": gocv.MatProfile.Count(),
				})
			}
		}
	}()
}

func (m *Manager) Shutdown() {
	m.logger.Info("MemoryManager", "shutdown initiated", nil)
	m.Cleanup()
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	matCount := 0
	for key, pool := range m.pools {
		matCount += pool.Cleanup()
		delete(m.pools, key)
	}

	m.logger.Info("MemoryManager", "cleanup completed", map[string]interface{}{
		"mats_cleaned": matCount,
		"final_count":  gocv.MatProfile.Count(),
	})

	m.usedMemory = 0
}

func (m *Manager) getMatTypeSize(matType gocv.MatType) int {
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
	default:
		return 1
	}
}
