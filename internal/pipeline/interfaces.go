package pipeline

import (
	"context"
	"image"
	"io"

	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

// ImageProcessor defines the contract for image processing algorithms
type ImageProcessor interface {
	ProcessImage(inputData *ImageData, algorithm Algorithm, params map[string]interface{}) (*ImageData, error)
	ProcessImageWithContext(ctx context.Context, inputData *ImageData, algorithm Algorithm, params map[string]interface{}) (*ImageData, error)
}

// Algorithm defines the base interface for processing algorithms
type Algorithm interface {
	Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
	ValidateParameters(params map[string]interface{}) error
	GetDefaultParameters() map[string]interface{}
	GetName() string
}

// ImageLoader handles loading images from various sources
type ImageLoader interface {
	LoadFromReader(reader fyne.URIReadCloser) (*ImageData, error)
	LoadFromBytes(data []byte, format string) (*ImageData, error)
}

// ImageSaver handles saving images to various formats
type ImageSaver interface {
	SaveToWriter(writer io.Writer, imageData *ImageData, format string) error
	SaveToPath(path string, imageData *ImageData) error
}

// ProcessingCoordinator manages the image processing pipeline
type ProcessingCoordinator interface {
	LoadImage(reader fyne.URIReadCloser) (*ImageData, error)
	ProcessImage(algorithmName string, params map[string]interface{}) (*ImageData, error)
	ProcessImageWithContext(ctx context.Context, algorithmName string, params map[string]interface{}) (*ImageData, error)
	SaveImage(writer fyne.URIWriteCloser, imageData *ImageData) error
	SaveImageToWriter(writer io.Writer, imageData *ImageData, format string) error
	GetOriginalImage() *ImageData
	GetProcessedImage() *ImageData
	CalculatePSNR(original, processed *ImageData) float64
	CalculateSSIM(original, processed *ImageData) float64
	Context() context.Context
	Cancel()
}

// MemoryManager handles OpenCV Mat memory management
type MemoryManager interface {
	GetMat(rows, cols int, matType gocv.MatType, tag string) (*safe.Mat, error)
	ReleaseMat(mat *safe.Mat, tag string)
	GetUsedMemory() int64
	GetStats() (allocCount, deallocCount int64, usedMemory int64)
	Cleanup()
}

// ImageData represents processed image information
type ImageData struct {
	Image       image.Image
	Mat         *safe.Mat
	Width       int
	Height      int
	Channels    int
	Format      string
	OriginalURI fyne.URI
}

// ProcessingMetrics contains algorithm performance data
type ProcessingMetrics struct {
	ProcessingTime float64
	MemoryUsed     int64
	PSNR           float64
	SSIM           float64
	ThresholdValue float64
}
