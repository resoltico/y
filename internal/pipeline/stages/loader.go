package stages

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"
	"otsu-obliterator/internal/pipeline"
)

type Loader struct {
	memoryManager *memory.Manager
	debugManager  *debug.Manager
}

func NewLoader(memMgr *memory.Manager, debugMgr *debug.Manager) *Loader {
	return &Loader{
		memoryManager: memMgr,
		debugManager:  debugMgr,
	}
}

func (l *Loader) LoadImage(reader fyne.URIReadCloser) (*pipeline.ImageData, error) {
	startTime := time.Now()
	
	originalURI := reader.URI()
	uriExtension := strings.ToLower(originalURI.Extension())
	
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with standard library: %w", err)
	}

	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with OpenCV: %w", err)
	}

	safeMat, err := safe.NewMatFromMat(mat)
	mat.Close() // Close original mat, use safe wrapper
	if err != nil {
		return nil, fmt.Errorf("failed to create safe Mat: %w", err)
	}

	actualFormat := l.determineActualFormat(uriExtension, standardLibFormat)

	bounds := img.Bounds()
	imageData := &pipeline.ImageData{
		Image:       img,
		Mat:         safeMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    safeMat.Channels(),
		Format:      actualFormat,
		OriginalURI: originalURI,
	}

	l.debugManager.LogInfo("PipelineLoader", 
		fmt.Sprintf("Loaded image: %dx%d, %d channels, format: %s, time: %v",
			imageData.Width, imageData.Height, imageData.Channels, actualFormat, time.Since(startTime)))

	return imageData, nil
}

func (l *Loader) determineActualFormat(uriExtension, stdLibFormat string) string {
	switch uriExtension {
	case ".tiff", ".tif":
		return "tiff"
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".bmp":
		return "bmp"
	case ".gif":
		return "gif"
	case ".webp":
		return "webp"
	default:
		if stdLibFormat != "" {
			return stdLibFormat
		}
		return "unknown"
	}
}