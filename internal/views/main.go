package views

import (
	"fmt"
	"image"

	"otsu-obliterator/internal/models"
	"otsu-obliterator/internal/views/components"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// MainView represents the main application view using MVC pattern
type MainView struct {
	// UI Components
	window        fyne.Window
	mainContainer *fyne.Container
	toolbar       *components.Toolbar
	imageDisplay  *components.ImageDisplay
	paramPanel    *components.ParameterPanel
	statusBar     *components.StatusBar
	progressBar   *components.ProgressBar

	// Event handlers - connected to controller
	loadImageHandler       func()
	saveImageHandler       func()
	processImageHandler    func()
	cancelProcessingHandler func()
	algorithmChangeHandler func(string)
	parameterChangeHandler func(string, interface{})
}

// NewMainView creates a new main view
func NewMainView(window fyne.Window) *MainView {
	view := &MainView{
		window: window,
	}

	view.initializeComponents()
	view.buildLayout()
	view.setupEventHandlers()

	return view
}

// initializeComponents creates all UI components
func (mv *MainView) initializeComponents() {
	mv.toolbar = components.NewToolbar()
	mv.imageDisplay = components.NewImageDisplay()
	mv.paramPanel = components.NewParameterPanel()
	mv.statusBar = components.NewStatusBar()
	mv.progressBar = components.NewProgressBar()
}

// buildLayout constructs the main layout
func (mv *MainView) buildLayout() {
	// Create main content area
	contentArea := container.NewVBox(
		mv.imageDisplay.GetContainer(),
		mv.paramPanel.GetContainer(),
	)

	// Create toolbar and status area
	topArea := container.NewVBox(
		mv.toolbar.GetContainer(),
		mv.progressBar.GetContainer(),
	)

	bottomArea := mv.statusBar.GetContainer()

	// Main layout with border container
	mv.mainContainer = container.NewBorder(
		topArea,   // top
		bottomArea, // bottom
		nil,       // left
		nil,       // right
		contentArea, // center
	)

	mv.window.SetContent(mv.mainContainer)
}

// setupEventHandlers connects internal component events
func (mv *MainView) setupEventHandlers() {
	// Toolbar events
	mv.toolbar.SetLoadHandler(func() {
		if mv.loadImageHandler != nil {
			fyne.Do(func() {
				mv.loadImageHandler()
			})
		}
	})

	mv.toolbar.SetSaveHandler(func() {
		if mv.saveImageHandler != nil {
			fyne.Do(func() {
				mv.saveImageHandler()
			})
		}
	})

	mv.toolbar.SetProcessHandler(func() {
		if mv.processImageHandler != nil {
			fyne.Do(func() {
				mv.processImageHandler()
			})
		}
	})

	mv.toolbar.SetCancelHandler(func() {
		if mv.cancelProcessingHandler != nil {
			fyne.Do(func() {
				mv.cancelProcessingHandler()
			})
		}
	})

	mv.toolbar.SetAlgorithmChangeHandler(func(algorithm string) {
		if mv.algorithmChangeHandler != nil {
			fyne.Do(func() {
				mv.algorithmChangeHandler(algorithm)
			})
		}
	})

	// Parameter panel events
	mv.paramPanel.SetParameterChangeHandler(func(name string, value interface{}) {
		if mv.parameterChangeHandler != nil {
			fyne.Do(func() {
				mv.parameterChangeHandler(name, value)
			})
		}
	})
}

// Event handler setters - called by controller

// SetLoadImageHandler sets the handler for load image requests
func (mv *MainView) SetLoadImageHandler(handler func()) {
	mv.loadImageHandler = handler
}

// SetSaveImageHandler sets the handler for save image requests
func (mv *MainView) SetSaveImageHandler(handler func()) {
	mv.saveImageHandler = handler
}

// SetProcessImageHandler sets the handler for process image requests
func (mv *MainView) SetProcessImageHandler(handler func()) {
	mv.processImageHandler = handler
}

// SetCancelProcessingHandler sets the handler for cancel processing requests
func (mv *MainView) SetCancelProcessingHandler(handler func()) {
	mv.cancelProcessingHandler = handler
}

// SetAlgorithmChangeHandler sets the handler for algorithm changes
func (mv *MainView) SetAlgorithmChangeHandler(handler func(string)) {
	mv.algorithmChangeHandler = handler
}

// SetParameterChangeHandler sets the handler for parameter changes
func (mv *MainView) SetParameterChangeHandler(handler func(string, interface{})) {
	mv.parameterChangeHandler = handler
}

// UI update methods - called by controller

// SetOriginalImage updates the original image display
func (mv *MainView) SetOriginalImage(img image.Image) {
	fyne.Do(func() {
		mv.imageDisplay.SetOriginalImage(img)
	})
}

// SetProcessedImage updates the processed image display
func (mv *MainView) SetProcessedImage(img image.Image) {
	fyne.Do(func() {
		mv.imageDisplay.SetProcessedImage(img)
	})
}

// UpdateAlgorithmParameters updates the parameter panel for a new algorithm
func (mv *MainView) UpdateAlgorithmParameters(algorithm string, parameters map[string]interface{}) {
	fyne.Do(func() {
		mv.paramPanel.UpdateParameters(algorithm, parameters)
		mv.toolbar.SetCurrentAlgorithm(algorithm)
	})
}

// UpdateStatus updates the status bar message
func (mv *MainView) UpdateStatus(status string) {
	fyne.Do(func() {
		mv.statusBar.SetStatus(status)
	})
}

// UpdateProcessingProgress updates the progress bar
func (mv *MainView) UpdateProcessingProgress(stage string, progress float64) {
	fyne.Do(func() {
		mv.progressBar.SetProgress(progress)
		mv.progressBar.SetStage(stage)
	})
}

// SetProcessingActive updates UI state for processing
func (mv *MainView) SetProcessingActive(active bool) {
	fyne.Do(func() {
		mv.toolbar.SetProcessingActive(active)
		mv.progressBar.SetVisible(active)
		
		if active {
			mv.progressBar.SetProgress(0.0)
			mv.progressBar.SetStage("Initializing...")
		} else {
			mv.progressBar.SetProgress(1.0)
			mv.progressBar.SetStage("Complete")
		}
	})
}

// UpdateSegmentationMetrics updates the metrics display
func (mv *MainView) UpdateSegmentationMetrics(metrics *models.SegmentationMetrics) {
	if metrics == nil {
		return
	}

	fyne.Do(func() {
		mv.toolbar.SetSegmentationMetrics(
			metrics.IoU,
			metrics.DiceCoefficient,
			metrics.MisclassificationError,
			metrics.RegionUniformity,
			metrics.BoundaryAccuracy,
		)
	})
}

// ShowError displays an error dialog
func (mv *MainView) ShowError(title string, err error) {
	fyne.Do(func() {
		dialog.ShowError(err, mv.window)
	})
}

// ShowInfo displays an information dialog
func (mv *MainView) ShowInfo(title, message string) {
	fyne.Do(func() {
		dialog.ShowInformation(title, message, mv.window)
	})
}

// ShowConfirm displays a confirmation dialog
func (mv *MainView) ShowConfirm(title, message string, callback func(bool)) {
	fyne.Do(func() {
		dialog.ShowConfirm(title, message, callback, mv.window)
	})
}

// ShowFileDialog displays a file selection dialog
func (mv *MainView) ShowFileDialog(callback func(fyne.URIReadCloser, error)) {
	fyne.Do(func() {
		dialog.ShowFileOpen(callback, mv.window)
	})
}

// ShowSaveDialog displays a file save dialog
func (mv *MainView) ShowSaveDialog(callback func(fyne.URIWriteCloser, error)) {
	fyne.Do(func() {
		dialog.ShowFileSave(callback, mv.window)
	})
}

// GetWindow returns the main window
func (mv *MainView) GetWindow() fyne.Window {
	return mv.window
}

// GetContainer returns the main container
func (mv *MainView) GetContainer() *fyne.Container {
	return mv.mainContainer
}

// SetWindowTitle updates the window title
func (mv *MainView) SetWindowTitle(title string) {
	fyne.Do(func() {
		mv.window.SetTitle(title)
	})
}

// EnableImageOperations enables/disables image-dependent operations
func (mv *MainView) EnableImageOperations(enabled bool) {
	fyne.Do(func() {
		mv.toolbar.EnableImageOperations(enabled)
	})
}

// SetImageInfo updates image information display
func (mv *MainView) SetImageInfo(width, height, channels int, format string) {
	fyne.Do(func() {
		mv.statusBar.SetImageInfo(width, height, channels, format)
	})
}

// SetMemoryInfo updates memory usage information
func (mv *MainView) SetMemoryInfo(used, total int64) {
	fyne.Do(func() {
		mv.statusBar.SetMemoryInfo(used, total)
	})
}

// ResetView resets the view to initial state
func (mv *MainView) ResetView() {
	fyne.Do(func() {
		mv.imageDisplay.ClearImages()
		mv.paramPanel.Reset()
		mv.statusBar.Reset()
		mv.progressBar.Reset()
		mv.toolbar.Reset()
	})
}

// Show displays the view
func (mv *MainView) Show() {
	fyne.Do(func() {
		mv.window.Show()
	})
}

// Hide hides the view
func (mv *MainView) Hide() {
	fyne.Do(func() {
		mv.window.Hide()
	})
}

// Close closes the view
func (mv *MainView) Close() {
	fyne.Do(func() {
		mv.window.Close()
	})
}

// Refresh refreshes the entire view
func (mv *MainView) Refresh() {
	fyne.Do(func() {
		mv.mainContainer.Refresh()
	})
}

// SetTheme applies a theme to the view
func (mv *MainView) SetTheme(theme fyne.Theme) {
	fyne.Do(func() {
		fyne.CurrentApp().Settings().SetTheme(theme)
		mv.Refresh()
	})
}

// GetImageDisplay returns the image display component
func (mv *MainView) GetImageDisplay() *components.ImageDisplay {
	return mv.imageDisplay
}

// GetParameterPanel returns the parameter panel component
func (mv *MainView) GetParameterPanel() *components.ParameterPanel {
	return mv.paramPanel
}

// GetToolbar returns the toolbar component
func (mv *MainView) GetToolbar() *components.Toolbar {
	return mv.toolbar
}

// ViewState represents the current state of the view
type ViewState struct {
	HasOriginalImage   bool
	HasProcessedImage  bool
	IsProcessing       bool
	CurrentAlgorithm   string
	ParameterCount     int
	StatusMessage      string
	ProgressValue      float64
	ProgressStage      string
}

// GetViewState returns the current view state
func (mv *MainView) GetViewState() ViewState {
	return ViewState{
		HasOriginalImage:  mv.imageDisplay.HasOriginalImage(),
		HasProcessedImage: mv.imageDisplay.HasProcessedImage(),
		IsProcessing:      mv.progressBar.IsVisible(),
		CurrentAlgorithm:  mv.toolbar.GetCurrentAlgorithm(),
		ParameterCount:    mv.paramPanel.GetParameterCount(),
		StatusMessage:     mv.statusBar.GetStatus(),
		ProgressValue:     mv.progressBar.GetProgress(),
		ProgressStage:     mv.progressBar.GetStage(),
	}
}

// ApplyViewState applies a view state
func (mv *MainView) ApplyViewState(state ViewState) {
	fyne.Do(func() {
		mv.toolbar.SetCurrentAlgorithm(state.CurrentAlgorithm)
		mv.statusBar.SetStatus(state.StatusMessage)
		mv.progressBar.SetProgress(state.ProgressValue)
		mv.progressBar.SetStage(state.ProgressStage)
		mv.progressBar.SetVisible(state.IsProcessing)
		mv.toolbar.SetProcessingActive(state.IsProcessing)
	})
}

// UpdateLayout adjusts the layout based on window size
func (mv *MainView) UpdateLayout() {
	fyne.Do(func() {
		// Get window size
		size := mv.window.Content().Size()
		
		// Adjust layout based on aspect ratio
		if size.Width > size.Height*1.5 {
			// Wide layout - arrange components horizontally
			mv.adjustForWideLayout()
		} else {
			// Tall layout - arrange components vertically
			mv.adjustForTallLayout()
		}
		
		mv.mainContainer.Refresh()
	})
}

// adjustForWideLayout optimizes layout for wide windows
func (mv *MainView) adjustForWideLayout() {
	// Reorganize for horizontal layout
	leftPanel := container.NewVBox(
		mv.imageDisplay.GetContainer(),
	)
	
	rightPanel := container.NewVBox(
		mv.paramPanel.GetContainer(),
	)
	
	contentArea := container.NewHSplit(leftPanel, rightPanel)
	contentArea.SetOffset(0.7) // 70% for images, 30% for parameters
	
	topArea := container.NewVBox(
		mv.toolbar.GetContainer(),
		mv.progressBar.GetContainer(),
	)
	
	mv.mainContainer.Objects = []fyne.CanvasObject{}
	mv.mainContainer = container.NewBorder(
		topArea,
		mv.statusBar.GetContainer(),
		nil,
		nil,
		contentArea,
	)
	
	mv.window.SetContent(mv.mainContainer)
}

// adjustForTallLayout optimizes layout for tall windows
func (mv *MainView) adjustForTallLayout() {
	// Reorganize for vertical layout
	contentArea := container.NewVBox(
		mv.imageDisplay.GetContainer(),
		mv.paramPanel.GetContainer(),
	)
	
	topArea := container.NewVBox(
		mv.toolbar.GetContainer(),
		mv.progressBar.GetContainer(),
	)
	
	mv.mainContainer.Objects = []fyne.CanvasObject{}
	mv.mainContainer = container.NewBorder(
		topArea,
		mv.statusBar.GetContainer(),
		nil,
		nil,
		contentArea,
	)
	
	mv.window.SetContent(mv.mainContainer)
}

// SetFullscreen toggles fullscreen mode
func (mv *MainView) SetFullscreen(fullscreen bool) {
	fyne.Do(func() {
		mv.window.SetFullScreen(fullscreen)
	})
}

// IsFullscreen returns true if window is in fullscreen mode
func (mv *MainView) IsFullscreen() bool {
	return mv.window.FullScreen()
}

// SetAlwaysOnTop sets window to always stay on top
func (mv *MainView) SetAlwaysOnTop(onTop bool) {
	fyne.Do(func() {
		// Note: This functionality may not be available in all Fyne versions
		// mv.window.SetOnTop(onTop)
	})
}

// Minimize minimizes the window
func (mv *MainView) Minimize() {
	fyne.Do(func() {
		// Note: This functionality may not be available in all Fyne versions
		// mv.window.SetIcon(fyne.CurrentApp().Icon())
	})
}

// Center centers the window on screen
func (mv *MainView) Center() {
	fyne.Do(func() {
		mv.window.CenterOnScreen()
	})
}

// Resize changes the window size
func (mv *MainView) Resize(width, height float32) {
	fyne.Do(func() {
		mv.window.Resize(fyne.NewSize(width, height))
	})
}

// GetSize returns the current window size
func (mv *MainView) GetSize() fyne.Size {
	return mv.window.Content().Size()
}

// SetMinSize sets the minimum window size
func (mv *MainView) SetMinSize(width, height float32) {
	fyne.Do(func() {
		// Note: Implementation depends on Fyne version capabilities
		mv.window.Resize(fyne.NewSize(
			max(width, mv.window.Content().Size().Width),
			max(height, mv.window.Content().Size().Height),
		))
	})
}

// ShowAboutDialog displays application information
func (mv *MainView) ShowAboutDialog(appName, version, description string) {
	fyne.Do(func() {
		content := container.NewVBox(
			widget.NewLabel(appName),
			widget.NewLabel(fmt.Sprintf("Version: %s", version)),
			widget.NewLabel(""),
			widget.NewLabel(description),
			widget.NewLabel(""),
			widget.NewLabel("Built with Go 1.24, Fyne v2.6.1, and GoCV v0.41.0"),
		)
		
		dialog.ShowCustom("About", "Close", content, mv.window)
	})
}

// ShowPreferences displays application preferences dialog
func (mv *MainView) ShowPreferences() {
	fyne.Do(func() {
		// Create preferences dialog content
		content := container.NewVBox(
			widget.NewLabel("Application Preferences"),
			widget.NewSeparator(),
			widget.NewLabel("Coming soon..."),
		)
		
		dialog.ShowCustom("Preferences", "Close", content, mv.window)
	})
}

// max returns the maximum of two float32 values
func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}