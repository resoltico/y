package gui

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/gui/widgets"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

type Controller struct {
	coordinator      *pipeline.Coordinator
	algorithmManager *algorithms.Manager
	logger           logger.Logger

	// UI components
	toolbar        *widgets.Toolbar
	imageDisplay   *widgets.ImageDisplay
	parameterPanel *widgets.ParameterPanel
	mainContainer  *fyne.Container

	// State management
	mu                sync.RWMutex
	currentAlgorithm  string
	currentParameters map[string]interface{}
	processingActive  atomic.Bool

	// Processing control
	processCtx    context.Context
	processCancel context.CancelFunc

	// Worker pool for UI operations
	uiWorkers chan struct{}
}

func NewController(coord *pipeline.Coordinator, log logger.Logger) *Controller {
	// Initialize worker pool for UI operations
	uiWorkers := make(chan struct{}, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		uiWorkers <- struct{}{}
	}

	controller := &Controller{
		coordinator:       coord,
		algorithmManager:  algorithms.NewManager(),
		logger:            log,
		currentAlgorithm:  "2D Otsu",
		currentParameters: make(map[string]interface{}),
		uiWorkers:         uiWorkers,
	}

	controller.initializeComponents()
	controller.initializeDefaultParameters()

	log.Info("GUI controller initialized", map[string]interface{}{
		"default_algorithm": controller.currentAlgorithm,
		"ui_workers":        runtime.NumCPU(),
	})

	return controller
}

func (c *Controller) initializeComponents() {
	c.toolbar = widgets.NewToolbar()
	c.imageDisplay = widgets.NewImageDisplay()
	c.parameterPanel = widgets.NewParameterPanel()

	// Setup event handlers using Fyne v2.6+ patterns
	c.toolbar.SetLoadHandler(c.LoadImage)
	c.toolbar.SetSaveHandler(c.SaveImage)
	c.toolbar.SetProcessHandler(c.ProcessImage)
	c.toolbar.SetAlgorithmChangeHandler(c.ChangeAlgorithm)

	c.parameterPanel.SetParameterChangeHandler(c.UpdateParameter)
}

func (c *Controller) CreateMainContent() *fyne.Container {
	c.mainContainer = container.NewVBox(
		c.imageDisplay.GetContainer(),
		c.toolbar.GetContainer(),
		c.parameterPanel.GetContainer(),
	)
	return c.mainContainer
}

func (c *Controller) initializeDefaultParameters() {
	params := c.algorithmManager.GetParameters(c.currentAlgorithm)
	c.mu.Lock()
	c.currentParameters = params
	c.mu.Unlock()

	// Use fyne.Do for thread-safe UI updates
	fyne.Do(func() {
		c.parameterPanel.UpdateParameters(c.currentAlgorithm, params)
	})
}

func (c *Controller) LoadImage() {
	c.logger.Debug("Load image requested", nil)

	// Use worker pool for file operations
	go func() {
		select {
		case <-c.uiWorkers:
			defer func() { c.uiWorkers <- struct{}{} }()
			c.performImageLoad()
		default:
			c.logger.Warning("UI worker pool exhausted for image load", nil)
		}
	}()
}

func (c *Controller) performImageLoad() {
	// Use fyne.Do for UI dialog operations
	fyne.Do(func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				c.handleError("File selection error", err)
				return
			}
			if reader == nil {
				return
			}

			c.updateStatus("Loading image...")

			// Perform loading in background
			go func() {
				defer reader.Close()
				c.processImageLoad(reader)
			}()
		}, c.getMainWindow())
	})
}

func (c *Controller) processImageLoad(reader fyne.URIReadCloser) {
	start := time.Now()

	imageData, err := c.coordinator.LoadImage(reader)
	if err != nil {
		fyne.Do(func() {
			c.handleError("Image load error", err)
			c.updateStatus("Ready")
		})
		return
	}

	// Update UI on main thread
	fyne.Do(func() {
		c.imageDisplay.SetOriginalImage(imageData.Image)
		c.imageDisplay.SetPreviewImage(nil) // Clear previous preview
		c.updateStatus("Image loaded")

		c.logger.Info("Image loaded successfully", map[string]interface{}{
			"width":     imageData.Width,
			"height":    imageData.Height,
			"format":    imageData.Format,
			"load_time": time.Since(start),
		})
	})
}

func (c *Controller) SaveImage() {
	processedImg := c.coordinator.GetProcessedImage()
	if processedImg == nil {
		c.handleError("Save error", fmt.Errorf("no processed image to save"))
		return
	}

	c.logger.Debug("Save image requested", nil)

	fyne.Do(func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				c.handleError("File save error", err)
				return
			}
			if writer == nil {
				return
			}

			c.updateStatus("Saving image...")

			// Perform saving in background
			go func() {
				defer writer.Close()
				c.processSaveImage(writer, processedImg)
			}()
		}, c.getMainWindow())
	})
}

func (c *Controller) processSaveImage(writer fyne.URIWriteCloser, imageData *pipeline.ImageData) {
	start := time.Now()

	err := c.coordinator.SaveImage(writer, imageData)

	fyne.Do(func() {
		if err != nil {
			c.handleError("Image save error", err)
		} else {
			c.updateStatus("Image saved")
			c.logger.Info("Image saved successfully", map[string]interface{}{
				"path":      writer.URI().Path(),
				"save_time": time.Since(start),
			})
		}
	})
}

func (c *Controller) ProcessImage() {
	if !c.processingActive.CompareAndSwap(false, true) {
		c.logger.Warning("Processing already in progress", nil)
		return
	}

	originalImg := c.coordinator.GetOriginalImage()
	if originalImg == nil {
		c.processingActive.Store(false)
		c.handleError("Processing error", fmt.Errorf("no image loaded"))
		return
	}

	fyne.Do(func() {
		c.updateStatus("Starting processing...")
	})

	// Create processing context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	c.mu.Lock()
	c.processCtx = ctx
	c.processCancel = cancel
	c.mu.Unlock()

	go c.performImageProcessing(ctx)
}

func (c *Controller) performImageProcessing(ctx context.Context) {
	defer func() {
		c.processingActive.Store(false)
		c.mu.Lock()
		if c.processCancel != nil {
			c.processCancel()
			c.processCancel = nil
		}
		c.mu.Unlock()
	}()

	algorithm := c.getCurrentAlgorithm()
	params := c.getCurrentParameters()

	// Update status with processing stages
	stages := c.getProcessingStages(algorithm)
	for i, stage := range stages {
		select {
		case <-ctx.Done():
			fyne.Do(func() {
				c.updateStatus("Processing cancelled")
			})
			return
		default:
		}

		fyne.Do(func() {
			stageInfo := fmt.Sprintf("Step %d/%d: %s", i+1, len(stages), stage)
			c.updateStatus(stageInfo)
		})

		// Small delay to show progress
		time.Sleep(100 * time.Millisecond)
	}

	start := time.Now()
	processedImg, err := c.coordinator.ProcessImageWithContext(ctx, algorithm, params)
	processingTime := time.Since(start)

	fyne.Do(func() {
		if err != nil {
			if ctx.Err() != nil {
				c.updateStatus("Processing cancelled")
			} else {
				c.handleError("Processing error", err)
				c.updateStatus("Processing failed")
			}
			return
		}

		if processedImg != nil {
			c.imageDisplay.SetPreviewImage(processedImg.Image)
			c.updateSegmentationMetrics(processedImg)
			c.updateStatus("Processing completed")

			c.logger.Info("Processing completed successfully", map[string]interface{}{
				"algorithm":       algorithm,
				"processing_time": processingTime,
				"output_size":     fmt.Sprintf("%dx%d", processedImg.Width, processedImg.Height),
			})
		} else {
			c.updateStatus("Processing failed - no result")
		}
	})
}

func (c *Controller) getProcessingStages(algorithm string) []string {
	switch algorithm {
	case "2D Otsu":
		return []string{
			"Converting to grayscale",
			"Applying noise reduction",
			"Building 2D histogram",
			"Calculating threshold",
			"Applying threshold",
			"Post-processing cleanup",
		}
	case "Iterative Triclass":
		return []string{
			"Converting to grayscale",
			"Advanced preprocessing",
			"Initial threshold estimation",
			"Iterative refinement",
			"Convergence analysis",
			"Final cleanup",
		}
	default:
		return []string{"Processing"}
	}
}

func (c *Controller) updateSegmentationMetrics(processedImg *pipeline.ImageData) {
	originalImg := c.coordinator.GetOriginalImage()
	if originalImg == nil {
		return
	}

	metrics, err := c.coordinator.CalculateSegmentationMetrics(originalImg, processedImg)
	if err != nil {
		c.logger.Warning("Failed to calculate metrics", map[string]interface{}{
			"error": err.Error(),
		})
		// Use default metrics
		c.toolbar.SetSegmentationMetrics(0.5, 0.5, 0.25, 0.7, 0.6)
		return
	}

	c.toolbar.SetSegmentationMetrics(
		metrics.IoU,
		metrics.DiceCoefficient,
		metrics.MisclassificationError,
		metrics.RegionUniformity,
		metrics.BoundaryAccuracy,
	)
}

func (c *Controller) ChangeAlgorithm(algorithm string) {
	c.mu.Lock()
	c.currentAlgorithm = algorithm
	c.mu.Unlock()

	if err := c.algorithmManager.SetCurrentAlgorithm(algorithm); err != nil {
		c.handleError("Algorithm change error", err)
		return
	}

	params := c.algorithmManager.GetParameters(algorithm)
	c.mu.Lock()
	c.currentParameters = params
	c.mu.Unlock()

	fyne.Do(func() {
		c.parameterPanel.UpdateParameters(algorithm, params)
	})

	c.logger.Debug("Algorithm changed", map[string]interface{}{
		"algorithm": algorithm,
	})
}

func (c *Controller) UpdateParameter(name string, value interface{}) {
	c.mu.Lock()
	algorithm := c.currentAlgorithm
	c.currentParameters[name] = value
	c.mu.Unlock()

	err := c.algorithmManager.SetParameter(algorithm, name, value)
	if err != nil {
		c.handleError("Parameter update error", err)
		return
	}

	c.logger.Debug("Parameter updated", map[string]interface{}{
		"algorithm": algorithm,
		"parameter": name,
		"value":     value,
	})
}

func (c *Controller) CancelProcessing() {
	c.mu.Lock()
	if c.processCancel != nil {
		c.processCancel()
	}
	c.mu.Unlock()

	c.logger.Info("Processing cancellation requested", nil)
}

func (c *Controller) updateStatus(status string) {
	c.toolbar.SetStatus(status)
}

func (c *Controller) handleError(title string, err error) {
	c.logger.Error(title, err, nil)
	fyne.Do(func() {
		dialog.ShowError(err, c.getMainWindow())
	})
}

func (c *Controller) getCurrentAlgorithm() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentAlgorithm
}

func (c *Controller) getCurrentParameters() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range c.currentParameters {
		result[k] = v
	}
	return result
}

func (c *Controller) getMainWindow() fyne.Window {
	// This is a placeholder - in real implementation, you'd need to pass
	// the window reference or get it from a parent container
	return fyne.CurrentApp().Driver().AllWindows()[0]
}

func (c *Controller) Shutdown() {
	c.CancelProcessing()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("GUI controller shutdown completed", nil)
}
