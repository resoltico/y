package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ControlsPanel struct {
	container           *fyne.Container
	algorithmRadio      *widget.RadioGroup
	parametersContainer *fyne.Container
	generateButton      *widget.Button
	progressBar         *widget.ProgressBar
	parameterPanel      *ParameterPanel

	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	parameterChangeHandler func(string, interface{})
	generatePreviewHandler func()
}

func NewControlsPanel() *ControlsPanel {
	panel := &ControlsPanel{}
	panel.setupControls()
	return panel
}

func (cp *ControlsPanel) setupControls() {
	algorithmLabel := widget.NewLabel("Algorithm")
	cp.algorithmRadio = widget.NewRadioGroup([]string{"2D Otsu", "Iterative Triclass"}, cp.onAlgorithmSelected)
	cp.algorithmRadio.SetSelected("2D Otsu")

	algorithmContainer := container.NewVBox(
		algorithmLabel,
		cp.algorithmRadio,
	)

	cp.parameterPanel = NewParameterPanel()
	cp.parametersContainer = cp.parameterPanel.GetContainer()

	cp.generateButton = widget.NewButton("Generate Preview", cp.onGeneratePreview)
	cp.generateButton.Importance = widget.HighImportance

	cp.progressBar = widget.NewProgressBar()
	cp.progressBar.Hide()

	buttonContainer := container.NewVBox(
		cp.generateButton,
		cp.progressBar,
	)

	topControls := container.NewHBox(
		algorithmContainer,
		widget.NewSeparator(),
		cp.parametersContainer,
		widget.NewSeparator(),
		buttonContainer,
	)

	menuBar := cp.createMenuBar()

	cp.container = container.NewVBox(
		menuBar,
		widget.NewSeparator(),
		topControls,
	)
}

func (cp *ControlsPanel) createMenuBar() *fyne.Container {
	loadButton := widget.NewButton("Load Image", cp.onImageLoad)
	saveButton := widget.NewButton("Save Image", cp.onImageSave)

	return container.NewHBox(
		loadButton,
		widget.NewSeparator(),
		saveButton,
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

func (cp *ControlsPanel) SetParameterChangeHandler(handler func(string, interface{})) {
	cp.parameterChangeHandler = handler
	cp.parameterPanel.SetParameterChangeHandler(handler)
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

func (cp *ControlsPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	cp.parameterPanel.UpdateParameters(algorithm, params)
}

func (cp *ControlsPanel) SetProgress(progress float64) {
	if progress > 0 && progress < 1 {
		cp.progressBar.Show()
		cp.progressBar.SetValue(progress)
	} else {
		cp.progressBar.Hide()
	}
}