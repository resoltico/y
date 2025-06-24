package pipeline

import (
	"fmt"
	"image"
	"io"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

func (pipeline *ImagePipeline) LoadImage(reader fyne.URIReadCloser) error {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()

	startTime := pipeline.debugManager.StartTiming("image_load")
	defer pipeline.debugManager.EndTiming("image_load", startTime)

	loadStartTime := time.Now()
	processingSteps := []string{}

	// Extract URI information
	originalURI := reader.URI()
	uriExtension := strings.ToLower(originalURI.Extension())
	uriScheme := originalURI.Scheme()
	uriPath := originalURI.Path()
	uriMimeType := originalURI.MimeType()

	processingSteps = append(processingSteps, fmt.Sprintf("URI analysis - Extension: %s, Scheme: %s, MimeType: %s", uriExtension, uriScheme, uriMimeType))

	// Read image data
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read image data: %w", err)
	}
	processingSteps = append(processingSteps, fmt.Sprintf("Read %d bytes from URI", len(data)))

	// Extract first 16 bytes for format signature analysis
	firstBytes := make([]byte, 16)
	if len(data) >= 16 {
		copy(firstBytes, data[:16])
	} else {
		copy(firstBytes, data)
	}

	pipeline.updateProgress(0.2)

	// Decode image using Go's standard library
	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	standardLibSuccess := err == nil
	standardLibError := ""
	if err != nil {
		standardLibError = err.Error()
	}

	pipeline.debugManager.LogStandardLibDecodingResult(standardLibFormat, standardLibSuccess, standardLibError)

	if err != nil {
		return fmt.Errorf("failed to decode image with standard library: %w", err)
	}
	processingSteps = append(processingSteps, fmt.Sprintf("Standard library detected format: %s", standardLibFormat))

	pipeline.updateProgress(0.4)

	// Convert to OpenCV Mat using IMDecode
	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	openCVSuccess := err == nil
	openCVError := ""
	matChannels := 0
	if err != nil {
		openCVError = err.Error()
	} else {
		matChannels = mat.Channels()
	}

	pipeline.debugManager.LogOpenCVDecodingResult(openCVSuccess, matChannels, openCVError)

	if err != nil {
		return fmt.Errorf("failed to decode image with OpenCV: %w", err)
	}
	processingSteps = append(processingSteps, "OpenCV IMDecode successful")

	pipeline.updateProgress(0.6)

	// Determine actual format using URI extension as primary source
	actualFormat := pipeline.determineActualFormat(uriExtension, standardLibFormat)
	processingSteps = append(processingSteps, fmt.Sprintf("Final format determined: %s", actualFormat))

	// Log format detection
	formatDetection := &debug.FormatDetection{
		URI:               originalURI,
		URIScheme:         uriScheme,
		URIPath:           uriPath,
		URIExtension:      uriExtension,
		URIMimeType:       uriMimeType,
		StandardLibFormat: standardLibFormat,
		OpenCVSupported:   openCVSuccess,
		FinalFormat:       actualFormat,
		DataSize:          len(data),
		FirstBytes:        firstBytes,
	}
	pipeline.debugManager.LogFormatDetection(formatDetection)

	// Check for format mismatch and log warning
	if pipeline.isFormatMismatch(uriExtension, standardLibFormat) {
		pipeline.debugManager.LogImageFormatMismatch(originalURI, uriExtension, standardLibFormat)
	}

	// Check for extension/mimetype mismatch
	if uriMimeType != "" {
		expectedFromExt := pipeline.mimeTypeFromExtension(uriExtension)
		if expectedFromExt != "" && !strings.Contains(uriMimeType, expectedFromExt) {
			pipeline.debugManager.LogExtensionMimeTypeMismatch(originalURI, expectedFromExt, uriMimeType)
		}
	}

	pipeline.updateProgress(0.8)

	// Clean up previous image
	if pipeline.originalImage != nil {
		pipeline.memoryManager.ReleaseMat(pipeline.originalImage.Mat)
	}

	// Store image data
	bounds := img.Bounds()
	pipeline.originalImage = &ImageData{
		Image:       img,
		Mat:         mat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    mat.Channels(),
		Format:      actualFormat,
		OriginalURI: originalURI,
	}

	pipeline.updateProgress(1.0)

	// Log debug information
	debugInfo := &debug.ImageDebugInfo{
		OriginalURI:      originalURI,
		ExtensionFromURI: uriExtension,
		DetectedFormat:   actualFormat,
		Width:            pipeline.originalImage.Width,
		Height:           pipeline.originalImage.Height,
		Channels:         pipeline.originalImage.Channels,
		DataSize:         len(data),
		LoadTime:         time.Since(loadStartTime),
		ProcessingSteps:  processingSteps,
	}
	pipeline.debugManager.LogImageLoad(debugInfo)

	pipeline.debugManager.LogInfo("Pipeline", fmt.Sprintf("Loaded image: %dx%d, %d channels, format: %s",
		pipeline.originalImage.Width, pipeline.originalImage.Height, pipeline.originalImage.Channels, actualFormat))

	return nil
}

func (pipeline *ImagePipeline) determineActualFormat(uriExtension, stdLibFormat string) string {
	// Prioritize URI extension as it's more reliable
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
		// Fallback to standard library detection if URI extension is unknown
		if stdLibFormat != "" {
			return stdLibFormat
		}
		return "unknown"
	}
}

func (pipeline *ImagePipeline) isFormatMismatch(uriExtension, stdLibFormat string) bool {
	if uriExtension == "" || stdLibFormat == "" {
		return false
	}

	// Check if there's a mismatch between URI extension and detected format
	expectedFromExt := pipeline.determineActualFormat(uriExtension, "")
	return expectedFromExt != stdLibFormat && expectedFromExt != "unknown"
}

func (pipeline *ImagePipeline) mimeTypeFromExtension(ext string) string {
	switch ext {
	case ".tiff", ".tif":
		return "image/tiff"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".bmp":
		return "image/bmp"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func (pipeline *ImagePipeline) GetOriginalImage() *ImageData {
	pipeline.mu.RLock()
	defer pipeline.mu.RUnlock()
	return pipeline.originalImage
}

func (pipeline *ImagePipeline) GetProcessedImage() *ImageData {
	pipeline.mu.RLock()
	defer pipeline.mu.RUnlock()
	return pipeline.processedImage
}
