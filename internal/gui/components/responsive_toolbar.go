package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ResponsiveToolbar struct {
	container      *fyne.Container
	LoadButton     *widget.Button
	SaveButton     *widget.Button
	AlgorithmGroup *fyne.Container
	algorithmRadio *widget.RadioGroup
	GenerateButton *widget.Button
	StatusGroup    *fyne.Container
	statusLabel    *widget.Label
	progressLabel  *widget.Label
	MetricsLabel   *widget.Label

	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	generatePreviewHandler func()
}

func NewResponsiveToolbar() *ResponsiveToolbar {
	toolbar := &ResponsiveToolbar{}
	toolbar.setupToolbar()
	return toolbar
}

func (rt *ResponsiveToolbar) setupToolbar() {
	// Create toolbar background
	background := canvas.NewRectangle(color.RGBA{R: 250, G: 249, B: 245, A: 255})

	// Left section: Load/Save buttons
	rt.LoadButton = widget.NewButton("Load", rt.onImageLoad)
	rt.SaveButton = widget.NewButton("Save", rt.onImageSave)
	leftSection := container.NewHBox(rt.LoadButton, rt.SaveButton)

	// Algorithm section with minimal spacing
	algorithmLabel := widget.NewLabel("Algorithm:")
	rt.algorithmRadio = widget.NewRadioGroup([]string{"2D Otsu", "Iterative Triclass"}, rt.onAlgorithmSelected)
	rt.algorithmRadio.SetSelected("2D Otsu")
	rt.algorithmRadio.Horizontal = true

	// Use BorderLayout to control spacing more precisely
	rt.AlgorithmGroup = container.NewBorder(
		nil, nil, algorithmLabel, nil, rt.algorithmRadio,
	)

	// Generate button
	rt.GenerateButton = widget.NewButton("Generate", rt.onGeneratePreview)
	rt.GenerateButton.Importance = widget.HighImportance

	// Status section
	rt.statusLabel = widget.NewLabel("Ready")
	rt.progressLabel = widget.NewLabel("")
	rt.StatusGroup = container.NewHBox(rt.statusLabel, rt.progressLabel)

	// Right section: Metrics
	rt.MetricsLabel = widget.NewLabel("PSNR: -- | SSIM: --")
	rightSection := container.NewHBox(rt.MetricsLabel)

	// Create center section with all controls
	centerSection := container.NewHBox(
		rt.AlgorithmGroup,
		widget.NewSeparator(),
		rt.GenerateButton,
		widget.NewSeparator(),
		rt.StatusGroup,
	)

	// Create main toolbar container using Border layout
	toolbarContent := container.NewBorder(
		nil, nil,
		leftSection,
		rightSection,
		centerSection,
	)

	// Layer background and content
	rt.container = container.NewStack(background, toolbarContent)
}

func (rt *ResponsiveToolbar) GetContainer() *fyne.Container {
	return rt.container
}

func (rt *ResponsiveToolbar) SetImageLoadHandler(handler func()) {
	rt.imageLoadHandler = handler
}

func (rt *ResponsiveToolbar) SetImageSaveHandler(handler func()) {
	rt.imageSaveHandler = handler
}

func (rt *ResponsiveToolbar) SetAlgorithmChangeHandler(handler func(string)) {
	rt.algorithmChangeHandler = handler
}

func (rt *ResponsiveToolbar) SetGeneratePreviewHandler(handler func()) {
	rt.generatePreviewHandler = handler
}

func (rt *ResponsiveToolbar) SetStatus(status string) {
	rt.statusLabel.SetText(status)
}

func (rt *ResponsiveToolbar) SetProgress(progress string) {
	rt.progressLabel.SetText(progress)
}

func (rt *ResponsiveToolbar) SetMetrics(psnr, ssim float64) {
	if psnr > 0 && ssim > 0 {
		rt.MetricsLabel.SetText(fmt.Sprintf("PSNR: %.2f dB | SSIM: %.4f", psnr, ssim))
	} else {
		rt.MetricsLabel.SetText("PSNR: -- | SSIM: --")
	}
}

func (rt *ResponsiveToolbar) onImageLoad() {
	if rt.imageLoadHandler != nil {
		rt.imageLoadHandler()
	}
}

func (rt *ResponsiveToolbar) onImageSave() {
	if rt.imageSaveHandler != nil {
		rt.imageSaveHandler()
	}
}

func (rt *ResponsiveToolbar) onAlgorithmSelected(algorithm string) {
	if rt.algorithmChangeHandler != nil {
		rt.algorithmChangeHandler(algorithm)
	}
}

func (rt *ResponsiveToolbar) onGeneratePreview() {
	if rt.generatePreviewHandler != nil {
		rt.generatePreviewHandler()
	}
}
