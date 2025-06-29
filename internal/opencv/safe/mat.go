package safe

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"gocv.io/x/gocv"
)

type MemoryTracker interface {
	ReleaseMat(mat *Mat, tag string)
}

type Mat struct {
	mat        gocv.Mat
	isValid    atomic.Bool
	refCount   atomic.Int64
	mu         sync.RWMutex
	id         uint64
	memTracker MemoryTracker
	tag        string
}

var (
	nextMatID atomic.Uint64
	matPool   = sync.Pool{
		New: func() interface{} {
			return &Mat{}
		},
	}
)

func NewMat(rows, cols int, matType gocv.MatType) (*Mat, error) {
	return NewMatWithTracker(rows, cols, matType, nil, "")
}

func NewMatWithTracker(rows, cols int, matType gocv.MatType, memTracker MemoryTracker, tag string) (*Mat, error) {
	if err := validateDimensions(rows, cols); err != nil {
		return nil, err
	}

	mat := gocv.NewMatWithSize(rows, cols, matType)
	if mat.Empty() {
		mat.Close()
		return nil, fmt.Errorf("failed to create Mat with dimensions %dx%d type %d", cols, rows, int(matType))
	}

	safeMat := matPool.Get().(*Mat)
	*safeMat = Mat{
		mat:        mat,
		id:         nextMatID.Add(1),
		memTracker: memTracker,
		tag:        tag,
	}
	safeMat.isValid.Store(true)
	safeMat.refCount.Store(1)

	// Use Go 1.24 cleanup patterns
	runtime.SetFinalizer(safeMat, (*Mat).finalize)
	return safeMat, nil
}

func NewMatFromMat(srcMat gocv.Mat) (*Mat, error) {
	return NewMatFromMatWithTracker(srcMat, nil, "")
}

func NewMatFromMatWithTracker(srcMat gocv.Mat, memTracker MemoryTracker, tag string) (*Mat, error) {
	if err := validateSourceMat(srcMat); err != nil {
		return nil, err
	}

	clonedMat := srcMat.Clone()
	if clonedMat.Empty() {
		clonedMat.Close()
		return nil, fmt.Errorf("failed to clone Mat")
	}

	safeMat := matPool.Get().(*Mat)
	*safeMat = Mat{
		mat:        clonedMat,
		id:         nextMatID.Add(1),
		memTracker: memTracker,
		tag:        tag,
	}
	safeMat.isValid.Store(true)
	safeMat.refCount.Store(1)

	runtime.SetFinalizer(safeMat, (*Mat).finalize)
	return safeMat, nil
}

func (sm *Mat) IsValid() bool {
	return sm.isValid.Load()
}

func (sm *Mat) Empty() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return !sm.IsValid() || sm.mat.Empty()
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

	if !sm.IsValid() || sm.mat.Empty() {
		return nil, fmt.Errorf("cannot clone invalid or empty Mat")
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

	sm.mat.CopyTo(&dst.mat)
	return nil
}

func (sm *Mat) GetUCharAt(row, col int) (uint8, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if err := sm.validateCoordinates(row, col); err != nil {
		return 0, err
	}

	return sm.mat.GetUCharAt(row, col), nil
}

func (sm *Mat) SetUCharAt(row, col int, value uint8) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.validateCoordinates(row, col); err != nil {
		return err
	}

	sm.mat.SetUCharAt(row, col, value)
	return nil
}

func (sm *Mat) GetUCharAt3(row, col, channel int) (uint8, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if err := sm.validateCoordinatesAndChannel(row, col, channel); err != nil {
		return 0, err
	}

	return sm.mat.GetUCharAt3(row, col, channel), nil
}

func (sm *Mat) SetUCharAt3(row, col, channel int, value uint8) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.validateCoordinatesAndChannel(row, col, channel); err != nil {
		return err
	}

	sm.mat.SetUCharAt3(row, col, channel, value)
	return nil
}

func (sm *Mat) GetDoubleAt(row, col int) (float64, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if err := sm.validateCoordinates(row, col); err != nil {
		return 0, err
	}

	return sm.mat.GetDoubleAt(row, col), nil
}

func (sm *Mat) SetDoubleAt(row, col int, value float64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.validateCoordinates(row, col); err != nil {
		return err
	}

	sm.mat.SetDoubleAt(row, col, value)
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
	sm.refCount.Add(1)
}

func (sm *Mat) Release() {
	if sm.refCount.Add(-1) == 0 {
		sm.Close()
	}
}

func (sm *Mat) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.mat.Empty() {
		sm.mat.Close()
	}
	sm.mat = gocv.Mat{}
	sm.isValid.Store(false)
	sm.refCount.Store(0)
	sm.memTracker = nil
	sm.tag = ""
	sm.id = 0
}

func (sm *Mat) Close() {
	if !sm.isValid.CompareAndSwap(true, false) {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.memTracker != nil {
		sm.memTracker.ReleaseMat(sm, sm.tag)
		sm.memTracker = nil
	}

	if !sm.mat.Empty() {
		sm.mat.Close()
	}

	runtime.SetFinalizer(sm, nil)
	sm.mat = gocv.Mat{}
	sm.tag = ""
	sm.refCount.Store(0)
	sm.id = 0

	matPool.Put(sm)
}

func (sm *Mat) finalize() {
	if sm.isValid.Load() {
		sm.Close()
	}
}

func (sm *Mat) validateCoordinates(row, col int) error {
	if !sm.IsValid() {
		return fmt.Errorf("Mat is invalid")
	}

	if row < 0 || row >= sm.mat.Rows() || col < 0 || col >= sm.mat.Cols() {
		return fmt.Errorf("coordinates out of bounds: (%d,%d) for size %dx%d",
			col, row, sm.mat.Cols(), sm.mat.Rows())
	}

	return nil
}

func (sm *Mat) validateCoordinatesAndChannel(row, col, channel int) error {
	if err := sm.validateCoordinates(row, col); err != nil {
		return err
	}

	if channel < 0 || channel >= sm.mat.Channels() {
		return fmt.Errorf("channel out of bounds: %d for %d channels", channel, sm.mat.Channels())
	}

	return nil
}

func validateDimensions(rows, cols int) error {
	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", cols, rows)
	}

	if rows > 65536 || cols > 65536 {
		return fmt.Errorf("dimensions %dx%d exceed maximum size", cols, rows)
	}

	return nil
}

func validateSourceMat(srcMat gocv.Mat) error {
	if srcMat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	if srcMat.Rows() <= 0 || srcMat.Cols() <= 0 {
		return fmt.Errorf("source Mat has invalid dimensions: %dx%d", srcMat.Cols(), srcMat.Rows())
	}

	return nil
}

func ValidateMatForOperation(mat *Mat, operation string) error {
	if mat == nil {
		return fmt.Errorf("Mat is nil for operation: %s", operation)
	}

	if !mat.IsValid() {
		return fmt.Errorf("Mat is invalid for operation: %s", operation)
	}

	if mat.Empty() {
		return fmt.Errorf("Mat is empty for operation: %s", operation)
	}

	if mat.Rows() <= 0 || mat.Cols() <= 0 {
		return fmt.Errorf("Mat has invalid dimensions %dx%d for operation: %s",
			mat.Cols(), mat.Rows(), operation)
	}

	return nil
}
