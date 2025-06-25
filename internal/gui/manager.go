package gui

import (
	"image"

	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

type Manager struct {
	window     fyne.Window
	debugMgr   *debug.Manager
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

func NewManager(window fyne.Window, debugMgr *debug.Manager) (*Manager, error) {
	imageDisplay := components.NewImageDisplay()
	controlsPanel := components.NewControlsPanel()
	parametersPanel := components.NewParametersPanel()
	statusBar := components.NewStatusBar()

	manager := &Manager{
		window:          window,
		debugMgr:        debugMgr,
		isShutdown:      false,
		imageDisplay:    imageDisplay,
		controlsPanel:   controlsPanel,
		parametersPanel: parametersPanel,
		statusBar:       statusBar,
	}

	// Set up initial algorithm selection
	manager.setupInitialState()

	return manager, nil
}

func (m *Manager) setupInitialState() {
	// Initialize with 2D Otsu algorithm pre-selected
	m.parametersPanel.UpdateParameters("2D Otsu", map[string]interface{}{
		"quality":                    "Fast",
		"window_size":                7,
		"histogram_bins":             64,
		"neighbourhood_metric":       "mean",
		"pixel_weight_factor":        0.5,
		"smoothing_sigma":            1.0,
		"use_log_histogram":          false,
		"normalize_histogram":        true,
		"apply_contrast_enhancement": false,
	})
}

func (m *Manager) GetMainContainer() *fyne.Container {
	// Left panel: Controls
	leftPanel := container.NewVBox(
		m.controlsPanel.GetContainer(),
	)
	leftPanel.Resize(fyne.NewSize(280, 0))

	// Right panel: Parameters with quality section at top
	rightPanel := container.NewVBox(
		m.parametersPanel.GetContainer(),
	)
	rightPanel.Resize(fyne.NewSize(320, 0))

	// Center: Image display
	centerPanel := m.imageDisplay.GetContainer()

	// Assemble three-column layout
	centerRightSplit := container.NewHSplit(centerPanel, rightPanel)
	centerRightSplit.SetOffset(0.75) // 75% for images, 25% for parameters

	mainLayout := container.NewHSplit(leftPanel, centerRightSplit)
	mainLayout.SetOffset(0.2) // 20% for controls, 80% for center+parameters

	return container.NewBorder(
		nil,                        // Top
		m.statusBar.GetContainer(), // Bottom
		nil, nil,                   // Left/right
		mainLayout, // Center
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
		handler(algorithm)
		// Update parameters panel when algorithm changes
		m.requestParameterUpdate(algorithm)
	})
}

func (m *Manager) SetParameterChangeHandler(handler func(string, interface{})) {
	m.parameterChangeHandler = handler
	m.parametersPanel.SetParameterChangeHandler(handler)
}

func (m *Manager) SetGeneratePreviewHandler(handler func()) {
	m.generatePreviewHandler = handler
	m.controlsPanel.SetGeneratePreviewHandler(handler)
}

func (m *Manager) SetOriginalImage(img image.Image) {
	fyne.Do(func() {
		m.imageDisplay.SetOriginalImage(img)
	})
}

func (m *Manager) SetPreviewImage(img image.Image) {
	fyne.Do(func() {
		m.imageDisplay.SetPreviewImage(img)
	})
}

func (m *Manager) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	fyne.Do(func() {
		m.parametersPanel.UpdateParameters(algorithm, params)
	})
}

func (m *Manager) UpdateStatus(status string) {
	fyne.Do(func() {
		m.statusBar.SetStatus(status)
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
	})
}

func (m *Manager) ShowError(title string, err error) {
	m.debugMgr.LogError("GUIManager", err)
	fyne.Do(func() {
		dialog.ShowError(err, m.window)
	})
}

func (m *Manager) requestParameterUpdate(algorithm string) {
	// This will be called by handlers to update parameters when algorithm changes
	if m.algorithmChangeHandler != nil {
		// The actual parameter update will be handled by the application handler
	}
}

func (m *Manager) Shutdown() {
	if m.isShutdown {
		return
	}

	m.isShutdown = true
}
