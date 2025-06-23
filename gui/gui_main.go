package gui

import (
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"otsu-obliterator/pipeline"
)

type MainInterface struct {
	window fyne.Window

	// Main containers
	mainContainer  *container.Split
	leftContainer  *container.Split
	rightContainer *fyne.Container

	// Image displays
	originalImage *canvas.Image
	previewImage  *canvas.Image

	// Parameter panel
	parameterPanel *ParameterPanel

	// Status and metrics
	statusBar   *StatusBar
	progressBar *widget.ProgressBar

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

	gui.setupInterface()
	return gui
}

func (gui *MainInterface) setupInterface() {
	// Create image displays
	gui.originalImage = canvas.NewImageFromResource(nil)
	gui.originalImage.FillMode = canvas.ImageFillContain
	gui.originalImage.SetMinSize(fyne.NewSize(320, 240))

	gui.previewImage = canvas.NewImageFromResource(nil)
	gui.previewImage.FillMode = canvas.ImageFillContain
	gui.previewImage.SetMinSize(fyne.NewSize(320, 240))

	// Create image containers with labels
	originalContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		gui.originalImage,
	)

	previewContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		gui.previewImage,
	)

	// Create left split (Original | Preview) - 50/50 fixed
	gui.leftContainer = container.NewHSplit(originalContainer, previewContainer)
	gui.leftContainer.SetOffset(0.5) // Fixed 50/50 split

	// Create parameter panel with real callbacks
	gui.parameterPanel = NewParameterPanel(
		gui.onAlgorithmChange,
		gui.onParameterChange,
		gui.onGeneratePreview,
	)

	// Create status bar
	gui.statusBar = NewStatusBar()

	// Create progress bar
	gui.progressBar = widget.NewProgressBar()
	gui.progressBar.Hide()

	// Create right container (parameter panel + status)
	gui.rightContainer = container.NewVBox(
		gui.parameterPanel.GetContainer(),
		widget.NewSeparator(),
		gui.progressBar,
		gui.statusBar.GetContainer(),
	)

	// Create main split (Images | Controls) - 50/50 fixed
	gui.mainContainer = container.NewHSplit(gui.leftContainer, gui.rightContainer)
	gui.mainContainer.SetOffset(0.5) // Fixed 50/50 split
}

func (gui *MainInterface) Initialize() {
	// Initialize parameter panel callbacks and trigger initial state
	gui.parameterPanel.Initialize()
}

func (gui *MainInterface) GetMainContainer() *container.Split {
	return gui.mainContainer
}

func (gui *MainInterface) SetOriginalImage(imageData *pipeline.ImageData) {
	if imageData == nil || imageData.Image == nil {
		return
	}

	// Resize for display while maintaining aspect ratio
	displayImg := gui.resizeForDisplay(imageData.Image)

	fyne.Do(func() {
		gui.originalImage.Image = displayImg
		gui.originalImage.Refresh()
	})
}

func (gui *MainInterface) SetPreviewImage(imageData *pipeline.ImageData) {
	if imageData == nil || imageData.Image == nil {
		return
	}

	// Resize for display while maintaining aspect ratio
	displayImg := gui.resizeForDisplay(imageData.Image)

	fyne.Do(func() {
		gui.previewImage.Image = displayImg
		gui.previewImage.Refresh()
	})
}

func (gui *MainInterface) resizeForDisplay(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate preview size: min(original_size×0.5, 640×480)
	maxWidth := 640
	maxHeight := 480

	// Scale to 50% first
	targetWidth := width / 2
	targetHeight := height / 2

	// Then clamp to maximum
	if targetWidth > maxWidth {
		ratio := float64(maxWidth) / float64(targetWidth)
		targetWidth = maxWidth
		targetHeight = int(float64(targetHeight) * ratio)
	}

	if targetHeight > maxHeight {
		ratio := float64(maxHeight) / float64(targetHeight)
		targetHeight = maxHeight
		targetWidth = int(float64(targetWidth) * ratio)
	}

	// For now, return original image (actual resizing would be done in pipeline)
	// This is where we'd call a resize function from the pipeline
	return img
}

func (gui *MainInterface) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	if gui.parameterPanel != nil {
		gui.parameterPanel.UpdateParameters(algorithm, params)
	}
}

func (gui *MainInterface) UpdateStatus(status string) {
	fyne.Do(func() {
		gui.statusBar.SetStatus(status)
	})
}

func (gui *MainInterface) UpdateProgress(progress float64) {
	fyne.Do(func() {
		if progress > 0 && progress < 1 {
			gui.progressBar.Show()
			gui.progressBar.SetValue(progress)
		} else {
			gui.progressBar.Hide()
		}
	})
}

func (gui *MainInterface) UpdateMetrics(psnr, ssim float64) {
	psnrText := fmt.Sprintf("PSNR: %.2f dB", psnr)
	ssimText := fmt.Sprintf("SSIM: %.4f", ssim)

	fyne.Do(func() {
		gui.statusBar.SetMetrics(psnrText, ssimText)
	})
}
