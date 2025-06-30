package controllers

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"otsu-obliterator/internal/models"
	"otsu-obliterator/internal/services"
	"otsu-obliterator/internal/views"

	"fyne.io/fyne/v2"
)

// MainController orchestrates the application using MVC pattern
type MainController struct {
	// Services
	imageService      *services.ImageService
	processingService *services.ProcessingService

	// Models/Repositories
	imageRepo    *models.ImageRepository
	configRepo   *models.ProcessingConfiguration
	stateRepo    *models.ProcessingStateRepository

	// Views
	mainView *views.MainView

	// State management
	mu                   sync.RWMutex
	currentWindow        fyne.Window
	processingCancelFunc context.CancelFunc
	lastImageLoad        time.Time
	
	// Event handlers
	eventHandlers map[string][]EventHandler
	eventMu       sync.RWMutex
}

// EventHandler represents a function that handles application events
type EventHandler func(data interface{}) error

// NewMainController creates a new main controller
func NewMainController(
	imageService *services.ImageService,
	processingService *services.ProcessingService,
	imageRepo *models.ImageRepository,
	configRepo *models.ProcessingConfiguration,
	stateRepo *models.ProcessingStateRepository,
) *MainController {
	controller := &MainController{
		imageService:      imageService,
		processingService: processingService,
		imageRepo:         imageRepo,
		configRepo:        configRepo,
		stateRepo:         stateRepo,
		eventHandlers:     make(map[string][]EventHandler),
	}

	controller.initializeEventHandlers()
	return controller
}

// SetMainView associates the main view with this controller
func (mc *MainController) SetMainView(view *views.MainView) {
	mc.mainView = view
	mc.setupViewEventHandlers()
}

// SetWindow sets the main application window
func (mc *MainController) SetWindow(window fyne.Window) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.currentWindow = window
}

// LoadImage handles image loading requests
func (mc *MainController) LoadImage() {
	if mc.currentWindow == nil {
		mc.handleError("Window not available", fmt.Errorf("main window not set"))
		return
	}

	// Use fyne.Do for thread-safe UI operations in Fyne v2.6+
	fyne.Do(func() {
		mc.showFileLoadDialog()
	})
}

// SaveImage handles image saving requests
func (mc *MainController) SaveImage() {
	processedImg := mc.imageRepo.GetLatestProcessedImage()
	if processedImg == nil {
		mc.handleError("Save failed", fmt.Errorf("no processed image available"))
		return
	}

	fyne.Do(func() {
		mc.showFileSaveDialog(processedImg)
	})
}

// ProcessImage initiates image processing with the current algorithm
func (mc *MainController) ProcessImage() {
	// Check if image is loaded
	originalImg := mc.imageRepo.GetOriginalImage()
	if originalImg == nil {
		mc.handleError("Processing failed", fmt.Errorf("no image loaded"))
		return
	}

	// Check if already processing
	if mc.processingService.IsProcessing() {
		return
	}

	// Get current algorithm and parameters
	algorithm := mc.configRepo.GetCurrentAlgorithm()
	
	// Update UI state to show processing started
	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.SetProcessingActive(true)
			mc.mainView.UpdateStatus("Starting processing...")
		}
	})

	// Start processing in background
	go mc.performImageProcessing(algorithm)
}

// CancelProcessing cancels ongoing processing
func (mc *MainController) CancelProcessing() {
	mc.mu.Lock()
	if mc.processingCancelFunc != nil {
		mc.processingCancelFunc()
	}
	mc.mu.Unlock()

	mc.processingService.CancelProcessing()

	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.SetProcessingActive(false)
			mc.mainView.UpdateStatus("Processing cancelled")
		}
	})
}

// ChangeAlgorithm switches to a different algorithm
func (mc *MainController) ChangeAlgorithm(algorithm string) {
	err := mc.configRepo.SetCurrentAlgorithm(algorithm)
	if err != nil {
		mc.handleError("Algorithm change failed", err)
		return
	}

	// Get new algorithm parameters
	params, err := mc.configRepo.GetAlgorithmParameters(algorithm)
	if err != nil {
		mc.handleError("Parameter retrieval failed", err)
		return
	}

	// Update view with new parameters
	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.UpdateAlgorithmParameters(algorithm, params.Parameters)
			mc.mainView.UpdateStatus(fmt.Sprintf("Algorithm changed to %s", algorithm))
		}
	})

	// Emit algorithm change event
	mc.emitEvent("algorithm_changed", algorithm)
}

// UpdateParameter updates an algorithm parameter
func (mc *MainController) UpdateParameter(name string, value interface{}) {
	algorithm := mc.configRepo.GetCurrentAlgorithm()
	
	err := mc.configRepo.SetAlgorithmParameter(algorithm, name, value)
	if err != nil {
		mc.handleError("Parameter update failed", err)
		return
	}

	// Emit parameter change event
	mc.emitEvent("parameter_changed", map[string]interface{}{
		"algorithm": algorithm,
		"parameter": name,
		"value":     value,
	})
}

// GetApplicationState returns the current application state
func (mc *MainController) GetApplicationState() ApplicationState {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	originalImg := mc.imageRepo.GetOriginalImage()
	processedImg := mc.imageRepo.GetLatestProcessedImage()
	processingState := mc.stateRepo.GetState()
	currentAlgorithm := mc.configRepo.GetCurrentAlgorithm()

	return ApplicationState{
		HasOriginalImage:  originalImg != nil,
		HasProcessedImage: processedImg != nil,
		IsProcessing:      processingState.IsActive,
		CurrentAlgorithm:  currentAlgorithm,
		ProcessingStage:   processingState.CurrentStage,
		ProcessingProgress: processingState.Progress,
		LastImageLoad:     mc.lastImageLoad,
	}
}

// ApplicationState represents the current state of the application
type ApplicationState struct {
	HasOriginalImage   bool
	HasProcessedImage  bool
	IsProcessing       bool
	CurrentAlgorithm   string
	ProcessingStage    string
	ProcessingProgress float64
	LastImageLoad      time.Time
}

// performImageProcessing handles the actual processing in background
func (mc *MainController) performImageProcessing(algorithm string) {
	// Create cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	mc.mu.Lock()
	mc.processingCancelFunc = cancel
	mc.mu.Unlock()

	// Start progress monitoring
	go mc.monitorProcessingProgress()

	// Perform processing
	result, err := mc.processingService.ProcessImage(ctx, algorithm)

	// Clear cancellation function
	mc.mu.Lock()
	mc.processingCancelFunc = nil
	mc.mu.Unlock()

	// Update UI based on result
	fyne.Do(func() {
		if mc.mainView == nil {
			return
		}

		mc.mainView.SetProcessingActive(false)

		if err != nil {
			if ctx.Err() != nil {
				mc.mainView.UpdateStatus("Processing cancelled")
			} else {
				mc.mainView.UpdateStatus("Processing failed")
				mc.handleError("Processing failed", err)
			}
			return
		}

		if result != nil && result.ProcessedImage != nil {
			mc.mainView.SetProcessedImage(result.ProcessedImage.Image)
			mc.mainView.UpdateSegmentationMetrics(result.Metrics)
			mc.mainView.UpdateStatus("Processing completed")

			// Emit processing complete event
			mc.emitEvent("processing_complete", result)
		} else {
			mc.mainView.UpdateStatus("Processing failed - no result")
		}
	})
}

// monitorProcessingProgress tracks processing progress and updates UI
func (mc *MainController) monitorProcessingProgress() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		state := mc.stateRepo.GetState()
		if !state.IsActive {
			break
		}

		// Update UI with current progress
		fyne.Do(func() {
			if mc.mainView != nil {
				mc.mainView.UpdateProcessingProgress(state.CurrentStage, state.Progress)
			}
		})
	}
}

// showFileLoadDialog displays the file selection dialog
func (mc *MainController) showFileLoadDialog() {
	if mc.currentWindow == nil {
		return
	}

	// This would be implemented with actual Fyne file dialog
	// For now, this is a placeholder that shows the pattern
	
	// dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
	//     if err != nil {
	//         mc.handleError("File selection error", err)
	//         return
	//     }
	//     if reader == nil {
	//         return
	//     }
	//     
	//     go mc.loadImageFromReader(reader)
	// }, mc.currentWindow)
}

// showFileSaveDialog displays the file save dialog
func (mc *MainController) showFileSaveDialog(imageData *models.ImageData) {
	if mc.currentWindow == nil {
		return
	}

	// This would be implemented with actual Fyne file dialog
	// For now, this is a placeholder that shows the pattern
	
	// dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
	//     if err != nil {
	//         mc.handleError("File save error", err)
	//         return
	//     }
	//     if writer == nil {
	//         return
	//     }
	//     
	//     go mc.saveImageToWriter(writer, imageData)
	// }, mc.currentWindow)
}

// loadImageFromReader loads an image from a file reader
func (mc *MainController) loadImageFromReader(reader fyne.URIReadCloser) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.UpdateStatus("Loading image...")
		}
	})

	imageData, err := mc.imageService.LoadImage(ctx, reader)
	if err != nil {
		fyne.Do(func() {
			mc.handleError("Image load failed", err)
			if mc.mainView != nil {
				mc.mainView.UpdateStatus("Ready")
			}
		})
		return
	}

	mc.mu.Lock()
	mc.lastImageLoad = time.Now()
	mc.mu.Unlock()

	// Update UI with loaded image
	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.SetOriginalImage(imageData.Image)
			mc.mainView.SetProcessedImage(nil) // Clear previous result
			mc.mainView.UpdateStatus("Image loaded")
		}
	})

	// Emit image loaded event
	mc.emitEvent("image_loaded", imageData)
}

// saveImageToWriter saves an image to a file writer
func (mc *MainController) saveImageToWriter(writer fyne.URIWriteCloser, imageData *models.ImageData) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.UpdateStatus("Saving image...")
		}
	})

	err := mc.imageService.SaveImage(ctx, writer, imageData, "")
	
	fyne.Do(func() {
		if mc.mainView != nil {
			if err != nil {
				mc.handleError("Image save failed", err)
				mc.mainView.UpdateStatus("Save failed")
			} else {
				mc.mainView.UpdateStatus("Image saved")
			}
		}
	})

	if err == nil {
		// Emit image saved event
		mc.emitEvent("image_saved", imageData)
	}
}

// Event system methods

// initializeEventHandlers sets up default event handlers
func (mc *MainController) initializeEventHandlers() {
	mc.addEventListener("image_loaded", mc.onImageLoaded)
	mc.addEventListener("processing_complete", mc.onProcessingComplete)
	mc.addEventListener("algorithm_changed", mc.onAlgorithmChanged)
}

// setupViewEventHandlers connects view events to controller methods
func (mc *MainController) setupViewEventHandlers() {
	if mc.mainView == nil {
		return
	}

	// Connect view callbacks to controller methods
	mc.mainView.SetLoadImageHandler(mc.LoadImage)
	mc.mainView.SetSaveImageHandler(mc.SaveImage)
	mc.mainView.SetProcessImageHandler(mc.ProcessImage)
	mc.mainView.SetCancelProcessingHandler(mc.CancelProcessing)
	mc.mainView.SetAlgorithmChangeHandler(mc.ChangeAlgorithm)
	mc.mainView.SetParameterChangeHandler(mc.UpdateParameter)
}

// addEventListener adds an event handler for a specific event type
func (mc *MainController) addEventListener(eventType string, handler EventHandler) {
	mc.eventMu.Lock()
	defer mc.eventMu.Unlock()

	if mc.eventHandlers[eventType] == nil {
		mc.eventHandlers[eventType] = make([]EventHandler, 0)
	}
	mc.eventHandlers[eventType] = append(mc.eventHandlers[eventType], handler)
}

// emitEvent triggers all handlers for a specific event type
func (mc *MainController) emitEvent(eventType string, data interface{}) {
	mc.eventMu.RLock()
	handlers := mc.eventHandlers[eventType]
	mc.eventMu.RUnlock()

	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h(data); err != nil {
				mc.handleError(fmt.Sprintf("Event handler error (%s)", eventType), err)
			}
		}(handler)
	}
}

// Event handlers

// onImageLoaded handles image loaded events
func (mc *MainController) onImageLoaded(data interface{}) error {
	imageData, ok := data.(*models.ImageData)
	if !ok {
		return fmt.Errorf("invalid data type for image_loaded event")
	}

	// Update memory optimization based on image size
	imageSize := int64(imageData.Width * imageData.Height * imageData.Channels)
	if imageSize > 10*1024*1024 { // 10MB
		mc.processingService.OptimizeMemoryUsage()
	}

	return nil
}

// onProcessingComplete handles processing completion events
func (mc *MainController) onProcessingComplete(data interface{}) error {
	result, ok := data.(*models.ProcessingResult)
	if !ok {
		return fmt.Errorf("invalid data type for processing_complete event")
	}

	// Perform post-processing cleanup
	mc.processingService.OptimizeMemoryUsage()

	// Update performance settings based on processing time
	if result.ProcessTime > 30*time.Second {
		perfSettings := mc.configRepo.GetPerformanceSettings()
		if perfSettings.MaxWorkers < runtime.NumCPU() {
			perfSettings.MaxWorkers = runtime.NumCPU()
			mc.configRepo.UpdatePerformanceSettings(perfSettings)
		}
	}

	return nil
}

// onAlgorithmChanged handles algorithm change events
func (mc *MainController) onAlgorithmChanged(data interface{}) error {
	algorithm, ok := data.(string)
	if !ok {
		return fmt.Errorf("invalid data type for algorithm_changed event")
	}

	// Clear processed images when algorithm changes
	mc.imageRepo.ClearProcessedImages()

	// Log algorithm change (in real implementation, use proper logger)
	_ = algorithm // Suppress unused variable warning

	return nil
}

// handleError handles application errors with consistent UI feedback
func (mc *MainController) handleError(title string, err error) {
	// In a real implementation, this would use proper logging
	// and potentially show user-friendly error dialogs
	
	fyne.Do(func() {
		if mc.mainView != nil {
			mc.mainView.ShowError(title, err)
		}
	})
}

// Shutdown performs cleanup when the application closes
func (mc *MainController) Shutdown() {
	// Cancel any ongoing processing
	mc.CancelProcessing()

	// Clean up services
	mc.imageService.Cleanup()
	mc.processingService.Shutdown()

	// Final memory cleanup
	runtime.GC()
}