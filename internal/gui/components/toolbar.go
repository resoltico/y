package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	container      *fyne.Container
	LoadButton     *widget.Button
	SaveButton     *widget.Button
	AlgorithmGroup *fyne.Container
	algorithmRadio *widget.RadioGroup
	QualityGroup   *fyne.Container
	qualityRadio   *widget.RadioGroup
	GenerateButton *widget.Button
	StatusGroup    *fyne.Container
	statusLabel    *widget.Label
	progressLabel  *widget.Label
	MetricsLabel   *widget.Label

	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	qualityChangeHandler   func(string)
	generatePreviewHandler func()
}

func NewToolbar() *Toolbar {
	toolbar := &Toolbar{}
	toolbar.setupToolbar()
	return toolbar
}

func (t *Toolbar) setupToolbar() {
	// Create toolbar background with controllable border
	background := canvas.NewRectangle(color.RGBA{R: 250, G: 249, B: 245, A: 255})
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 1.0
	border.StrokeColor = color.RGBA{R: 231, G: 231, B: 231, A: 255}

	// Left section: Load/Save buttons with high importance styling
	t.LoadButton = widget.NewButton("Load", t.onImageLoad)
	t.LoadButton.Importance = widget.HighImportance
	t.SaveButton = widget.NewButton("Save", t.onImageSave)
	t.SaveButton.Importance = widget.HighImportance
	leftSection := container.NewHBox(t.LoadButton, t.SaveButton)

	// Algorithm section with vertical layout
	algorithmLabel := widget.NewLabel("Algorithm:")
	t.algorithmRadio = widget.NewRadioGroup([]string{"2D Otsu", "Iterative Triclass"}, t.onAlgorithmSelected)
	t.algorithmRadio.SetSelected("2D Otsu")
	t.algorithmRadio.Horizontal = false

	// Place label alongside first radio button
	t.AlgorithmGroup = container.NewHBox(
		algorithmLabel,
		t.algorithmRadio,
	)

	// Quality section with vertical layout
	qualityLabel := widget.NewLabel("Quality:")
	t.qualityRadio = widget.NewRadioGroup([]string{"Fast", "Best"}, t.onQualitySelected)
	t.qualityRadio.SetSelected("Fast")
	t.qualityRadio.Horizontal = false

	// Place label alongside first radio button
	t.QualityGroup = container.NewHBox(
		qualityLabel,
		t.qualityRadio,
	)

	// Generate button
	t.GenerateButton = widget.NewButton("Generate", t.onGeneratePreview)
	t.GenerateButton.Importance = widget.HighImportance

	// Status section
	t.statusLabel = widget.NewLabel("Ready")
	t.progressLabel = widget.NewLabel("")
	t.StatusGroup = container.NewHBox(t.statusLabel, t.progressLabel)

	// Right section: Metrics
	t.MetricsLabel = widget.NewLabel("PSNR: -- | SSIM: --")
	rightSection := container.NewHBox(t.MetricsLabel)

	// Create center section with all controls
	centerSection := container.NewHBox(
		t.AlgorithmGroup,
		widget.NewSeparator(),
		t.QualityGroup,
		widget.NewSeparator(),
		t.GenerateButton,
		widget.NewSeparator(),
		t.StatusGroup,
	)

	// Create main toolbar container using Border layout
	toolbarContent := container.NewBorder(
		nil, nil,
		leftSection,
		rightSection,
		centerSection,
	)

	// Create content with 1px padding for border effect
	contentWithPadding := container.NewPadded(toolbarContent)

	// Layer border, background and content
	t.container = container.NewStack(
		border,
		container.NewPadded(
			container.NewStack(background, contentWithPadding),
		),
	)
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}

func (t *Toolbar) SetImageLoadHandler(handler func()) {
	t.imageLoadHandler = handler
}

func (t *Toolbar) SetImageSaveHandler(handler func()) {
	t.imageSaveHandler = handler
}

func (t *Toolbar) SetAlgorithmChangeHandler(handler func(string)) {
	t.algorithmChangeHandler = handler
}

func (t *Toolbar) SetQualityChangeHandler(handler func(string)) {
	t.qualityChangeHandler = handler
}

func (t *Toolbar) SetGeneratePreviewHandler(handler func()) {
	t.generatePreviewHandler = handler
}

func (t *Toolbar) SetStatus(status string) {
	t.statusLabel.SetText(status)
}

func (t *Toolbar) SetProgress(progress string) {
	t.progressLabel.SetText(progress)
}

func (t *Toolbar) SetMetrics(psnr, ssim float64) {
	if psnr > 0 && ssim > 0 {
		t.MetricsLabel.SetText(fmt.Sprintf("PSNR: %.2f dB | SSIM: %.4f", psnr, ssim))
	} else {
		t.MetricsLabel.SetText("PSNR: -- | SSIM: --")
	}
}

func (t *Toolbar) onImageLoad() {
	if t.imageLoadHandler != nil {
		t.imageLoadHandler()
	}
}

func (t *Toolbar) onImageSave() {
	if t.imageSaveHandler != nil {
		t.imageSaveHandler()
	}
}

func (t *Toolbar) onAlgorithmSelected(algorithm string) {
	if t.algorithmChangeHandler != nil {
		t.algorithmChangeHandler(algorithm)
	}
}

func (t *Toolbar) onQualitySelected(quality string) {
	if t.qualityChangeHandler != nil {
		t.qualityChangeHandler(quality)
	}
}

func (t *Toolbar) onGeneratePreview() {
	if t.generatePreviewHandler != nil {
		t.generatePreviewHandler()
	}
}
