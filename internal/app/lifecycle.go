package app

import (
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"
)

type Lifecycle struct {
	memoryManager *memory.Manager
	debugManager  *debug.Manager
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	isShutdown    bool
}

func NewLifecycle(mm *memory.Manager, dm *debug.Manager, gm *gui.Manager) *Lifecycle {
	return &Lifecycle{
		memoryManager: mm,
		debugManager:  dm,
		guiManager:    gm,
		isShutdown:    false,
	}
}

func (l *Lifecycle) SetCoordinator(coord pipeline.ProcessingCoordinator) {
	l.coordinator = coord
}

func (l *Lifecycle) Shutdown() {
	if l.isShutdown {
		return
	}
	
	l.isShutdown = true
	
	if l.coordinator != nil {
		if coordWithCleanup, ok := l.coordinator.(*pipeline.Coordinator); ok {
			coordWithCleanup.Cleanup()
		}
	}
	
	if l.guiManager != nil {
		l.guiManager.Shutdown()
	}
	
	if l.memoryManager != nil {
		l.memoryManager.Cleanup()
	}
	
	if l.debugManager != nil {
		l.debugManager.Cleanup()
	}
}
