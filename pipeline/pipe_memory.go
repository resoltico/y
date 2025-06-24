package pipeline

import (
	"fmt"
	"sync"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

type MemoryManager struct {
	mu           sync.Mutex
	matPool      map[string][]gocv.Mat
	debugManager *debug.Manager
}

func NewMemoryManager(debugManager *debug.Manager) *MemoryManager {
	return &MemoryManager{
		matPool:      make(map[string][]gocv.Mat),
		debugManager: debugManager,
	}
}

func (mm *MemoryManager) GetMat(rows, cols int, matType gocv.MatType) gocv.Mat {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	key := mm.getPoolKey(rows, cols, matType)

	// Try to reuse from pool
	if mats, exists := mm.matPool[key]; exists && len(mats) > 0 {
		mat := mats[len(mats)-1]
		mm.matPool[key] = mats[:len(mats)-1]
		mm.debugManager.LogInfo("MemoryManager", "Reused Mat from pool")
		return mat
	}

	// Create new Mat
	mat := gocv.NewMatWithSize(rows, cols, matType)
	mm.debugManager.LogInfo("MemoryManager", "Created new Mat")
	return mat
}

func (mm *MemoryManager) ReleaseMat(mat gocv.Mat) {
	if mat.Empty() {
		return
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	key := mm.getPoolKey(mat.Rows(), mat.Cols(), mat.Type())

	// Add to pool if not too many
	if mats, exists := mm.matPool[key]; exists {
		if len(mats) < 5 { // Limit pool size
			mm.matPool[key] = append(mats, mat)
			mm.debugManager.LogInfo("MemoryManager", "Added Mat to pool")
			return
		}
	} else {
		mm.matPool[key] = []gocv.Mat{mat}
		mm.debugManager.LogInfo("MemoryManager", "Created new pool for Mat")
		return
	}

	// Pool full, close immediately
	mat.Close()
	mm.debugManager.LogInfo("MemoryManager", "Closed Mat (pool full)")
}

func (mm *MemoryManager) getPoolKey(rows, cols int, matType gocv.MatType) string {
	return fmt.Sprintf("%d_%d_%d", rows, cols, int(matType))
}

func (mm *MemoryManager) Cleanup() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	matCount := 0
	for key, mats := range mm.matPool {
		for _, mat := range mats {
			mat.Close()
			matCount++
		}
		delete(mm.matPool, key)
	}

	mm.debugManager.LogInfo("MemoryManager",
		fmt.Sprintf("Cleaned up %d pooled Mats", matCount))
}
