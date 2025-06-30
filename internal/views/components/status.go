package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// StatusBar displays application status and information
type StatusBar struct {
	container    *fyne.Container
	statusLabel  *widget.Label
	imageInfo    *widget.Label
	memoryInfo   *widget.Label
}

// NewStatusBar creates a new status bar component
func NewStatusBar() *StatusBar {
	sb := &StatusBar{}
	sb.createComponents()
	sb.buildLayout()
	return sb
}

// createComponents initializes status bar components
func (sb *StatusBar) createComponents() {
	sb.statusLabel = widget.NewLabel("Ready")
	sb.imageInfo = widget.NewLabel("No image loaded")
	sb.memoryInfo = widget.NewLabel("Memory: --")
}

// buildLayout constructs the status bar layout
func (sb *StatusBar) buildLayout() {
	sb.container = container.NewHBox(
		sb.statusLabel,
		widget.NewSeparator(),
		sb.imageInfo,
		widget.NewSeparator(),
		sb.memoryInfo,
	)
}

// SetStatus updates the main status message
func (sb *StatusBar) SetStatus(status string) {
	fyne.Do(func() {
		sb.statusLabel.SetText(status)
	})
}

// GetStatus returns the current status message
func (sb *StatusBar) GetStatus() string {
	return sb.statusLabel.Text
}

// SetImageInfo updates the image information display
func (sb *StatusBar) SetImageInfo(width, height, channels int, format string) {
	fyne.Do(func() {
		info := fmt.Sprintf("Image: %dx%d, %d channels, %s", width, height, channels, format)
		sb.imageInfo.SetText(info)
	})
}

// SetMemoryInfo updates the memory usage display
func (sb *StatusBar) SetMemoryInfo(used, total int64) {
	fyne.Do(func() {
		usedMB := used / (1024 * 1024)
		totalMB := total / (1024 * 1024)
		info := fmt.Sprintf("Memory: %d/%d MB", usedMB, totalMB)
		sb.memoryInfo.SetText(info)
	})
}

// Reset resets the status bar to initial state
func (sb *StatusBar) Reset() {
	fyne.Do(func() {
		sb.statusLabel.SetText("Ready")
		sb.imageInfo.SetText("No image loaded")
		sb.memoryInfo.SetText("Memory: --")
	})
}

// GetContainer returns the status bar container
func (sb *StatusBar) GetContainer() *fyne.Container {
	return sb.container
}

// Refresh refreshes the status bar display
func (sb *StatusBar) Refresh() {
	fyne.Do(func() {
		sb.container.Refresh()
	})
}

// ProgressBar displays processing progress with stage information
type ProgressBar struct {
	container   *fyne.Container
	progressBar *widget.ProgressBar
	stageLabel  *widget.Label
	visible     bool
}

// NewProgressBar creates a new progress bar component
func NewProgressBar() *ProgressBar {
	pb := &ProgressBar{}
	pb.createComponents()
	pb.buildLayout()
	return pb
}

// createComponents initializes progress bar components
func (pb *ProgressBar) createComponents() {
	pb.progressBar = widget.NewProgressBar()
	pb.progressBar.SetValue(0.0)
	pb.stageLabel = widget.NewLabel("Ready")
	pb.visible = false
}

// buildLayout constructs the progress bar layout
func (pb *ProgressBar) buildLayout() {
	pb.container = container.NewVBox(
		pb.stageLabel,
		pb.progressBar,
	)
	pb.container.Hide() // Initially hidden
}

// SetProgress updates the progress value (0.0 to 1.0)
func (pb *ProgressBar) SetProgress(progress float64) {
	fyne.Do(func() {
		if progress < 0.0 {
			progress = 0.0
		} else if progress > 1.0 {
			progress = 1.0
		}
		pb.progressBar.SetValue(progress)
	})
}

// GetProgress returns the current progress value
func (pb *ProgressBar) GetProgress() float64 {
	return pb.progressBar.Value
}

// SetStage updates the current processing stage
func (pb *ProgressBar) SetStage(stage string) {
	fyne.Do(func() {
		pb.stageLabel.SetText(stage)
	})
}

// GetStage returns the current stage
func (pb *ProgressBar) GetStage() string {
	return pb.stageLabel.Text
}

// SetVisible shows or hides the progress bar
func (pb *ProgressBar) SetVisible(visible bool) {
	fyne.Do(func() {
		pb.visible = visible
		if visible {
			pb.container.Show()
		} else {
			pb.container.Hide()
		}
	})
}

// IsVisible returns true if the progress bar is visible
func (pb *ProgressBar) IsVisible() bool {
	return pb.visible
}

// Reset resets the progress bar to initial state
func (pb *ProgressBar) Reset() {
	fyne.Do(func() {
		pb.progressBar.SetValue(0.0)
		pb.stageLabel.SetText("Ready")
		pb.SetVisible(false)
	})
}

// GetContainer returns the progress bar container
func (pb *ProgressBar) GetContainer() *fyne.Container {
	return pb.container
}

// Refresh refreshes the progress bar display
func (pb *ProgressBar) Refresh() {
	fyne.Do(func() {
		pb.container.Refresh()
	})
}