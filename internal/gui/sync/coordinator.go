package sync

import (
	"fyne.io/fyne/v2"
)

type UpdateType int

const (
	UpdateTypeImageDisplay UpdateType = iota
	UpdateTypeParameterPanel
	UpdateTypeStatus
	UpdateTypeProgress
	UpdateTypeMetrics
)

type Update struct {
	Type UpdateType
	Data interface{}
}

type Coordinator struct {
	updateChan chan *Update
	done       chan struct{}
	processor  *UpdateProcessor
}

func NewCoordinator() *Coordinator {
	processor := NewUpdateProcessor()
	
	return &Coordinator{
		updateChan: make(chan *Update, 100),
		done:       make(chan struct{}),
		processor:  processor,
	}
}

func (c *Coordinator) ScheduleUpdate(update *Update) {
	select {
	case c.updateChan <- update:
	default:
		// Channel full, skip this update to prevent blocking
	}
}

func (c *Coordinator) Run() {
	for {
		select {
		case update := <-c.updateChan:
			fyne.Do(func() {
				c.processor.ProcessUpdate(update)
			})
		case <-c.done:
			return
		}
	}
}

func (c *Coordinator) Stop() {
	close(c.done)
}

func (c *Coordinator) SetImageDisplay(display ImageDisplayHandler) {
	c.processor.SetImageDisplay(display)
}

func (c *Coordinator) SetParameterPanel(panel ParameterPanelHandler) {
	c.processor.SetParameterPanel(panel)
}

func (c *Coordinator) SetStatusBar(statusBar StatusBarHandler) {
	c.processor.SetStatusBar(statusBar)
}

func (c *Coordinator) SetProgressBar(progressBar ProgressBarHandler) {
	c.processor.SetProgressBar(progressBar)
}