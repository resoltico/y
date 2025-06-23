package pipeline

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"strings"
	"time"

	"fyne.io/fyne/v2"
)

func (pipeline *ImagePipeline) SaveImage(writer fyne.URIWriteCloser) error {
	pipeline.mu.RLock()
	defer pipeline.mu.RUnlock()

	if pipeline.processedImage == nil {
		return fmt.Errorf("no processed image to save")
	}

	startTime := pipeline.debugManager.StartTiming("image_save")
	defer pipeline.debugManager.EndTiming("image_save", startTime)

	saveStartTime := time.Now()

	// Determine format from URI, fallback to original format
	uri := writer.URI()
	ext := strings.ToLower(uri.Extension())

	var saveFormat string
	switch ext {
	case ".jpg", ".jpeg":
		saveFormat = "jpeg"
	case ".png":
		saveFormat = "png"
	case ".tiff", ".tif":
		saveFormat = "tiff"
	case ".bmp":
		saveFormat = "bmp"
	default:
		// Use original format if available
		if pipeline.originalImage != nil {
			saveFormat = pipeline.originalImage.Format
		} else {
			saveFormat = "png" // Default fallback
		}
	}

	pipeline.debugManager.LogImageConversion(pipeline.processedImage.Format, saveFormat, time.Since(saveStartTime))

	switch saveFormat {
	case "jpeg":
		return jpeg.Encode(writer, pipeline.processedImage.Image, &jpeg.Options{Quality: 95})
	case "png":
		return png.Encode(writer, pipeline.processedImage.Image)
	case "tiff", "bmp":
		// Go standard library doesn't support TIFF/BMP encoding
		// Fall back to PNG for processed images
		pipeline.debugManager.LogWarning("Pipeline", fmt.Sprintf("%s encoding not supported, saving as PNG", strings.ToUpper(saveFormat)))
		return png.Encode(writer, pipeline.processedImage.Image)
	default:
		// Default to PNG
		return png.Encode(writer, pipeline.processedImage.Image)
	}
}
