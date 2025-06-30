package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Toolbar represents the main application toolbar
type Toolbar struct {
	container               *fyne.Container
	loadButton              *widget.Button
	saveButton              *widget.Button
	processButton           *widget.Button
	cancelButton            *widget.Button
	algorithmSelect         *widget.Select
	metricsLabel            *widget.Label
	
	// Event handlers
	loadHandler             func()
	saveHandler             func()
	processHandler          func()
	cancelHandler           func()
	algorithmChangeHandler  func(string)
	
	// State
	currentAlgorithm        string
	processingActive        bool
}

// NewToolbar creates a new toolbar component
func NewToolbar() *Toolbar {
	toolbar := &Toolbar{}
	toolbar.createComponents()
	toolbar.buildLayout()
	toolbar.setupEventHandlers()
	return toolbar
}

// createComponents initializes all toolbar components
func (t *Toolbar) createComponents() {
	// Action buttons
	t.loadButton = widget.NewButton("Load Image", nil)
	t.loadButton.Importance = widget.HighImportance
	
	t.saveButton = widget.NewButton("Save Result", nil)
	t.saveButton.Importance = widget.HighImportance
	t.saveButton.Disable()
	
	t.processButton = widget.NewButton("Process", nil)
	t.processButton.Importance = widget.HighImportance
	t.processButton.Disable()
	
	t.cancelButton = widget.NewButton("Cancel", nil)
	t.cancelButton.Importance = widget.MediumImportance
	t.cancelButton.Disable()
	
	// Algorithm selection
	t.algorithmSelect = widget.NewSelect(
		[]string{"2D Otsu", "Iterative Triclass"},
		nil,
	)
	t.algorithmSelect.SetSelected("2D Otsu")
	t.currentAlgorithm = "2D Otsu"
	
	// Metrics display
	t.metricsLabel = widget.NewLabel("IoU: -- | Dice: -- | Error: --")
}

// buildLayout constructs the toolbar layout
func (t *Toolbar) buildLayout() {
	// Action section
	actionSection := container.NewHBox(
		t.loadButton,
		widget.NewSeparator(),
		t.saveButton,
	)
	
	// Algorithm section
	algorithmSection := container.NewVBox(
		widget.NewLabel("Algorithm"),
		t.algorithmSelect,
	)
	
	// Processing section
	processSection := container.NewVBox(
		widget.NewLabel("Processing"),
		container.NewHBox(t.processButton, t.cancelButton),
	)
	
	// Metrics section
	metricsSection := container.NewVBox(
		widget.NewLabel("Quality Metrics"),
		t.metricsLabel,
	)
	
	// Main toolbar layout
	t.container = container.NewHBox(
		actionSection,
		widget.NewSeparator(),
		algorithmSection,
		widget.NewSeparator(),
		processSection,
		widget.NewSeparator(),
		metricsSection,
	)
}

// setupEventHandlers connects button events
func (t *Toolbar) setupEventHandlers() {
	t.loadButton.OnTapped = func() {
		if t.loadHandler != nil {
			t.loadHandler()
		}
	}
	
	t.saveButton.OnTapped = func() {
		if t.saveHandler != nil {
			t.saveHandler()
		}
	}
	
	t.processButton.OnTapped = func() {
		if t.processHandler != nil {
			t.processHandler()
		}
	}
	
	t.cancelButton.OnTapped = func() {
		if t.cancelHandler != nil {
			t.cancelHandler()
		}
	}
	
	t.algorithmSelect.OnChanged = func(algorithm string) {
		t.currentAlgorithm = algorithm
		if t.algorithmChangeHandler != nil {
			t.algorithmChangeHandler(algorithm)
		}
	}
}

// Event handler setters

// SetLoadHandler sets the load image handler
func (t *Toolbar) SetLoadHandler(handler func()) {
	t.loadHandler = handler
}

// SetSaveHandler sets the save image handler
func (t *Toolbar) SetSaveHandler(handler func()) {
	t.saveHandler = handler
}

// SetProcessHandler sets the process image handler
func (t *Toolbar) SetProcessHandler(handler func()) {
	t.processHandler = handler
}

// SetCancelHandler sets the cancel processing handler
func (t *Toolbar) SetCancelHandler(handler func()) {
	t.cancelHandler = handler
}

// SetAlgorithmChangeHandler sets the algorithm change handler
func (t *Toolbar) SetAlgorithmChangeHandler(handler func(string)) {
	t.algorithmChangeHandler = handler
}

// State management methods

// SetProcessingActive updates the processing state
func (t *Toolbar) SetProcessingActive(active bool) {
	fyne.Do(func() {
		t.processingActive = active
		
		if active {
			t.processButton.Disable()
			t.cancelButton.Enable()
			t.saveButton.Disable()
		} else {
			t.processButton.Enable()
			t.cancelButton.Disable()
			t.saveButton.Enable()
		}
	})
}

// EnableImageOperations enables/disables image-dependent operations
func (t *Toolbar) EnableImageOperations(enabled bool) {
	fyne.Do(func() {
		if enabled && !t.processingActive {
			t.processButton.Enable()
		} else {
			t.processButton.Disable()
		}
	})
}

// SetCurrentAlgorithm updates the current algorithm
func (t *Toolbar) SetCurrentAlgorithm(algorithm string) {
	fyne.Do(func() {
		t.currentAlgorithm = algorithm
		t.algorithmSelect.SetSelected(algorithm)
	})
}

// GetCurrentAlgorithm returns the current algorithm
func (t *Toolbar) GetCurrentAlgorithm() string {
	return t.currentAlgorithm
}

// SetSegmentationMetrics updates the metrics display
func (t *Toolbar) SetSegmentationMetrics(iou, dice, misclassError, uniformity, boundaryAccuracy float64) {
	fyne.Do(func() {
		if iou >= 0 && dice >= 0 {
			if misclassError >= 0 {
				text := fmt.Sprintf("IoU: %.3f | Dice: %.3f | Error: %.3f", iou, dice, misclassError)
				t.metricsLabel.SetText(text)
			} else {
				text := fmt.Sprintf("IoU: %.3f | Dice: %.3f | Error: --", iou, dice)
				t.metricsLabel.SetText(text)
			}
		} else {
			t.metricsLabel.SetText("IoU: -- | Dice: -- | Error: --")
		}
	})
}

// Reset resets the toolbar to initial state
func (t *Toolbar) Reset() {
	fyne.Do(func() {
		t.processButton.Disable()
		t.cancelButton.Disable()
		t.saveButton.Disable()
		t.metricsLabel.SetText("IoU: -- | Dice: -- | Error: --")
		t.algorithmSelect.SetSelected("2D Otsu")
		t.currentAlgorithm = "2D Otsu"
		t.processingActive = false
	})
}

// GetContainer returns the toolbar container
func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}

// Refresh refreshes the toolbar display
func (t *Toolbar) Refresh() {
	fyne.Do(func() {
		t.container.Refresh()
	})
}