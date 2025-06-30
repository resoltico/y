package models

import (
	"fmt"
	"image"
	"sync"
	"time"

	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
)

// ImageData represents an image with its metadata and processing state
type ImageData struct {
	ID          string
	Image       image.Image
	Mat         *safe.Mat
	Width       int
	Height      int
	Channels    int
	Format      string
	OriginalURI fyne.URI
	LoadTime    time.Time
	ProcessTime time.Duration
	Metadata    ImageMetadata
}

// ImageMetadata contains additional information about the image
type ImageMetadata struct {
	FileSize    int64
	ColorSpace  string
	BitDepth    int
	Compression string
	DPI         float64
	Author      string
	Software    string
	Keywords    []string
}

// ProcessingResult contains the output of image processing operations
type ProcessingResult struct {
	ProcessedImage *ImageData
	Algorithm      string
	Parameters     map[string]interface{}
	Metrics        *SegmentationMetrics
	ProcessTime    time.Duration
	MemoryUsed     int64
}

// SegmentationMetrics contains quality evaluation metrics
type SegmentationMetrics struct {
	IoU                    float64
	DiceCoefficient        float64
	MisclassificationError float64
	RegionUniformity       float64
	BoundaryAccuracy       float64
	HausdorffDistance      float64
}

// ImageRepository manages image data storage and retrieval
type ImageRepository struct {
	mu               sync.RWMutex
	originalImage    *ImageData
	processedImages  map[string]*ImageData
	processingHistory []ProcessingResult
	maxHistorySize   int
}

// NewImageRepository creates a new image repository
func NewImageRepository() *ImageRepository {
	return &ImageRepository{
		processedImages:  make(map[string]*ImageData),
		processingHistory: make([]ProcessingResult, 0),
		maxHistorySize:   10,
	}
}

// SetOriginalImage stores the original loaded image
func (r *ImageRepository) SetOriginalImage(img *ImageData) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.originalImage != nil && r.originalImage.Mat != nil {
		r.originalImage.Mat.Close()
	}
	r.originalImage = img
}

// GetOriginalImage retrieves the original image
func (r *ImageRepository) GetOriginalImage() *ImageData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.originalImage
}

// AddProcessedImage stores a processed image result
func (r *ImageRepository) AddProcessedImage(result ProcessingResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store processed image
	imageID := fmt.Sprintf("%s_%d", result.Algorithm, time.Now().Unix())
	result.ProcessedImage.ID = imageID
	r.processedImages[imageID] = result.ProcessedImage

	// Add to processing history
	r.processingHistory = append(r.processingHistory, result)

	// Limit history size
	if len(r.processingHistory) > r.maxHistorySize {
		// Remove oldest entry and clean up its image
		oldest := r.processingHistory[0]
		if oldImage, exists := r.processedImages[oldest.ProcessedImage.ID]; exists {
			if oldImage.Mat != nil {
				oldImage.Mat.Close()
			}
			delete(r.processedImages, oldest.ProcessedImage.ID)
		}
		r.processingHistory = r.processingHistory[1:]
	}
}

// GetLatestProcessedImage returns the most recently processed image
func (r *ImageRepository) GetLatestProcessedImage() *ImageData {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.processingHistory) == 0 {
		return nil
	}

	latest := r.processingHistory[len(r.processingHistory)-1]
	return latest.ProcessedImage
}

// GetProcessedImage retrieves a specific processed image by ID
func (r *ImageRepository) GetProcessedImage(id string) *ImageData {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.processedImages[id]
}

// GetProcessingHistory returns the history of processing operations
func (r *ImageRepository) GetProcessingHistory() []ProcessingResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	history := make([]ProcessingResult, len(r.processingHistory))
	copy(history, r.processingHistory)
	return history
}

// GetLatestMetrics returns metrics from the most recent processing
func (r *ImageRepository) GetLatestMetrics() *SegmentationMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.processingHistory) == 0 {
		return nil
	}

	return r.processingHistory[len(r.processingHistory)-1].Metrics
}

// ClearProcessedImages removes all processed images and history
func (r *ImageRepository) ClearProcessedImages() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up Mat resources
	for _, img := range r.processedImages {
		if img.Mat != nil {
			img.Mat.Close()
		}
	}

	r.processedImages = make(map[string]*ImageData)
	r.processingHistory = make([]ProcessingResult, 0)
}

// ClearAll removes all images including original
func (r *ImageRepository) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clean up original image
	if r.originalImage != nil && r.originalImage.Mat != nil {
		r.originalImage.Mat.Close()
		r.originalImage = nil
	}

	// Clean up processed images
	for _, img := range r.processedImages {
		if img.Mat != nil {
			img.Mat.Close()
		}
	}

	r.processedImages = make(map[string]*ImageData)
	r.processingHistory = make([]ProcessingResult, 0)
}

// GetImageStats returns statistics about stored images
func (r *ImageRepository) GetImageStats() ImageStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := ImageStats{
		HasOriginal:        r.originalImage != nil,
		ProcessedCount:     len(r.processedImages),
		HistorySize:        len(r.processingHistory),
		TotalMemoryUsage:   r.calculateMemoryUsage(),
		AverageProcessTime: r.calculateAverageProcessTime(),
	}

	return stats
}

// ImageStats contains statistics about the image repository
type ImageStats struct {
	HasOriginal        bool
	ProcessedCount     int
	HistorySize        int
	TotalMemoryUsage   int64
	AverageProcessTime time.Duration
}

// calculateMemoryUsage estimates total memory usage of stored images
func (r *ImageRepository) calculateMemoryUsage() int64 {
	var total int64

	if r.originalImage != nil {
		total += int64(r.originalImage.Width * r.originalImage.Height * r.originalImage.Channels)
	}

	for _, img := range r.processedImages {
		total += int64(img.Width * img.Height * img.Channels)
	}

	return total
}

// calculateAverageProcessTime computes average processing time
func (r *ImageRepository) calculateAverageProcessTime() time.Duration {
	if len(r.processingHistory) == 0 {
		return 0
	}

	var total time.Duration
	for _, result := range r.processingHistory {
		total += result.ProcessTime
	}

	return total / time.Duration(len(r.processingHistory))
}

// Shutdown releases all resources
func (r *ImageRepository) Shutdown() {
	r.ClearAll()
}