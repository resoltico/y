package safe

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"gocv.io/x/gocv"
)

// MemoryTracker interface to avoid import cycles
type MemoryTracker interface {
	TrackAllocation(ptr uintptr, size int64, tag string)
	TrackDeallocation(ptr uintptr, tag string)
}

type Mat struct {
	mat        gocv.Mat
	isValid    int32
	refCount   int32
	mu         sync.RWMutex
	id         uint64
	memTracker MemoryTracker
	tag        string
}

var nextMatID uint64

func NewMat(rows, cols int, matType gocv.MatType) (*Mat, error) {
	return NewMatWithTracker(rows, cols, matType, nil, "")
}

func NewMatWithTracker(rows, cols int, matType gocv.MatType, memTracker MemoryTracker, tag string) (*Mat, error) {
	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("invalid dimensions: %dx%d", cols, rows)
	}

	mat := gocv.NewMatWithSize(rows, cols, matType)
	if mat.Empty() {
		mat.Close()
		return nil, fmt.Errorf("failed to create Mat with size %dx%d", cols, rows)
	}

	safeMat := &Mat{
		mat:        mat,
		isValid:    1,
		refCount:   1,
		id:         atomic.AddUint64(&nextMatID, 1),
		memTracker: memTracker,
		tag:        tag,
	}

	if memTracker != nil {
		size := int64(rows * cols * getMatTypeSize(matType))
		ptr := uintptr(unsafe.Pointer(&mat))
		memTracker.TrackAllocation(ptr, size, tag)
	}

	// Set finalizer for cleanup if Close() is not called
	runtime.SetFinalizer(safeMat, (*Mat).finalize)

	return safeMat, nil
}

func NewMatFromMat(srcMat gocv.Mat) (*Mat, error) {
	return NewMatFromMatWithTracker(srcMat, nil, "")
}

func NewMatFromMatWithTracker(srcMat gocv.Mat, memTracker MemoryTracker, tag string) (*Mat, error) {
	if srcMat.Empty() {
		return nil, fmt.Errorf("source Mat is empty")
	}

	if srcMat.Rows() <= 0 || srcMat.Cols() <= 0 {
		return nil, fmt.Errorf("source Mat has invalid dimensions: %dx%d", srcMat.Cols(), srcMat.Rows())
	}

	clonedMat := srcMat.Clone()
	if clonedMat.Empty() {
		clonedMat.Close()
		return nil, fmt.Errorf("failed to clone Mat")
	}

	safeMat := &Mat{
		mat:        clonedMat,
		isValid:    1,
		refCount:   1,
		id:         atomic.AddUint64(&nextMatID, 1),
		memTracker: memTracker,
		tag:        tag,
	}

	if memTracker != nil {
		size := int64(srcMat.Rows() * srcMat.Cols() * getMatTypeSize(srcMat.Type()))
		ptr := uintptr(unsafe.Pointer(&clonedMat))
		memTracker.TrackAllocation(ptr, size, tag)
	}

	runtime.SetFinalizer(safeMat, (*Mat).finalize)

	return safeMat, nil
}

func (sm *Mat) IsValid() bool {
	return atomic.LoadInt32(&sm.isValid) == 1
}

func (sm *Mat) Empty() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return true
	}

	return sm.mat.Empty()
}

func (sm *Mat) Rows() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0
	}

	return sm.mat.Rows()
}

func (sm *Mat) Cols() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0
	}

	return sm.mat.Cols()
}

func (sm *Mat) Channels() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0
	}

	return sm.mat.Channels()
}

func (sm *Mat) Type() gocv.MatType {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return gocv.MatTypeCV8UC1
	}

	return sm.mat.Type()
}

func (sm *Mat) Clone() (*Mat, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return nil, fmt.Errorf("cannot clone invalid Mat")
	}

	if sm.mat.Empty() {
		return nil, fmt.Errorf("cannot clone empty Mat")
	}

	return NewMatFromMatWithTracker(sm.mat, sm.memTracker, sm.tag+"_clone")
}

func (sm *Mat) CopyTo(dst *Mat) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return fmt.Errorf("source Mat is invalid")
	}

	dst.mu.Lock()
	defer dst.mu.Unlock()

	if !dst.IsValid() {
		return fmt.Errorf("destination Mat is invalid")
	}

	if sm.mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	sm.mat.CopyTo(&dst.mat)
	return nil
}

func (sm *Mat) GetUCharAt(row, col int) (uint8, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0, fmt.Errorf("Mat is invalid")
	}

	if row < 0 || row >= sm.mat.Rows() || col < 0 || col >= sm.mat.Cols() {
		return 0, fmt.Errorf("coordinates out of bounds: (%d,%d) for size %dx%d",
			col, row, sm.mat.Cols(), sm.mat.Rows())
	}

	return sm.mat.GetUCharAt(row, col), nil
}

func (sm *Mat) SetUCharAt(row, col int, value uint8) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.IsValid() {
		return fmt.Errorf("Mat is invalid")
	}

	if row < 0 || row >= sm.mat.Rows() || col < 0 || col >= sm.mat.Cols() {
		return fmt.Errorf("coordinates out of bounds: (%d,%d) for size %dx%d",
			col, row, sm.mat.Cols(), sm.mat.Rows())
	}

	sm.mat.SetUCharAt(row, col, value)
	return nil
}

func (sm *Mat) GetUCharAt3(row, col, channel int) (uint8, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0, fmt.Errorf("Mat is invalid")
	}

	if row < 0 || row >= sm.mat.Rows() || col < 0 || col >= sm.mat.Cols() {
		return 0, fmt.Errorf("coordinates out of bounds: (%d,%d) for size %dx%d",
			col, row, sm.mat.Cols(), sm.mat.Rows())
	}

	if channel < 0 || channel >= sm.mat.Channels() {
		return 0, fmt.Errorf("channel out of bounds: %d for %d channels", channel, sm.mat.Channels())
	}

	return sm.mat.GetUCharAt3(row, col, channel), nil
}

func (sm *Mat) SetUCharAt3(row, col, channel int, value uint8) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.IsValid() {
		return fmt.Errorf("Mat is invalid")
	}

	if row < 0 || row >= sm.mat.Rows() || col < 0 || col >= sm.mat.Cols() {
		return fmt.Errorf("coordinates out of bounds: (%d,%d) for size %dx%d",
			col, row, sm.mat.Cols(), sm.mat.Rows())
	}

	if channel < 0 || channel >= sm.mat.Channels() {
		return fmt.Errorf("channel out of bounds: %d for %d channels", channel, sm.mat.Channels())
	}

	sm.mat.SetUCharAt3(row, col, channel, value)
	return nil
}

func (sm *Mat) GetMat() gocv.Mat {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.mat
}

func (sm *Mat) ID() uint64 {
	return sm.id
}

func (sm *Mat) AddRef() {
	atomic.AddInt32(&sm.refCount, 1)
}

func (sm *Mat) Release() {
	if atomic.AddInt32(&sm.refCount, -1) == 0 {
		sm.Close()
	}
}

func (sm *Mat) Close() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if atomic.CompareAndSwapInt32(&sm.isValid, 1, 0) {
		if sm.memTracker != nil {
			ptr := uintptr(unsafe.Pointer(&sm.mat))
			sm.memTracker.TrackDeallocation(ptr, sm.tag)
		}

		if !sm.mat.Empty() {
			sm.mat.Close()
		}

		// Clear finalizer since we're cleaning up manually
		runtime.SetFinalizer(sm, nil)
	}
}

// finalize is called by Go's garbage collector as last resort cleanup
func (sm *Mat) finalize() {
	if atomic.LoadInt32(&sm.isValid) == 1 {
		// Force cleanup if Close() was never called
		sm.Close()
	}
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
	default:
		return 1
	}
}
