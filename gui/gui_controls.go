package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ControlsPanel handles algorithm selection, parameters, and generation
type ControlsPanel struct {
	container      *fyne.Container
	algorithmRadio *widget.RadioGroup
	parameterPanel *ParameterPanel
	generateButton *widget.Button
	progressBar    *widget.ProgressBar

	onAlgorithmChange func(string)
	onParameterChange func(string, interface{})
	onGenerate        func()
}

func NewControlsPanel(
	onAlgorithmChange func(string),
	onParameterChange func(string, interface{}),
	onGenerate func()) *ControlsPanel {

	panel := &ControlsPanel{
		onAlgorithmChange: onAlgorithmChange,
		onParameterChange: onParameterChange,
		onGenerate:        onGenerate,
	}

	panel.setupControls()
	return panel
}

func (cp *ControlsPanel) setupControls() {
	// Algorithm selection
	algorithmLabel := widget.NewLabel("Algorithm")
	cp.algorithmRadio = widget.NewRadioGroup([]string{"2D Otsu", "Iterative Triclass"}, nil)

	algorithmContainer := container.NewVBox(
		algorithmLabel,
		cp.algorithmRadio,
	)

	// Parameter panel
	cp.parameterPanel = NewParameterPanel(cp.onAlgorithmChange, cp.onParameterChange, cp.onGenerate)

	// Generate button and progress
	cp.generateButton = widget.NewButton("Generate Preview", cp.onGenerate)
	cp.generateButton.Importance = widget.HighImportance

	cp.progressBar = widget.NewProgressBar()
	cp.progressBar.Hide()

	buttonContainer := container.NewVBox(
		cp.generateButton,
		cp.progressBar,
	)

	// Create horizontal layout: Algorithm | Parameters | Generate
	controlsLayout := container.NewHBox(
		algorithmContainer,
		widget.NewSeparator(),
		cp.parameterPanel.GetContainer(),
		widget.NewSeparator(),
		buttonContainer,
	)

	// Wrap in container with padding
	cp.container = container.NewPadded(controlsLayout)
}

func (cp *ControlsPanel) GetContainer() *fyne.Container {
	return cp.container
}

func (cp *ControlsPanel) Initialize() {
	// Set callback and default selection after setup
	cp.algorithmRadio.OnChanged = cp.onAlgorithmSelected
	cp.algorithmRadio.SetSelected("2D Otsu")
	cp.parameterPanel.Initialize()
}

func (cp *ControlsPanel) onAlgorithmSelected(algorithm string) {
	cp.onAlgorithmChange(algorithm)
}

func (cp *ControlsPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	fyne.Do(func() {
		cp.parameterPanel.UpdateParameters(algorithm, params)
	})
}

func (cp *ControlsPanel) UpdateProgress(progress float64) {
	fyne.Do(func() {
		if progress > 0 && progress < 1 {
			cp.progressBar.Show()
			cp.progressBar.SetValue(progress)
		} else {
			cp.progressBar.Hide()
		}
	})
}
