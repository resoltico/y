package gui

import (
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
)

// Manager coordinates the MVC pattern for the GUI
type Manager struct {
	window     fyne.Window
	controller *Controller
	view       *View
	logger     logger.Logger
	isShutdown bool
}

func NewManager(window fyne.Window, log logger.Logger) (*Manager, error) {
	manager := &Manager{
		window:     window,
		logger:     log,
		isShutdown: false,
	}

	manager.initializeComponents()

	log.Info("GUIManager", "initialized with MVC pattern", map[string]interface{}{
		"window_title": window.Title(),
	})

	return manager, nil
}

func (m *Manager) initializeComponents() {
	// Create view first
	m.view = NewView(m.window)

	// Create controller - will be set with processing coordinator later
	m.controller = NewController(nil, m.logger)

	// Connect view and controller
	m.view.SetController(m.controller)
	m.controller.SetView(m.view)
}

// SetProcessingCoordinator connects the processing pipeline
func (m *Manager) SetProcessingCoordinator(coordinator pipeline.ProcessingCoordinator) {
	// Update controller with coordinator
	m.controller = NewController(coordinator, m.logger)

	// Reconnect view and controller
	m.view.SetController(m.controller)
	m.controller.SetView(m.view)

	m.logger.Info("GUIManager", "processing coordinator connected", nil)
}

// Public interface for the application layer
func (m *Manager) GetMainContainer() *fyne.Container {
	return m.view.GetMainContainer()
}

func (m *Manager) GetWindow() fyne.Window {
	return m.window
}

func (m *Manager) Show() {
	m.view.Show()
	m.logger.Info("GUIManager", "GUI displayed", nil)
}

// Direct interface methods - these delegate to the controller using fyne.Do
func (m *Manager) SetOriginalImage(img interface{}) {
	if imageData, ok := img.(*pipeline.ImageData); ok {
		fyne.Do(func() {
			m.controller.view.SetOriginalImage(imageData.Image)
		})
	}
}

func (m *Manager) SetPreviewImage(img interface{}) {
	if imageData, ok := img.(*pipeline.ImageData); ok {
		fyne.Do(func() {
			m.controller.view.SetPreviewImage(imageData.Image)
		})
	}
}

func (m *Manager) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	fyne.Do(func() {
		m.view.UpdateParameterPanel(algorithm, params)
	})
}

func (m *Manager) UpdateStatus(status string) {
	fyne.Do(func() {
		m.view.SetStatus(status)
	})
}

func (m *Manager) UpdateProgress(progress float64) {
	fyne.Do(func() {
		m.view.SetProgress(progress)
	})
}

func (m *Manager) UpdateMetrics(psnr, ssim float64) {
	fyne.Do(func() {
		m.view.SetMetrics(psnr, ssim)
	})
}

func (m *Manager) ShowError(title string, err error) {
	fyne.Do(func() {
		m.view.ShowError(title, err)
	})
}

func (m *Manager) Shutdown() {
	if m.isShutdown {
		return
	}

	m.isShutdown = true
	m.logger.Info("GUIManager", "shutdown initiated", nil)

	if m.controller != nil {
		m.controller.Shutdown()
	}

	if m.view != nil {
		m.view.Shutdown()
	}

	m.logger.Info("GUIManager", "shutdown completed", nil)
}
