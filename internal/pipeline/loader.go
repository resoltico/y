package pipeline

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"

	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

type imageLoader struct {
	memoryManager *memory.Manager
	logger        Logger
	timingTracker TimingTracker
}

func (l *imageLoader) LoadFromReader(reader fyne.URIReadCloser) (*ImageData, error) {
	ctx := l.timingTracker.StartTiming("load_from_reader")
	defer l.timingTracker.EndTiming(ctx)

	originalURI := reader.URI()
	uriExtension := strings.ToLower(originalURI.Extension())

	l.logger.Debug("ImageLoader", "loading image", map[string]interface{}{
		"path":      originalURI.Path(),
		"extension": uriExtension,
	})

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	l.logger.Debug("ImageLoader", "image data read", map[string]interface{}{
		"size_bytes": len(data),
	})

	return l.LoadFromBytes(data, uriExtension)
}

func (l *imageLoader) LoadFromBytes(data []byte, format string) (*ImageData, error) {
	ctx := l.timingTracker.StartTiming("load_from_bytes")
	defer l.timingTracker.EndTiming(ctx)

	// Decode with standard library for Go image interface
	stdCtx := l.timingTracker.StartTiming("stdlib_decode")
	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	l.timingTracker.EndTiming(stdCtx)

	if err != nil {
		return nil, fmt.Errorf("failed to decode image with standard library: %w", err)
	}

	// Decode with OpenCV for Mat operations
	cvCtx := l.timingTracker.StartTiming("opencv_decode")
	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	l.timingTracker.EndTiming(cvCtx)

	if err != nil {
		return nil, fmt.Errorf("failed to decode image with OpenCV: %w", err)
	}

	safeMat, err := safe.NewMatFromMatWithTracker(mat, l.memoryManager, "loaded_image")
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

	l.logger.Info("ImageLoader", "image loaded successfully", map[string]interface{}{
		"width":    imageData.Width,
		"height":   imageData.Height,
		"channels": imageData.Channels,
		"format":   actualFormat,
	})

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
