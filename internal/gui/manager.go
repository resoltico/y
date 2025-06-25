package gui

import (
	"image"

	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

const (
	LeftPanelWidth  = 280
	RightPanelWidth = 320
)

type Manager struct {
	window     fyne.Window
	debugCoord debug.Coordinator
	logger     debug.Logger
	isShutdown bool

	// Components
	imageDisplay    *components.ImageDisplay
	controlsPanel   *components.ControlsPanel
	parametersPanel *components.ParametersPanel
	statusBar       *components.StatusBar

	// Event handlers
	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	parameterChangeHandler func(string, interface{})
	generatePreviewHandler func()
}

func NewManager(window fyne.Window, debugCoord debug.Coordinator) (*Manager, error) {
	logger := debugCoord.Logger()

	imageDisplay := components.NewImageDisplay()
	controlsPanel := components.NewControlsPanel()
	parametersPanel := components.NewParametersPanel()
	statusBar := components.NewStatusBar()

	manager := &Manager{
		window:          window,
		debugCoord:      debugCoord,
		logger:          logger,
		isShutdown:      false,
		imageDisplay:    imageDisplay,
		controlsPanel:   controlsPanel,
		parametersPanel: parametersPanel,
		statusBar:       statusBar,
	}

	manager.setupInitialState()

	logger.Info("GUIManager", "initialized with constrained image display", map[string]interface{}{
		"left_width":   LeftPanelWidth,
		"right_width":  RightPanelWidth,
		"image_width":  components.ImageConstraintWidth,
		"image_height": components.ImageConstraintHeight,
	})

	return manager, nil
}

func (m *Manager) setupInitialState() {
	// Initialize with 2D Otsu algorithm pre-selected
	defaultParams := map[string]interface{}{
		"quality":                    "Fast",
		"window_size":                7,
		"histogram_bins":             64,
		"neighbourhood_metric":       "mean",
		"pixel_weight_factor":        0.5,
		"smoothing_sigma":            1.0,
		"use_log_histogram":          false,
		"normalize_histogram":        true,
		"apply_contrast_enhancement": false,
	}

	m.parametersPanel.UpdateParameters("2D Otsu", defaultParams)

	m.logger.Debug("GUIManager", "initial state configured", map[string]interface{}{
		"algorithm": "2D Otsu",
		"params":    len(defaultParams),
	})
}

func (m *Manager) GetMainContainer() *fyne.Container {
	// Create left panel with width constraint
	leftPanel := container.NewVBox(m.controlsPanel.GetContainer())

	// Create right panel with width constraint
	rightPanel := container.NewVBox(m.parametersPanel.GetContainer())

	// Get center panel (constrained image display)
	centerPanel := m.imageDisplay.GetContainer()

	// Use BorderLayout for layout management
	mainContent := container.NewBorder(
		nil, nil,
		leftPanel,
		rightPanel,
		centerPanel,
	)

	// Add status bar at bottom
	return container.NewBorder(
		nil,
		m.statusBar.GetContainer(),
		nil, nil,
		mainContent,
	)
}

func (m *Manager) GetWindow() fyne.Window {
	return m.window
}

func (m *Manager) SetImageLoadHandler(handler func()) {
	m.imageLoadHandler = handler
	m.controlsPanel.SetImageLoadHandler(handler)
}

func (m *Manager) SetImageSaveHandler(handler func()) {
	m.imageSaveHandler = handler
	m.controlsPanel.SetImageSaveHandler(handler)
}

func (m *Manager) SetAlgorithmChangeHandler(handler func(string)) {
	m.algorithmChangeHandler = handler
	m.controlsPanel.SetAlgorithmChangeHandler(func(algorithm string) {
		m.logger.Debug("GUIManager", "algorithm change requested", map[string]interface{}{
			"algorithm": algorithm,
		})

		handler(algorithm)
		m.requestParameterUpdate(algorithm)
	})
}

func (m *Manager) SetParameterChangeHandler(handler func(string, interface{})) {
	m.parameterChangeHandler = handler
	m.parametersPanel.SetParameterChangeHandler(func(name string, value interface{}) {
		m.logger.Debug("GUIManager", "parameter change", map[string]interface{}{
			"parameter": name,
			"value":     value,
		})

		handler(name, value)
	})
}

func (m *Manager) SetGeneratePreviewHandler(handler func()) {
	m.generatePreviewHandler = handler
	m.controlsPanel.SetGeneratePreviewHandler(func() {
		m.logger.Info("GUIManager", "preview generation started", nil)
		handler()
	})
}

func (m *Manager) SetOriginalImage(img image.Image) {
	fyne.Do(func() {
		m.imageDisplay.SetOriginalImage(img)
		m.logger.Debug("GUIManager", "original image set", map[string]interface{}{
			"bounds": img.Bounds(),
		})
	})
}

func (m *Manager) SetPreviewImage(img image.Image) {
	fyne.Do(func() {
		m.imageDisplay.SetPreviewImage(img)
		m.logger.Debug("GUIManager", "preview image set", map[string]interface{}{
			"bounds": img.Bounds(),
		})
	})
}

func (m *Manager) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	fyne.Do(func() {
		m.parametersPanel.UpdateParameters(algorithm, params)
		m.logger.Debug("GUIManager", "parameter panel updated", map[string]interface{}{
			"algorithm":   algorithm,
			"param_count": len(params),
		})
	})
}

func (m *Manager) UpdateStatus(status string) {
	fyne.Do(func() {
		m.statusBar.SetStatus(status)
		m.logger.Debug("GUIManager", "status updated", map[string]interface{}{
			"status": status,
		})
	})
}

func (m *Manager) UpdateProgress(progress float64) {
	fyne.Do(func() {
		m.statusBar.SetProgress(progress)
	})
}

func (m *Manager) UpdateMetrics(psnr, ssim float64) {
	fyne.Do(func() {
		m.statusBar.SetMetrics(psnr, ssim)
		m.logger.Debug("GUIManager", "metrics updated", map[string]interface{}{
			"psnr": psnr,
			"ssim": ssim,
		})
	})
}

func (m *Manager) ShowError(title string, err error) {
	m.logger.Error("GUIManager", err, map[string]interface{}{
		"title": title,
	})

	fyne.Do(func() {
		dialog.ShowError(err, m.window)
	})
}

func (m *Manager) requestParameterUpdate(algorithm string) {
	// Parameter update handled by application handler
}

func (m *Manager) Shutdown() {
	if m.isShutdown {
		return
	}

	m.isShutdown = true
	m.logger.Info("GUIManager", "shutdown initiated", nil)
}
