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
	container       *fyne.Container
	loadButton      *widget.Button
	saveButton      *widget.Button
	algorithmSelect *widget.Select
	processButton   *widget.Button
	cancelButton    *widget.Button
	statusLabel     *widget.Label
	metricsLabel    *widget.Label

	loadHandler            func()
	saveHandler            func()
	processHandler         func()
	cancelHandler          func()
	algorithmChangeHandler func(string)
}

func NewToolbar() *Toolbar {
	toolbar := &Toolbar{}
	toolbar.createComponents()
	toolbar.buildLayout()
	return toolbar
}

func (t *Toolbar) createComponents() {
	// Action buttons with modern styling
	t.loadButton = widget.NewButton("Load Image", t.onLoadClicked)
	t.loadButton.Importance = widget.HighImportance

	t.saveButton = widget.NewButton("Save Result", t.onSaveClicked)
	t.saveButton.Importance = widget.HighImportance
	t.saveButton.Disable() // Disabled until processing completes

	t.processButton = widget.NewButton("Process", t.onProcessClicked)
	t.processButton.Importance = widget.HighImportance
	t.processButton.Disable() // Disabled until image loaded

	t.cancelButton = widget.NewButton("Cancel", t.onCancelClicked)
	t.cancelButton.Importance = widget.MediumImportance
	t.cancelButton.Disable() // Disabled until processing starts

	// Algorithm selection
	t.algorithmSelect = widget.NewSelect(
		[]string{"2D Otsu", "Iterative Triclass"},
		t.onAlgorithmChanged,
	)
	t.algorithmSelect.SetSelected("2D Otsu")

	// Status and metrics display
	t.statusLabel = widget.NewLabel("Ready")
	t.metricsLabel = widget.NewLabel("IoU: -- | Dice: -- | Error: --")
}

func (t *Toolbar) buildLayout() {
	// Create modern background with subtle styling
	background := canvas.NewRectangle(color.RGBA{R: 248, G: 249, B: 250, A: 255})

	// Action buttons section
	actionSection := container.NewHBox(
		t.loadButton,
		widget.NewSeparator(),
		t.saveButton,
	)

	// Algorithm selection section
	algorithmGroup := container.NewVBox(
		widget.NewLabel("Algorithm"),
		t.algorithmSelect,
	)

	// Processing control section
	processGroup := container.NewVBox(
		widget.NewLabel("Processing"),
		container.NewHBox(t.processButton, t.cancelButton),
	)

	// Status section
	statusGroup := container.NewVBox(
		widget.NewLabel("Status"),
		t.statusLabel,
	)

	// Metrics section
	metricsGroup := container.NewVBox(
		widget.NewLabel("Quality Metrics"),
		t.metricsLabel,
	)

	// Main content layout
	content := container.NewHBox(
		actionSection,
		widget.NewSeparator(),
		algorithmGroup,
		widget.NewSeparator(),
		processGroup,
		widget.NewSeparator(),
		statusGroup,
		widget.NewSeparator(),
		metricsGroup,
	)

	// Apply padding and background
	t.container = container.NewStack(
		background,
		container.NewPadded(content),
	)
}

func (t *Toolbar) onLoadClicked() {
	if t.loadHandler != nil {
		// Use fyne.Do for thread safety in Fyne v2.6+
		fyne.Do(func() {
			t.loadHandler()
		})
	}
}

func (t *Toolbar) onSaveClicked() {
	if t.saveHandler != nil {
		fyne.Do(func() {
			t.saveHandler()
		})
	}
}

func (t *Toolbar) onProcessClicked() {
	if t.processHandler != nil {
		// Enable cancel button, disable process button during processing
		t.processButton.Disable()
		t.cancelButton.Enable()

		fyne.Do(func() {
			t.processHandler()
		})
	}
}

func (t *Toolbar) onCancelClicked() {
	if t.cancelHandler != nil {
		fyne.Do(func() {
			t.cancelHandler()
		})

		// Reset button states
		t.cancelButton.Disable()
		t.processButton.Enable()
	}
}

func (t *Toolbar) onAlgorithmChanged(algorithm string) {
	if t.algorithmChangeHandler != nil {
		fyne.Do(func() {
			t.algorithmChangeHandler(algorithm)
		})
	}
}

func (t *Toolbar) GetContainer() *fyne.Container {
	return t.container
}

// Event handler setters
func (t *Toolbar) SetLoadHandler(handler func()) {
	t.loadHandler = handler
}

func (t *Toolbar) SetSaveHandler(handler func()) {
	t.saveHandler = handler
}

func (t *Toolbar) SetProcessHandler(handler func()) {
	t.processHandler = handler
}

func (t *Toolbar) SetCancelHandler(handler func()) {
	t.cancelHandler = handler
}

func (t *Toolbar) SetAlgorithmChangeHandler(handler func(string)) {
	t.algorithmChangeHandler = handler
}

// UI state management methods
func (t *Toolbar) SetStatus(status string) {
	fyne.Do(func() {
		t.statusLabel.SetText(status)

		// Update button states based on status
		switch status {
		case "Ready":
			t.processButton.Enable()
			t.cancelButton.Disable()
			t.saveButton.Enable()
		case "Loading image...", "Saving image...":
			t.processButton.Disable()
			t.cancelButton.Disable()
		case "Starting processing...", "Processing":
			t.processButton.Disable()
			t.cancelButton.Enable()
			t.saveButton.Disable()
		case "Processing completed":
			t.processButton.Enable()
			t.cancelButton.Disable()
			t.saveButton.Enable()
		case "Processing cancelled", "Processing failed":
			t.processButton.Enable()
			t.cancelButton.Disable()
		}
	})
}

func (t *Toolbar) SetMetrics(iou, dice, misclassError float64) {
	t.SetSegmentationMetrics(iou, dice, misclassError, -1, -1)
}

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

// Convenience methods for enabling/disabling functionality
func (t *Toolbar) EnableImageLoaded() {
	fyne.Do(func() {
		t.processButton.Enable()
	})
}

func (t *Toolbar) EnableProcessingCompleted() {
	fyne.Do(func() {
		t.processButton.Enable()
		t.cancelButton.Disable()
		t.saveButton.Enable()
	})
}

func (t *Toolbar) DisableAllControls() {
	fyne.Do(func() {
		t.loadButton.Disable()
		t.saveButton.Disable()
		t.processButton.Disable()
		t.cancelButton.Disable()
		t.algorithmSelect.Disable()
	})
}

func (t *Toolbar) EnableAllControls() {
	fyne.Do(func() {
		t.loadButton.Enable()
		t.saveButton.Enable()
		t.processButton.Enable()
		t.algorithmSelect.Enable()
	})
}
