package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

type ImageData struct {
	Image       image.Image
	Mat         *safe.Mat
	Width       int
	Height      int
	Channels    int
	Format      string
	OriginalURI fyne.URI
}

type SegmentationMetrics struct {
	IoU                    float64
	DiceCoefficient        float64
	MisclassificationError float64
	RegionUniformity       float64
	BoundaryAccuracy       float64
	HausdorffDistance      float64
}

type Coordinator struct {
	mu               sync.RWMutex
	originalImage    *ImageData
	processedImage   *ImageData
	memoryManager    *memory.Manager
	logger           logger.Logger
	algorithmManager *algorithms.Manager
	ctx              context.Context
	cancel           context.CancelFunc

	// Go 1.24 worker pool for parallel operations
	workers          chan struct{}
	processingActive atomic.Bool
}

func NewCoordinator(memMgr *memory.Manager, log logger.Logger) *Coordinator {
	algMgr := algorithms.NewManager()
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize worker pool with CPU count
	workers := make(chan struct{}, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		workers <- struct{}{}
	}

	coord := &Coordinator{
		memoryManager:    memMgr,
		logger:           log,
		algorithmManager: algMgr,
		ctx:              ctx,
		cancel:           cancel,
		workers:          workers,
	}

	log.Info("Pipeline coordinator initialized", map[string]interface{}{
		"worker_count": runtime.NumCPU(),
	})

	return coord
}

func (c *Coordinator) LoadImage(reader fyne.URIReadCloser) (*ImageData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := time.Now()

	// Release previous images with modern memory management
	c.releaseImage(&c.originalImage, "original_image")
	c.releaseImage(&c.processedImage, "processed_image")

	originalURI := reader.URI()
	uriExtension := strings.ToLower(filepath.Ext(originalURI.Path()))

	bufferedReader := bufio.NewReader(reader)
	data, err := io.ReadAll(bufferedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	imageData, err := c.loadFromBytes(data, uriExtension, originalURI)
	if err != nil {
		return nil, err
	}

	c.originalImage = imageData

	c.logger.Info("Image loaded successfully", map[string]interface{}{
		"width":     imageData.Width,
		"height":    imageData.Height,
		"channels":  imageData.Channels,
		"format":    imageData.Format,
		"load_time": time.Since(start),
		"size_mb":   len(data) / (1024 * 1024),
	})

	return imageData, nil
}

func (c *Coordinator) loadFromBytes(data []byte, format string, uri fyne.URI) (*ImageData, error) {
	// Decode with standard library first
	img, standardLibFormat, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with standard library: %w", err)
	}

	// Decode with GoCV for Mat operations
	mat, err := gocv.IMDecode(data, gocv.IMReadColor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image with OpenCV: %w", err)
	}
	defer mat.Close() // Clean up original Mat

	// Create safe Mat with memory tracking
	safeMat, err := safe.NewMatFromMatWithTracker(mat, c.memoryManager, "loaded_image")
	if err != nil {
		return nil, fmt.Errorf("failed to create safe Mat: %w", err)
	}

	actualFormat := c.determineFormat(format, standardLibFormat)
	bounds := img.Bounds()

	return &ImageData{
		Image:       img,
		Mat:         safeMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    safeMat.Channels(),
		Format:      actualFormat,
		OriginalURI: uri,
	}, nil
}

func (c *Coordinator) ProcessImageWithContext(ctx context.Context, algorithmName string, params map[string]interface{}) (*ImageData, error) {
	if !c.processingActive.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("processing already in progress")
	}
	defer c.processingActive.Store(false)

	c.mu.Lock()
	if c.originalImage == nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("no image loaded")
	}
	originalImage := c.originalImage
	c.mu.Unlock()

	algorithm, err := c.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm: %w", err)
	}

	start := time.Now()
	memoryBefore := c.memoryManager.GetStats()

	// Use worker pool for processing
	select {
	case <-c.workers:
		defer func() { c.workers <- struct{}{} }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	processedData, err := c.processImageInternal(ctx, originalImage, algorithm, params)
	if err != nil {
		return nil, err
	}

	processingTime := time.Since(start)
	memoryAfter := c.memoryManager.GetStats()

	c.mu.Lock()
	c.releaseImage(&c.processedImage, "processed_image")
	c.processedImage = processedData
	c.mu.Unlock()

	// Calculate metrics
	metrics, err := c.calculateMetrics(originalImage, processedData)
	if err != nil {
		c.logger.Warning("Failed to calculate segmentation metrics", map[string]interface{}{
			"error": err.Error(),
		})
	}

	c.logger.Info("Image processing completed", map[string]interface{}{
		"algorithm":         algorithmName,
		"width":             processedData.Width,
		"height":            processedData.Height,
		"processing_time":   processingTime,
		"memory_delta":      memoryAfter.usedMemory - memoryBefore.usedMemory,
		"iou_score":         metrics.IoU,
		"dice_coefficient":  metrics.DiceCoefficient,
		"region_uniformity": metrics.RegionUniformity,
	})

	return processedData, nil
}

func (c *Coordinator) processImageInternal(ctx context.Context, inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	if err := safe.ValidateMatForOperation(inputData.Mat, "ProcessImage"); err != nil {
		return nil, err
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Process with context support
	var resultMat *safe.Mat
	var err error

	if contextualAlg, ok := algorithm.(interface {
		ProcessWithContext(context.Context, *safe.Mat, map[string]interface{}) (*safe.Mat, error)
	}); ok {
		resultMat, err = contextualAlg.ProcessWithContext(ctx, inputData.Mat, params)
	} else {
		resultMat, err = algorithm.Process(inputData.Mat, params)
	}

	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	if resultMat == nil {
		return nil, fmt.Errorf("algorithm returned nil result")
	}

	// Check cancellation again
	select {
	case <-ctx.Done():
		c.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, ctx.Err()
	default:
	}

	// Convert Mat to Image using modern approach
	resultImage, err := c.matToImage(resultMat)
	if err != nil {
		c.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion failed: %w", err)
	}

	bounds := resultImage.Bounds()
	return &ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      inputData.Format,
		OriginalURI: inputData.OriginalURI,
	}, nil
}

func (c *Coordinator) matToImage(mat *safe.Mat) (image.Image, error) {
	if err := safe.ValidateMatForOperation(mat, "MatToImage"); err != nil {
		return nil, err
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	switch channels {
	case 1:
		return c.matToGray(mat, rows, cols)
	case 3:
		return c.matToRGBA(mat, rows, cols)
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}
}

func (c *Coordinator) matToGray(mat *safe.Mat, rows, cols int) (*image.Gray, error) {
	img := image.NewGray(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value, err := mat.GetUCharAt(y, x)
			if err != nil {
				return nil, fmt.Errorf("failed to get pixel at (%d,%d): %w", x, y, err)
			}
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}

	return img, nil
}

func (c *Coordinator) matToRGBA(mat *safe.Mat, rows, cols int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b, err := mat.GetUCharAt3(y, x, 0)
			if err != nil {
				return nil, fmt.Errorf("failed to get B channel at (%d,%d): %w", x, y, err)
			}

			g, err := mat.GetUCharAt3(y, x, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to get G channel at (%d,%d): %w", x, y, err)
			}

			r, err := mat.GetUCharAt3(y, x, 2)
			if err != nil {
				return nil, fmt.Errorf("failed to get R channel at (%d,%d): %w", x, y, err)
			}

			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img, nil
}

func (c *Coordinator) SaveImage(writer fyne.URIWriteCloser, imageData *ImageData) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	ext := strings.ToLower(writer.URI().Extension())
	format := c.determineFormat(ext, imageData.Format)

	start := time.Now()
	err := c.saveToWriter(writer, imageData, format)
	if err != nil {
		return err
	}

	c.logger.Info("Image saved successfully", map[string]interface{}{
		"path":      writer.URI().Path(),
		"format":    format,
		"save_time": time.Since(start),
	})

	return nil
}

func (c *Coordinator) SaveImageToWriter(writer io.Writer, imageData *ImageData, format string) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	start := time.Now()
	err := c.saveToWriter(writer, imageData, strings.ToLower(format))
	if err != nil {
		return err
	}

	c.logger.Info("Image saved with format", map[string]interface{}{
		"format":    format,
		"save_time": time.Since(start),
	})

	return nil
}

func (c *Coordinator) saveToWriter(writer io.Writer, imageData *ImageData, format string) error {
	img, ok := imageData.Image.(image.Image)
	if !ok {
		return fmt.Errorf("image data is not a valid image")
	}

	saveFormat := format
	if saveFormat == "" {
		saveFormat = imageData.Format
	}
	if saveFormat == "" {
		saveFormat = "png"
	}

	switch saveFormat {
	case "jpeg", ".jpg", ".jpeg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 95})
	case "png", ".png":
		return png.Encode(writer, img)
	default:
		c.logger.Warning("Unsupported format, using PNG", map[string]interface{}{
			"requested_format": saveFormat,
		})
		return png.Encode(writer, img)
	}
}

func (c *Coordinator) calculateMetrics(original, segmented *ImageData) (*SegmentationMetrics, error) {
	if original.Width != segmented.Width || original.Height != segmented.Height {
		return nil, fmt.Errorf("image dimensions do not match")
	}

	// Simple segmentation quality metrics
	metrics := &SegmentationMetrics{}

	rows := original.Height
	cols := original.Width
	totalPixels := float64(rows * cols)

	var truePositive, falsePositive, falseNegative float64
	var foregroundVar, backgroundVar float64
	var foregroundSum, backgroundSum float64
	var foregroundCount, backgroundCount int

	// Calculate basic segmentation metrics
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Get original pixel intensity (assuming grayscale or using luminance)
			originalVal := c.getPixelIntensity(original.Image, x, y)
			segmentedVal := c.getPixelIntensity(segmented.Image, x, y)

			// Simple threshold-based ground truth
			groundTruth := originalVal > 127
			segmentedBinary := segmentedVal > 127

			if groundTruth && segmentedBinary {
				truePositive++
			} else if !groundTruth && segmentedBinary {
				falsePositive++
			} else if groundTruth && !segmentedBinary {
				falseNegative++
			}

			// Calculate region uniformity
			if segmentedBinary {
				foregroundSum += float64(originalVal)
				foregroundCount++
			} else {
				backgroundSum += float64(originalVal)
				backgroundCount++
			}
		}
	}

	// Calculate IoU and Dice coefficient
	intersection := truePositive
	union := truePositive + falsePositive + falseNegative
	if union > 0 {
		metrics.IoU = intersection / union
		metrics.DiceCoefficient = (2.0 * intersection) / (2.0*truePositive + falsePositive + falseNegative)
	} else {
		metrics.IoU = 1.0
		metrics.DiceCoefficient = 1.0
	}

	// Calculate misclassification error
	metrics.MisclassificationError = (falsePositive + falseNegative) / totalPixels

	// Calculate region uniformity
	if foregroundCount > 0 && backgroundCount > 0 {
		foregroundMean := foregroundSum / float64(foregroundCount)
		backgroundMean := backgroundSum / float64(backgroundCount)

		// Simple uniformity measure based on mean differences
		uniformity := 1.0 - math.Abs(foregroundMean-backgroundMean)/255.0
		metrics.RegionUniformity = math.Max(0, uniformity)
	} else {
		metrics.RegionUniformity = 0.5
	}

	// Simple boundary accuracy (placeholder)
	metrics.BoundaryAccuracy = (metrics.IoU + metrics.DiceCoefficient) / 2.0
	metrics.HausdorffDistance = (1.0 - metrics.IoU) * 10.0

	return metrics, nil
}

func (c *Coordinator) getPixelIntensity(img image.Image, x, y int) uint8 {
	r, g, b, _ := img.At(x, y).RGBA()
	// Convert to grayscale using luminance formula
	intensity := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256.0
	return uint8(intensity)
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

func (c *Coordinator) CalculateSegmentationMetrics(original, processed *ImageData) (*SegmentationMetrics, error) {
	return c.calculateMetrics(original, processed)
}

func (c *Coordinator) releaseImage(imagePtr **ImageData, tag string) {
	if *imagePtr != nil && (*imagePtr).Mat != nil {
		c.memoryManager.ReleaseMat((*imagePtr).Mat, tag)
		*imagePtr = nil
	}
}

func (c *Coordinator) determineFormat(uriExtension, stdLibFormat string) string {
	switch uriExtension {
	case ".tiff", ".tif":
		return "tiff"
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".bmp":
		return "bmp"
	default:
		if stdLibFormat != "" {
			return stdLibFormat
		}
		return "png"
	}
}

func (c *Coordinator) Shutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Pipeline coordinator shutdown starting", nil)
	c.cancel()

	c.releaseImage(&c.originalImage, "original_image")
	c.releaseImage(&c.processedImage, "processed_image")

	allocCount, deallocCount, usedMemory := c.memoryManager.GetStats()
	c.logger.Info("Pipeline coordinator shutdown completed", map[string]interface{}{
		"final_memory_usage":    usedMemory,
		"total_allocations":     allocCount,
		"total_deallocations":   deallocCount,
		"memory_leak_indicator": allocCount - deallocCount,
	})
}
