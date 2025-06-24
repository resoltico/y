package app

import (
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/opencv/memory"
)

type Lifecycle struct {
	memoryManager *memory.Manager
	debugManager  *debug.Manager
	guiManager    *gui.Manager
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

func (l *Lifecycle) Shutdown() {
	if l.isShutdown {
		return
	}
	
	l.isShutdown = true
	
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