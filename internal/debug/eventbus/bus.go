package eventbus

import (
	"context"
	"sync"
	"time"
)

type Event struct {
	Type      string
	Timestamp time.Time
	Data      map[string]interface{}
	Context   context.Context
}

type EventHandler interface {
	Handle(event Event)
	GetID() string
}

type Bus struct {
	subscribers map[string][]EventHandler
	mu          sync.RWMutex
	buffer      chan Event
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewBus(bufferSize int) *Bus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &Bus{
		subscribers: make(map[string][]EventHandler),
		buffer:      make(chan Event, bufferSize),
		ctx:         ctx,
		cancel:      cancel,
	}

	bus.startWorker()
	return bus
}

func (b *Bus) Publish(event Event) {
	event.Timestamp = time.Now()

	select {
	case b.buffer <- event:
	case <-b.ctx.Done():
	default:
		// Drop event if buffer full to prevent blocking
	}
}

func (b *Bus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *Bus) Unsubscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.subscribers[eventType]
	for i, h := range handlers {
		if h.GetID() == handler.GetID() {
			b.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (b *Bus) Shutdown() {
	b.cancel()
	close(b.buffer)
	b.wg.Wait()
}

func (b *Bus) startWorker() {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		for {
			select {
			case event, ok := <-b.buffer:
				if !ok {
					return
				}
				b.dispatchEvent(event)
			case <-b.ctx.Done():
				return
			}
		}
	}()
}

func (b *Bus) dispatchEvent(event Event) {
	b.mu.RLock()
	handlers := make([]EventHandler, len(b.subscribers[event.Type]))
	copy(handlers, b.subscribers[event.Type])
	b.mu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic but continue processing
				}
			}()
			h.Handle(event)
		}(handler)
	}
}
