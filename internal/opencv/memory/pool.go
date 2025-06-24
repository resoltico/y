package memory

import (
	"sync"

	"otsu-obliterator/internal/opencv/safe"
)

type Pool struct {
	mats    []*safe.Mat
	maxSize int
	mu      sync.Mutex
}

func NewPool(maxSize int) *Pool {
	return &Pool{
		mats:    make([]*safe.Mat, 0, maxSize),
		maxSize: maxSize,
	}
}

func (p *Pool) Get() *safe.Mat {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.mats) == 0 {
		return nil
	}

	mat := p.mats[len(p.mats)-1]
	p.mats = p.mats[:len(p.mats)-1]

	if mat.IsValid() && !mat.Empty() {
		return mat
	}

	mat.Close()
	return nil
}

func (p *Pool) Put(mat *safe.Mat) bool {
	if mat == nil || !mat.IsValid() || mat.Empty() {
		return false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.mats) >= p.maxSize {
		return false
	}

	p.mats = append(p.mats, mat)
	return true
}

func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.mats)
}

func (p *Pool) Cleanup() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := len(p.mats)
	for _, mat := range p.mats {
		mat.Close()
	}
	p.mats = p.mats[:0]
	return count
}