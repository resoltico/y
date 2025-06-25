package pipeline

import (
	"image"
	"io"
	"time"

	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

// ImageProcessor defines the contract for image processing algorithms
type ImageProcessor interface {
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
	SaveImage(writer fyne.URIWriteCloser, imageData *ImageData) error
	GetOriginalImage() *ImageData
	GetProcessedImage() *ImageData
	CalculatePSNR(original, processed *ImageData) float64
	CalculateSSIM(original, processed *ImageData) float64
}

// MemoryManager handles OpenCV Mat memory management
type MemoryManager interface {
	GetMat(rows, cols int, matType gocv.MatType) (*safe.Mat, error)
	ReleaseMat(mat *safe.Mat)
	Cleanup()
}

// DebugManager handles logging and debugging
type DebugManager interface {
	LogInfo(component, message string)
	LogError(component string, err error)
	LogWarning(component, message string)
	StartTiming(operation string) time.Time
	EndTiming(operation string, startTime time.Time)
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
}
