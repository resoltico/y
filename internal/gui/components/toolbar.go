package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Toolbar struct {
	container              *fyne.Container
	loadButton             *widget.Button
	saveButton             *widget.Button
	algorithmRadio         *widget.RadioGroup
	generateButton         *widget.Button
	statusLabel            *widget.Label
	progressLabel          *widget.Label
	metricsLabel           *widget.Label
	
	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	generatePreviewHandler func()
}

func NewToolbar() *Toolbar {
	toolbar := &Toolbar{}
	toolbar.setupToolbar()
	return toolbar
}

func (t *Toolbar) setupToolbar() {
	t.loadButton = widget.NewButton("Load", t.onImageLoad)
	t.saveButton = widget.NewButton("Save", t.onImageSave)
	
	algorithmLabel := widget.NewLabel("Algorithm:")
	t.algorithmRadio = widget.NewRadioGroup([]string{"2D Otsu", "Iterative Triclass"}, t.onAlgorithmSelected)
	t.algorithmRadio.SetSelected("2D Otsu")
	t.algorithmRadio.Horizontal = true
	
	t.generateButton = widget.NewButton("Generate", t.onGeneratePreview)
	t.generateButton.Importance = widget.HighImportance
	
	t.statusLabel = widget.NewLabel("Ready")
	t.progressLabel = widget.NewLabel("")
	t.metricsLabel = widget.NewLabel("PSNR: -- | SSIM: --")

	t.container = container.NewHBox(
		t.loadButton,
		widget.NewSeparator(),
		t.saveButton,
		widget.NewSeparator(),
		algorithmLabel,
		t.algorithmRadio,
		widget.NewSeparator(),
		t.generateButton,
		widget.NewSeparator(),
		t.statusLabel,
		t.progressLabel,
		widget.NewSeparator(),
		t.metricsLabel,
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
		t.metricsLabel.SetText(fmt.Sprintf("PSNR: %.2f dB | SSIM: %.4f", psnr, ssim))
	} else {
		t.metricsLabel.SetText("PSNR: -- | SSIM: --")
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

func (t *Toolbar) onGeneratePreview() {
	if t.generatePreviewHandler != nil {
		t.generatePreviewHandler()
	}
}