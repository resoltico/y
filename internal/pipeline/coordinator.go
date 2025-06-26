package pipeline

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/opencv/memory"

	"fyne.io/fyne/v2"
)

type Coordinator struct {
	mu               sync.RWMutex
	originalImage    *ImageData
	processedImage   *ImageData
	memoryManager    *memory.Manager
	debugCoord       debug.Coordinator
	logger           debug.Logger
	algorithmManager *algorithms.Manager
	loader           ImageLoader
	processor        ImageProcessor
	saver            ImageSaver
}

func NewCoordinator(memMgr *memory.Manager, debugCoord debug.Coordinator) *Coordinator {
	algMgr := algorithms.NewManager()
	logger := debugCoord.Logger()

	coord := &Coordinator{
		memoryManager:    memMgr,
		debugCoord:       debugCoord,
		logger:           logger,
		algorithmManager: algMgr,
	}

	// Initialize internal components with debug capabilities
	coord.loader = &imageLoader{
		memoryManager: memMgr,
		logger:        logger,
		timingTracker: debugCoord.TimingTracker(),
	}

	coord.saver = &imageSaver{
		logger:        logger,
		timingTracker: debugCoord.TimingTracker(),
	}

	logger.Info("PipelineCoordinator", "initialized", nil)

	return coord
}

func (c *Coordinator) LoadImage(reader fyne.URIReadCloser) (*ImageData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := c.debugCoord.TimingTracker().StartTiming("coordinator_load_image")
	defer c.debugCoord.TimingTracker().EndTiming(ctx)

	imageData, err := c.loader.LoadFromReader(reader)
	if err != nil {
		c.logger.Error("PipelineCoordinator", err, map[string]interface{}{
			"operation": "load_image",
		})
		return nil, err
	}

	if c.originalImage != nil {
		if c.originalImage.Mat != nil {
			c.memoryManager.ReleaseMat(c.originalImage.Mat, "original_image")
		}
		c.originalImage = nil
	}

	c.originalImage = imageData

	c.logger.Info("PipelineCoordinator", "image loaded", map[string]interface{}{
		"width":    imageData.Width,
		"height":   imageData.Height,
		"channels": imageData.Channels,
		"format":   imageData.Format,
	})

	return imageData, nil
}

func (c *Coordinator) ProcessImage(algorithmName string, params map[string]interface{}) (*ImageData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	ctx := c.debugCoord.TimingTracker().StartTiming("coordinator_process_image")
	defer c.debugCoord.TimingTracker().EndTiming(ctx)

	algorithm, err := c.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		c.logger.Error("PipelineCoordinator", err, map[string]interface{}{
			"algorithm": algorithmName,
		})
		return nil, fmt.Errorf("failed to get algorithm: %w", err)
	}

	processor := &imageProcessor{
		memoryManager:    c.memoryManager,
		logger:           c.logger,
		timingTracker:    c.debugCoord.TimingTracker(),
		algorithmManager: c.algorithmManager,
	}

	processedData, err := processor.ProcessImage(c.originalImage, algorithm, params)
	if err != nil {
		c.logger.Error("PipelineCoordinator", err, map[string]interface{}{
			"algorithm": algorithmName,
		})
		return nil, err
	}

	if c.processedImage != nil {
		if c.processedImage.Mat != nil {
			c.memoryManager.ReleaseMat(c.processedImage.Mat, "processed_image")
		}
		c.processedImage = nil
	}

	c.processedImage = processedData

	c.logger.Info("PipelineCoordinator", "image processed", map[string]interface{}{
		"algorithm": algorithmName,
		"width":     processedData.Width,
		"height":    processedData.Height,
	})

	return processedData, nil
}

func (c *Coordinator) SaveImage(writer fyne.URIWriteCloser, imageData *ImageData) error {
	ctx := c.debugCoord.TimingTracker().StartTiming("coordinator_save_image")
	defer c.debugCoord.TimingTracker().EndTiming(ctx)

	err := c.saver.SaveToWriter(writer, imageData, "")
	if err != nil {
		c.logger.Error("PipelineCoordinator", err, map[string]interface{}{
			"operation": "save_image",
		})
		return err
	}

	c.logger.Info("PipelineCoordinator", "image saved", map[string]interface{}{
		"path": writer.URI().Path(),
	})

	return nil
}

func (c *Coordinator) SaveImageToWriter(writer io.Writer, imageData *ImageData, format string) error {
	ctx := c.debugCoord.TimingTracker().StartTiming("coordinator_save_image_with_format")
	defer c.debugCoord.TimingTracker().EndTiming(ctx)

	err := c.saver.SaveToWriter(writer, imageData, strings.ToLower(format))
	if err != nil {
		c.logger.Error("PipelineCoordinator", err, map[string]interface{}{
			"operation": "save_image_with_format",
			"format":    format,
		})
		return err
	}

	c.logger.Info("PipelineCoordinator", "image saved with format", map[string]interface{}{
		"format": format,
	})

	return nil
}

func (c *Coordinator) GetOriginalImage() *ImageData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.originalImage
}

func (c *Coordinator) GetProcessedImage() *ImageData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.processedImage
}

func (c *Coordinator) CalculatePSNR(original, processed *ImageData) float64 {
	// Placeholder implementation - replace with actual PSNR calculation
	return 27.04
}

func (c *Coordinator) CalculateSSIM(original, processed *ImageData) float64 {
	// Placeholder implementation - replace with actual SSIM calculation
	return 0.8829
}

func (c *Coordinator) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("PipelineCoordinator", "cleanup started", nil)

	if c.originalImage != nil && c.originalImage.Mat != nil {
		c.memoryManager.ReleaseMat(c.originalImage.Mat, "original_image")
		c.originalImage = nil
	}

	if c.processedImage != nil && c.processedImage.Mat != nil {
		c.memoryManager.ReleaseMat(c.processedImage.Mat, "processed_image")
		c.processedImage = nil
	}

	c.logger.Info("PipelineCoordinator", "cleanup completed", nil)
}
