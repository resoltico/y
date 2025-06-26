package gui

import (
	"fmt"
	"image"

	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

type Manager struct {
	window     fyne.Window
	debugCoord debug.Coordinator
	logger     debug.Logger
	isShutdown bool

	imageDisplay      *components.ImageDisplay
	toolbar           *components.ResponsiveToolbar
	parametersSection *components.ParametersSection

	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	parameterChangeHandler func(string, interface{})
	generatePreviewHandler func()
}

func NewManager(window fyne.Window, debugCoord debug.Coordinator) (*Manager, error) {
	logger := debugCoord.Logger()

	imageDisplay := components.NewImageDisplay()
	toolbar := components.NewResponsiveToolbar()
	parametersSection := components.NewParametersSection()

	manager := &Manager{
		window:            window,
		debugCoord:        debugCoord,
		logger:            logger,
		isShutdown:        false,
		imageDisplay:      imageDisplay,
		toolbar:           toolbar,
		parametersSection: parametersSection,
	}

	manager.setupInitialParameters()

	logger.Info("GUIManager", "initialized with synchronized layout", map[string]interface{}{
		"image_width":  components.ImageAreaWidth,
		"image_height": components.ImageAreaHeight,
	})

	return manager, nil
}

func (m *Manager) setupInitialParameters() {
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

	m.parametersSection.UpdateParameters("2D Otsu", defaultParams)
}

func (m *Manager) GetMainContainer() *fyne.Container {
	// Create responsive toolbar with proper positioning
	leftSection := container.NewHBox(
		m.toolbar.LoadButton,
		m.toolbar.SaveButton,
	)

	centerSection := container.NewHBox(
		m.toolbar.AlgorithmGroup,
		container.NewCenter(m.toolbar.GenerateButton),
		m.toolbar.StatusGroup,
	)

	rightSection := container.NewHBox(
		m.toolbar.MetricsLabel,
	)

	responsiveToolbar := container.NewBorder(
		nil, nil,
		leftSection,
		rightSection,
		centerSection,
	)

	return container.NewVBox(
		m.imageDisplay.GetContainer(),
		responsiveToolbar,
		m.parametersSection.GetContainer(),
	)
}

func (m *Manager) GetWindow() fyne.Window {
	return m.window
}

func (m *Manager) SetImageLoadHandler(handler func()) {
	m.imageLoadHandler = handler
	m.toolbar.SetImageLoadHandler(handler)
}

func (m *Manager) SetImageSaveHandler(handler func()) {
	m.imageSaveHandler = handler
	m.toolbar.SetImageSaveHandler(handler)
}

func (m *Manager) SetAlgorithmChangeHandler(handler func(string)) {
	m.algorithmChangeHandler = handler
	m.toolbar.SetAlgorithmChangeHandler(func(algorithm string) {
		m.logger.Debug("GUIManager", "algorithm change requested", map[string]interface{}{
			"algorithm": algorithm,
		})

		handler(algorithm)
		m.requestParameterUpdate(algorithm)
	})
}

func (m *Manager) SetParameterChangeHandler(handler func(string, interface{})) {
	m.parameterChangeHandler = handler
	m.parametersSection.SetParameterChangeHandler(func(name string, value interface{}) {
		m.logger.Debug("GUIManager", "parameter change", map[string]interface{}{
			"parameter": name,
			"value":     value,
		})

		handler(name, value)
	})
}

func (m *Manager) SetGeneratePreviewHandler(handler func()) {
	m.generatePreviewHandler = handler
	m.toolbar.SetGeneratePreviewHandler(func() {
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
		m.parametersSection.UpdateParameters(algorithm, params)
		m.logger.Debug("GUIManager", "parameter panel updated", map[string]interface{}{
			"algorithm":   algorithm,
			"param_count": len(params),
		})
	})
}

func (m *Manager) UpdateStatus(status string) {
	fyne.Do(func() {
		m.toolbar.SetStatus(status)
		m.logger.Debug("GUIManager", "status updated", map[string]interface{}{
			"status": status,
		})
	})
}

func (m *Manager) UpdateProgress(progress float64) {
	fyne.Do(func() {
		if progress > 0 && progress < 1 {
			m.toolbar.SetProgress(fmt.Sprintf("[%.0f%%]", progress*100))
		} else {
			m.toolbar.SetProgress("")
		}
	})
}

func (m *Manager) UpdateMetrics(psnr, ssim float64) {
	fyne.Do(func() {
		m.toolbar.SetMetrics(psnr, ssim)
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
