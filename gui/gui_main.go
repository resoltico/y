package gui

import (
	"fyne.io/fyne/v2"

	"otsu-obliterator/pipeline"
)

type MainInterface struct {
	window        fyne.Window
	layoutManager *LayoutManager

	// Callbacks
	onImageLoad       func()
	onImageSave       func()
	onAlgorithmChange func(string)
	onParameterChange func(string, interface{})
	onGeneratePreview func()
}

func NewMainInterface(window fyne.Window,
	onImageLoad, onImageSave func(),
	onAlgorithmChange func(string),
	onParameterChange func(string, interface{}),
	onGeneratePreview func()) *MainInterface {

	gui := &MainInterface{
		window:            window,
		onImageLoad:       onImageLoad,
		onImageSave:       onImageSave,
		onAlgorithmChange: onAlgorithmChange,
		onParameterChange: onParameterChange,
		onGeneratePreview: onGeneratePreview,
	}

	// Create layout manager with callbacks
	gui.layoutManager = NewLayoutManager(
		onImageLoad,
		onImageSave,
		onAlgorithmChange,
		onParameterChange,
		onGeneratePreview,
	)

	return gui
}

func (gui *MainInterface) Initialize() {
	gui.layoutManager.Initialize()
}

func (gui *MainInterface) GetMainContainer() *fyne.Container {
	return gui.layoutManager.GetMainContainer()
}

func (gui *MainInterface) SetOriginalImage(imageData *pipeline.ImageData) {
	gui.layoutManager.SetOriginalImage(imageData)
}

func (gui *MainInterface) SetPreviewImage(imageData *pipeline.ImageData) {
	gui.layoutManager.SetPreviewImage(imageData)
}

func (gui *MainInterface) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	gui.layoutManager.UpdateParameterPanel(algorithm, params)
}

func (gui *MainInterface) UpdateStatus(status string) {
	gui.layoutManager.UpdateStatus(status)
}

func (gui *MainInterface) UpdateProgress(progress float64) {
	gui.layoutManager.UpdateProgress(progress)
}

func (gui *MainInterface) UpdateMetrics(psnr, ssim float64) {
	gui.layoutManager.UpdateMetrics(psnr, ssim)
}
