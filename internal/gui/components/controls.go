package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ControlsPanel struct {
	container      *fyne.Container
	algorithmRadio *widget.RadioGroup
	generateButton *widget.Button

	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	generatePreviewHandler func()
}

func NewControlsPanel() *ControlsPanel {
	panel := &ControlsPanel{}
	panel.setupControls()
	return panel
}

func (cp *ControlsPanel) setupControls() {
	loadButton := widget.NewButton("Load Image", cp.onImageLoad)
	saveButton := widget.NewButton("Save Image", cp.onImageSave)

	fileOpsCard := widget.NewCard("File Operations", "", container.NewVBox(
		loadButton,
		saveButton,
	))

	cp.algorithmRadio = widget.NewRadioGroup([]string{"2D Otsu", "Iterative Triclass"}, cp.onAlgorithmSelected)
	cp.algorithmRadio.SetSelected("2D Otsu")

	algorithmCard := widget.NewCard("Algorithm", "", cp.algorithmRadio)

	cp.generateButton = widget.NewButton("Generate Preview", cp.onGeneratePreview)
	cp.generateButton.Importance = widget.HighImportance

	processingCard := widget.NewCard("Processing", "", cp.generateButton)

	cp.container = container.NewVBox(
		fileOpsCard,
		algorithmCard,
		processingCard,
	)
}

func (cp *ControlsPanel) GetContainer() *fyne.Container {
	return cp.container
}

func (cp *ControlsPanel) SetImageLoadHandler(handler func()) {
	cp.imageLoadHandler = handler
}

func (cp *ControlsPanel) SetImageSaveHandler(handler func()) {
	cp.imageSaveHandler = handler
}

func (cp *ControlsPanel) SetAlgorithmChangeHandler(handler func(string)) {
	cp.algorithmChangeHandler = handler
}

func (cp *ControlsPanel) SetGeneratePreviewHandler(handler func()) {
	cp.generatePreviewHandler = handler
}

func (cp *ControlsPanel) onImageLoad() {
	if cp.imageLoadHandler != nil {
		cp.imageLoadHandler()
	}
}

func (cp *ControlsPanel) onImageSave() {
	if cp.imageSaveHandler != nil {
		cp.imageSaveHandler()
	}
}

func (cp *ControlsPanel) onAlgorithmSelected(algorithm string) {
	if cp.algorithmChangeHandler != nil {
		cp.algorithmChangeHandler(algorithm)
	}
}

func (cp *ControlsPanel) onGeneratePreview() {
	if cp.generatePreviewHandler != nil {
		cp.generatePreviewHandler()
	}
}
