package pipeline

import (
	"sync"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
	"image"

	"otsu-obliterator/debug"
)

type ImageData struct {
	Image       image.Image
	Mat         gocv.Mat
	Width       int
	Height      int
	Channels    int
	Format      string   // Track original format
	OriginalURI fyne.URI // Track original file URI
}

type ImagePipeline struct {
	mu               sync.RWMutex
	originalImage    *ImageData
	processedImage   *ImageData
	debugManager     *debug.Manager
	progressCallback func(float64)
	statusCallback   func(string)
}

func NewImagePipeline(debugManager *debug.Manager) *ImagePipeline {
	return &ImagePipeline{
		debugManager: debugManager,
	}
}

func (pipeline *ImagePipeline) SetProgressCallback(callback func(float64)) {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	pipeline.progressCallback = callback
}

func (pipeline *ImagePipeline) SetStatusCallback(callback func(string)) {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()
	pipeline.statusCallback = callback
}

func (pipeline *ImagePipeline) updateProgress(progress float64) {
	if pipeline.progressCallback != nil {
		pipeline.progressCallback(progress)
	}
}

func (pipeline *ImagePipeline) updateStatus(status string) {
	if pipeline.statusCallback != nil {
		pipeline.statusCallback(status)
	}
}

func (pipeline *ImagePipeline) Cleanup() {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()

	if pipeline.originalImage != nil {
		pipeline.originalImage.Mat.Close()
		pipeline.originalImage = nil
	}

	if pipeline.processedImage != nil {
		pipeline.processedImage.Mat.Close()
		pipeline.processedImage = nil
	}
}
