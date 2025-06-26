package widgets

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	container          *fyne.Container
	loadButton         *widget.Button
	saveButton         *widget.Button
	algorithmSelect    *widget.Select
	qualitySelect      *widget.Select
	processButton      *widget.Button
	statusLabel        *widget.Label
	progressLabel      *widget.Label
	metricsLabel       *widget.Label

	// Event handlers
	loadHandler            func()
	saveHandler            func()
	processHandler         func()
	algorithmChangeHandler func(string)
	qualityChangeHandler   func(string)
}

func NewToolbar() *Toolbar {
	toolbar := &Toolbar{}
	toolbar.createComponents()
	toolbar.buildLayout()
	return toolbar
}

func (t *Toolbar) createComponents() {
	// Action buttons
	t.loadButton = widget.NewButton("Load", t.onLoadClicked)
	t.loadButton.Importance = widget.HighImportance
	
	t.saveButton = widget.NewButton("Save", t.onSaveClicked)
	t.saveButton.Importance = widget.HighImportance
	
	t.processButton = widget.NewButton("Process", t.onProcessClicked)
	t.processButton.Importance = widget.HighImportance

	// Algorithm selection
	t.algorithmSelect = widget.NewSelect(
		[]string{"2D Otsu", "Iterative Triclass"},
		t.onAlgorithmChanged,
	)
	t.algorithmSelect.SetSelected("2D Otsu")

	// Quality selection
	t.qualitySelect = widget.NewSelect(
		[]string{"Fast", "Best"},
		t.onQualityChanged,
	)
	t.qualitySelect.SetSelected("Fast")

	// Status and metrics
	t.statusLabel = widget.NewLabel("Ready")
	t.progressLabel = widget.NewLabel("")
	t.metricsLabel = widget.NewLabel("PSNR: -- | SSIM: --")
}

func (t *Toolbar) buildLayout() {
	// Create background
	background := canvas.NewRectangle(color.RGBA{R: 250, G: 249, B: 245, A: 255})
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeWidth = 1.0
	border.StrokeColor = color.RGBA{R: 231, G: 231, B: 231, A: 255}

	// Left section: Load/Save
	leftSection := container.NewHBox(t.loadButton, t.saveButton)

	// Center section: Algorithm controls
	algorithmGroup := container.NewVBox(
		widget.NewLabel("Algorithm"),
		t.algorithmSelect,
	)
	
	qualityGroup := container.NewVBox(
		widget.NewLabel("Quality"),
		t.qualitySelect,
	)
	
	processGroup := container.NewVBox(
		widget.NewLabel("Action"),
		t.processButton,
	)

	centerSection := container.NewHBox(
		algorithmGroup,
		widget.NewSeparator(),
		qualityGroup,
		widget.NewSeparator(),
		processGroup,
	)

	// Status section
	statusSection := container.NewHBox(t.statusLabel, t.progressLabel)

	// Right section: Metrics
	rightSection := container.NewHBox(t.metricsLabel)

	// Main layout
	content := container.NewBorder(
		nil, nil,
		leftSection,
		rightSection,
		container.NewHBox(centerSection, widget.NewSeparator(), statusSection),
	)

	// Layer with background
	t.container = container.NewStack(
		border,
		container.NewPadded(
			container.NewStack(background, container.NewPadded(content)),
		),
	)
}

// Event handlers
func (t *Toolbar) onLoadClicked() {
	if t.loadHandler != nil {
		t.loadHandler()
	}
}

func (t *Toolbar) onSaveClicked() {
	if t.saveHandler != nil {
		t.saveHandler()
	}
}

func (t *Toolbar) onProcessClicked() {
	if t.processHandler != nil {
		t.processHandler()
	}
}

func (t *Toolbar) onAlgorithmChanged(algorithm string) {
	if t.algorithmChangeHandler != nil {
		t.algorithmChangeHandler(algorithm)
	}
}

func (t *Toolbar) onQualityChanged(quality string) {
	if t.qualityChangeHandler != nil {
		t.qualityChangeHandler(quality)
	}
}

// Public interface
func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}

func (t *Toolbar) SetLoadHandler(handler func()) {
	t.loadHandler = handler
}

func (t *Toolbar) SetSaveHandler(handler func()) {
	t.saveHandler = handler
}

func (t *Toolbar) SetProcessHandler(handler func()) {
	t.processHandler = handler
}

func (t *Toolbar) SetAlgorithmChangeHandler(handler func(string)) {
	t.algorithmChangeHandler = handler
}

func (t *Toolbar) SetQualityChangeHandler(handler func(string)) {
	t.qualityChangeHandler = handler
}

func (t *Toolbar) SetStatus(status string) {
	t.statusLabel.SetText(status)
}

func (t *Toolbar) SetProgress(progress string) {
	t.progressLabel.SetText(progress)
}

func (t *Toolbar) SetMetrics(psnr, ssim float64) {
	if psnr > 0 && ssim > 0 {
		t.metricsLabel.SetText(fmt.Sprintf("PSNR: %.2f dB | SSIM: %.4f", psnr, ssim))
	} else {
		t.metricsLabel.SetText("PSNR: -- | SSIM: --")
	}
}
