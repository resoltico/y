package app

import (
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"
)

type Lifecycle struct {
	memoryManager *memory.Manager
	debugCoord    debug.Coordinator
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	logger        debug.Logger
	isShutdown    bool
}

func NewLifecycle(mm *memory.Manager, dc debug.Coordinator, gm *gui.Manager) *Lifecycle {
	return &Lifecycle{
		memoryManager: mm,
		debugCoord:    dc,
		guiManager:    gm,
		logger:        dc.Logger(),
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
	l.logger.Info("Lifecycle", "shutdown sequence initiated", nil)

	// Shutdown components in reverse dependency order
	if l.coordinator != nil {
		if coordWithCleanup, ok := l.coordinator.(*pipeline.Coordinator); ok {
			coordWithCleanup.Cleanup()
			l.logger.Debug("Lifecycle", "coordinator cleanup completed", nil)
		}
	}

	if l.guiManager != nil {
		l.guiManager.Shutdown()
		l.logger.Debug("Lifecycle", "GUI manager shutdown completed", nil)
	}

	if l.memoryManager != nil {
		l.memoryManager.Cleanup()
		l.logger.Debug("Lifecycle", "memory manager cleanup completed", nil)
	}

	// Debug coordinator shutdown last to capture all cleanup events
	if l.debugCoord != nil {
		l.logger.Info("Lifecycle", "shutdown sequence completed", nil)
		l.debugCoord.Shutdown()
	}
}
