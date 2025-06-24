package pipeline

import (
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline/stages"
)

type ImageData struct {
	Image       interface{}
	Mat         interface{}
	Width       int
	Height      int
	Channels    int
	Format      string
	OriginalURI fyne.URI
}

type Coordinator struct {
	mu               sync.RWMutex
	originalImage    *ImageData
	processedImage   *ImageData
	memoryManager    *memory.Manager
	debugManager     *debug.Manager
	algorithmManager *algorithms.Manager
	loader           *stages.Loader
	processor        *stages.Processor
	saver            *stages.Saver
}

func NewCoordinator(memMgr *memory.Manager, debugMgr *debug.Manager) *Coordinator {
	algMgr := algorithms.NewManager()
	
	return &Coordinator{
		memoryManager:    memMgr,
		debugManager:     debugMgr,
		algorithmManager: algMgr,
		loader:           stages.NewLoader(memMgr, debugMgr),
		processor:        stages.NewProcessor(memMgr, debugMgr, algMgr),
		saver:            stages.NewSaver(debugMgr),
	}
}

func (c *Coordinator) LoadImage(reader fyne.URIReadCloser) (*ImageData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	imageData, err := c.loader.LoadImage(reader)
	if err != nil {
		return nil, err
	}

	if c.originalImage != nil {
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

	processedData, err := c.processor.ProcessImage(c.originalImage, algorithmName, params)
	if err != nil {
		return nil, err
	}

	if c.processedImage != nil {
		c.processedImage = nil
	}

	c.processedImage = processedData
	return processedData, nil
}

func (c *Coordinator) SaveImage(writer fyne.URIWriteCloser, imageData *ImageData) error {
	return c.saver.SaveImage(writer, imageData)
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
	// Placeholder implementation
	return 27.04
}

func (c *Coordinator) CalculateSSIM(original, processed *ImageData) float64 {
	// Placeholder implementation
	return 0.8829
}