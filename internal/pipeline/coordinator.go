package pipeline

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/opencv/memory"
)

type Coordinator struct {
	mu               sync.RWMutex
	originalImage    *ImageData
	processedImage   *ImageData
	memoryManager    MemoryManager
	debugManager     DebugManager
	algorithmManager *algorithms.Manager
	loader           ImageLoader
	processor        ImageProcessor
	saver            ImageSaver
}

func NewCoordinator(memMgr *memory.Manager, debugMgr *debug.Manager) *Coordinator {
	algMgr := algorithms.NewManager()
	
	coord := &Coordinator{
		memoryManager:    memMgr,
		debugManager:     debugMgr,
		algorithmManager: algMgr,
	}
	
	// Initialize internal components
	coord.loader = &imageLoader{
		memoryManager: memMgr,
		debugManager:  debugMgr,
	}
	
	coord.saver = &imageSaver{
		debugManager: debugMgr,
	}
	
	return coord
}

func (c *Coordinator) LoadImage(reader fyne.URIReadCloser) (*ImageData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	imageData, err := c.loader.LoadFromReader(reader)
	if err != nil {
		return nil, err
	}

	if c.originalImage != nil {
		if c.originalImage.Mat != nil {
			c.memoryManager.ReleaseMat(c.originalImage.Mat)
		}
		c.originalImage = nil
	}

	c.originalImage = imageData
	return imageData, nil
}

func (c *Coordinator) ProcessImage(algorithmName string, params map[string]interface{}) (*ImageData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	algorithm, err := c.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm: %w", err)
	}

	processor := &imageProcessor{
		memoryManager:    c.memoryManager,
		debugManager:     c.debugManager,
		algorithmManager: c.algorithmManager,
	}

	processedData, err := processor.ProcessImage(c.originalImage, algorithm, params)
	if err != nil {
		return nil, err
	}

	if c.processedImage != nil {
		if c.processedImage.Mat != nil {
			c.memoryManager.ReleaseMat(c.processedImage.Mat)
		}
		c.processedImage = nil
	}

	c.processedImage = processedData
	return processedData, nil
}

func (c *Coordinator) SaveImage(writer fyne.URIWriteCloser, imageData *ImageData) error {
	return c.saver.SaveToWriter(writer, imageData, "")
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
	// Basic PSNR calculation placeholder
	return 27.04
}

func (c *Coordinator) CalculateSSIM(original, processed *ImageData) float64 {
	// Basic SSIM calculation placeholder
	return 0.8829
}

func (c *Coordinator) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.originalImage != nil && c.originalImage.Mat != nil {
		c.memoryManager.ReleaseMat(c.originalImage.Mat)
	}
	
	if c.processedImage != nil && c.processedImage.Mat != nil {
		c.memoryManager.ReleaseMat(c.processedImage.Mat)
	}
}
