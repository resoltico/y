package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// LayoutManager coordinates the main application layout
type LayoutManager struct {
	mainContainer *fyne.Container
	imageDisplay  *ImageDisplay
	controlsPanel *ControlsPanel
	statusBar     *StatusBar
}

func NewLayoutManager(
	onImageLoad, onImageSave func(),
	onAlgorithmChange func(string),
	onParameterChange func(string, interface{}),
	onGeneratePreview func()) *LayoutManager {

	// Create image display (top row: Original | Preview)
	imageDisplay := NewImageDisplay()

	// Create controls panel (bottom row: full width)
	controlsPanel := NewControlsPanel(onAlgorithmChange, onParameterChange, onGeneratePreview)

	// Create status bar
	statusBar := NewStatusBar()

	// Main layout using border layout
	// Top: image display, Bottom: status, Center: controls
	mainContainer := container.NewBorder(
		imageDisplay.GetContainer(),  // top
		statusBar.GetContainer(),     // bottom
		nil,                          // left
		nil,                          // right
		controlsPanel.GetContainer(), // center
	)

	return &LayoutManager{
		mainContainer: mainContainer,
		imageDisplay:  imageDisplay,
		controlsPanel: controlsPanel,
		statusBar:     statusBar,
	}
}

func (lm *LayoutManager) GetMainContainer() *fyne.Container {
	return lm.mainContainer
}

func (lm *LayoutManager) Initialize() {
	lm.controlsPanel.Initialize()
}

// Image display methods
func (lm *LayoutManager) SetOriginalImage(imageData interface{}) {
	lm.imageDisplay.SetOriginalImage(imageData)
}

func (lm *LayoutManager) SetPreviewImage(imageData interface{}) {
	lm.imageDisplay.SetPreviewImage(imageData)
}

// Controls methods
func (lm *LayoutManager) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	lm.controlsPanel.UpdateParameters(algorithm, params)
}

// Status methods
func (lm *LayoutManager) UpdateStatus(status string) {
	lm.statusBar.SetStatus(status)
}

func (lm *LayoutManager) UpdateProgress(progress float64) {
	lm.controlsPanel.UpdateProgress(progress)
}

func (lm *LayoutManager) UpdateMetrics(psnr, ssim float64) {
	lm.statusBar.SetMetrics(psnr, ssim)
}
