package pipeline

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

type imageLoader struct {
	memoryManager MemoryManager
	debugManager  DebugManager
}

func (l *imageLoader) LoadFromReader(reader fyne.URIReadCloser) (*ImageData, error) {
	startTime := l.debugManager.StartTiming("LoadFromReader")
	defer l.debugManager.EndTiming("LoadFromReader", startTime)

	originalURI := reader.URI()
	uriExtension := strings.ToLower(originalURI.Extension())

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return l.LoadFromBytes(data, uriExtension)
}

func (l *imageLoader) LoadFromBytes(data []byte, format string) (*ImageData, error) {
	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with standard library: %w", err)
	}

	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with OpenCV: %w", err)
	}

	safeMat, err := safe.NewMatFromMat(mat)
	mat.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to create safe Mat: %w", err)
	}

	actualFormat := l.determineActualFormat(format, standardLibFormat)
	bounds := img.Bounds()

	imageData := &ImageData{
		Image:    img,
		Mat:      safeMat,
		Width:    bounds.Dx(),
		Height:   bounds.Dy(),
		Channels: safeMat.Channels(),
		Format:   actualFormat,
	}

	l.debugManager.LogInfo("ImageLoader",
		fmt.Sprintf("Loaded image: %dx%d, %d channels, format: %s",
			imageData.Width, imageData.Height, imageData.Channels, actualFormat))

	return imageData, nil
}

func (l *imageLoader) determineActualFormat(uriExtension, stdLibFormat string) string {
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
