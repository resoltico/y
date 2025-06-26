package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"otsu-obliterator/internal/logger"
)

type Shutdownable interface {
	Shutdown()
}

type Manager struct {
	components []Shutdownable
	logger     logger.Logger
	mu         sync.Mutex
	done       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewManager(log logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	
	m := &Manager{
		components: make([]Shutdownable, 0),
		logger:     log,
		done:       make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	return m
}

func (m *Manager) Register(component Shutdownable) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.components = append(m.components, component)
}

func (m *Manager) Listen() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	
	go func() {
		sig := <-sigChan
		m.logger.Info("ShutdownManager", "shutdown signal received", map[string]interface{}{
			"signal": sig.String(),
		})
		m.Shutdown()
	}()
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	select {
	case <-m.done:
		return // Already shutting down
	default:
		close(m.done)
	}
	
	m.logger.Info("ShutdownManager", "shutdown sequence initiated", map[string]interface{}{
		"components": len(m.components),
	})
	
	m.cancel()
	
	// Shutdown components in reverse order
	for i := len(m.components) - 1; i >= 0; i-- {
		component := m.components[i]
		
		done := make(chan struct{})
		go func() {
			defer close(done)
			component.Shutdown()
		}()
		
		select {
		case <-done:
			// Component shut down successfully
		case <-time.After(10 * time.Second):
			m.logger.Warning("ShutdownManager", "component shutdown timeout", map[string]interface{}{
				"component_index": i,
			})
		}
	}
	
	m.logger.Info("ShutdownManager", "shutdown sequence completed", nil)
}

func (m *Manager) Context() context.Context {
	return m.ctx
}

func (m *Manager) Done() <-chan struct{} {
	return m.done
}