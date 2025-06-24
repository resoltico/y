package memory

import (
	"fmt"
	"sync"
	"time"

	"gocv.io/x/gocv"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/opencv/safe"
)

type Manager struct {
	pools       map[PoolKey]*Pool
	allocations map[uint64]*AllocationRecord
	mu          sync.RWMutex
	stats       *Stats
	debugMgr    *debug.Manager
}

type PoolKey struct {
	Rows     int
	Cols     int
	MatType  gocv.MatType
}

type AllocationRecord struct {
	Mat        *safe.Mat
	CreatedAt  time.Time
	StackTrace string
	Size       int64
}

type Stats struct {
	TotalAllocated int64
	TotalReleased  int64
	ActiveMats     int64
	PoolHits       int64
	PoolMisses     int64
	MaxAllowed     int64
}

func NewManager(debugMgr *debug.Manager) *Manager {
	return &Manager{
		pools:       make(map[PoolKey]*Pool),
		allocations: make(map[uint64]*AllocationRecord),
		stats: &Stats{
			MaxAllowed: 2 * 1024 * 1024 * 1024, // 2GB limit
		},
		debugMgr: debugMgr,
	}
}

func (m *Manager) GetMat(rows, cols int, matType gocv.MatType) (*safe.Mat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stats.TotalAllocated-m.stats.TotalReleased > m.stats.MaxAllowed {
		return nil, fmt.Errorf("memory limit exceeded: %d bytes allocated", 
			m.stats.TotalAllocated-m.stats.TotalReleased)
	}

	key := PoolKey{Rows: rows, Cols: cols, MatType: matType}
	
	if pool, exists := m.pools[key]; exists {
		if mat := pool.Get(); mat != nil {
			m.stats.PoolHits++
			m.debugMgr.LogInfo("MemoryManager", "Reused Mat from pool")
			return mat, nil
		}
	}

	m.stats.PoolMisses++
	mat, err := safe.NewMat(rows, cols, matType)
	if err != nil {
		return nil, err
	}

	size := int64(rows * cols * m.getMatTypeSize(matType))
	record := &AllocationRecord{
		Mat:        mat,
		CreatedAt:  time.Now(),
		StackTrace: m.captureStackTrace(),
		Size:       size,
	}

	m.allocations[mat.ID()] = record
	m.stats.TotalAllocated += size
	m.stats.ActiveMats++

	m.debugMgr.LogInfo("MemoryManager", "Created new Mat")
	return mat, nil
}

func (m *Manager) ReleaseMat(mat *safe.Mat) {
	if mat == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	id := mat.ID()
	record, exists := m.allocations[id]
	if !exists {
		m.debugMgr.LogWarning("MemoryManager", "Attempting to release untracked Mat")
		mat.Close()
		return
	}

	key := PoolKey{
		Rows:    mat.Rows(),
		Cols:    mat.Cols(),
		MatType: mat.Type(),
	}

	if pool, exists := m.pools[key]; exists {
		if pool.Put(mat) {
			m.debugMgr.LogInfo("MemoryManager", "Added Mat to pool")
			delete(m.allocations, id)
			m.stats.TotalReleased += record.Size
			m.stats.ActiveMats--
			return
		}
	} else {
		pool = NewPool(5) // Max 5 mats per pool
		m.pools[key] = pool
		if pool.Put(mat) {
			m.debugMgr.LogInfo("MemoryManager", "Created new pool for Mat")
			delete(m.allocations, id)
			m.stats.TotalReleased += record.Size
			m.stats.ActiveMats--
			return
		}
	}

	mat.Close()
	delete(m.allocations, id)
	m.stats.TotalReleased += record.Size
	m.stats.ActiveMats--
	m.debugMgr.LogInfo("MemoryManager", "Closed Mat (pool full)")
}

func (m *Manager) GetStats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	statsCopy := *m.stats
	return &statsCopy
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	matCount := 0
	for key, pool := range m.pools {
		matCount += pool.Cleanup()
		delete(m.pools, key)
	}

	for id, record := range m.allocations {
		record.Mat.Close()
		delete(m.allocations, id)
		matCount++
	}

	m.debugMgr.LogInfo("MemoryManager", 
		fmt.Sprintf("Cleaned up %d Mats", matCount))
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

func (m *Manager) captureStackTrace() string {
	// Simplified stack trace capture for debugging
	return "stack trace capture not implemented"
}