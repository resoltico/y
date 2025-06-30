package services

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/models"
	"otsu-obliterator/internal/opencv/conversion"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"
)

// ProcessingService handles image processing operations
type ProcessingService struct {
	memoryManager    *memory.Manager
	algorithmManager *algorithms.Manager
	imageRepo        *models.ImageRepository
	configRepo       *models.ProcessingConfiguration
	stateRepo        *models.ProcessingStateRepository
	workerPool       chan struct{}
	mu               sync.RWMutex
}

// NewProcessingService creates a new processing service
func NewProcessingService(
	memMgr *memory.Manager,
	imageRepo *models.ImageRepository,
	configRepo *models.ProcessingConfiguration,
	stateRepo *models.ProcessingStateRepository,
) *ProcessingService {
	// Initialize worker pool
	workers := make(chan struct{}, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		workers <- struct{}{}
	}

	return &ProcessingService{
		memoryManager:    memMgr,
		algorithmManager: algorithms.NewManager(),
		imageRepo:        imageRepo,
		configRepo:       configRepo,
		stateRepo:        stateRepo,
		workerPool:       workers,
	}
}

// ProcessImage processes an image using the specified algorithm
func (ps *ProcessingService) ProcessImage(ctx context.Context, algorithmName string) (*models.ProcessingResult, error) {
	// Get original image
	originalImage := ps.imageRepo.GetOriginalImage()
	if originalImage == nil {
		return nil, fmt.Errorf("no original image loaded")
	}

	// Get algorithm parameters
	params, err := ps.configRepo.GetAlgorithmParameters(algorithmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm parameters: %w", err)
	}

	// Check if processing is already active
	if ps.stateRepo.IsProcessing() {
		return nil, fmt.Errorf("processing already in progress")
	}

	// Start processing state tracking
	ps.stateRepo.StartProcessing(algorithmName)
	defer ps.stateRepo.CompleteProcessing()

	// Acquire worker from pool
	select {
	case <-ps.workerPool:
		defer func() { ps.workerPool <- struct{}{} }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	startTime := time.Now()
	var memoryBefore, memoryAfter memory.Stats
	memoryBefore.AllocCount, memoryBefore.DeallocCount, memoryBefore.UsedMemory = ps.memoryManager.GetStats()

	// Process the image
	result, err := ps.processImageInternal(ctx, originalImage, algorithmName, params.Parameters)
	if err != nil {
		ps.stateRepo.CancelProcessing()
		return nil, err
	}

	processingTime := time.Since(startTime)
	memoryAfter.AllocCount, memoryAfter.DeallocCount, memoryAfter.UsedMemory = ps.memoryManager.GetStats()

	// Calculate metrics
	metrics, err := ps.calculateSegmentationMetrics(originalImage, result)
	if err != nil {
		// Don't fail the whole operation for metrics calculation failure
		metrics = &models.SegmentationMetrics{}
	}

	// Create processing result
	processingResult := &models.ProcessingResult{
		ProcessedImage: result,
		Algorithm:      algorithmName,
		Parameters:     params.Parameters,
		Metrics:        metrics,
		ProcessTime:    processingTime,
		MemoryUsed:     memoryAfter.UsedMemory - memoryBefore.UsedMemory,
	}

	// Store result in repository
	ps.imageRepo.AddProcessedImage(*processingResult)

	return processingResult, nil
}

// processImageInternal handles the actual image processing
func (ps *ProcessingService) processImageInternal(
	ctx context.Context,
	inputImage *models.ImageData,
	algorithmName string,
	parameters map[string]interface{},
) (*models.ImageData, error) {
	// Validate input
	if err := safe.ValidateMatForOperation(inputImage.Mat, "image processing"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Get algorithm instance
	algorithm, err := ps.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm: %w", err)
	}

	// Update processing stage
	ps.stateRepo.UpdateProgress("Initializing algorithm", 0.1)

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Process with context if algorithm supports it
	var resultMat *safe.Mat
	if contextualAlg, ok := algorithm.(algorithms.ContextualAlgorithm); ok {
		ps.stateRepo.UpdateProgress("Processing with context support", 0.2)
		resultMat, err = contextualAlg.ProcessWithContext(ctx, inputImage.Mat, parameters)
	} else {
		ps.stateRepo.UpdateProgress("Processing", 0.2)
		resultMat, err = algorithm.Process(inputImage.Mat, parameters)
	}

	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	if resultMat == nil {
		return nil, fmt.Errorf("algorithm returned nil result")
	}

	// Update progress
	ps.stateRepo.UpdateProgress("Converting result", 0.8)

	// Check for cancellation again
	select {
	case <-ctx.Done():
		ps.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, ctx.Err()
	default:
	}

	// Convert Mat to Image
	resultImage, err := conversion.MatToImage(resultMat)
	if err != nil {
		ps.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion failed: %w", err)
	}

	// Create result image data
	bounds := resultImage.Bounds()
	resultData := &models.ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      inputImage.Format,
		OriginalURI: inputImage.OriginalURI,
		LoadTime:    time.Now(),
		ProcessTime: 0, // Will be set by caller
		Metadata:    inputImage.Metadata,
	}

	// Update metadata
	resultData.Metadata.Software = fmt.Sprintf("Otsu Obliterator - %s", algorithmName)

	ps.stateRepo.UpdateProgress("Complete", 1.0)

	return resultData, nil
}

// calculateSegmentationMetrics computes quality metrics for the processed result
func (ps *ProcessingService) calculateSegmentationMetrics(original, processed *models.ImageData) (*models.SegmentationMetrics, error) {
	if original.Width != processed.Width || original.Height != processed.Height {
		return nil, fmt.Errorf("image dimensions do not match")
	}

	// Use simplified metrics calculation for now
	metrics := &models.SegmentationMetrics{}

	rows := original.Height
	cols := original.Width
	totalPixels := float64(rows * cols)

	var truePositive, falsePositive, falseNegative float64
	var foregroundSum, backgroundSum float64
	var foregroundCount, backgroundCount int

	// Calculate basic segmentation metrics
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Get pixel intensities
			originalVal := ps.getPixelIntensity(original.Image, x, y)
			processedVal := ps.getPixelIntensity(processed.Image, x, y)

			// Simple threshold-based ground truth
			groundTruth := originalVal > 127
			segmentedBinary := processedVal > 127

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
		
		// Simple uniformity measure
		uniformity := 1.0 - abs(foregroundMean-backgroundMean)/255.0
		if uniformity < 0 {
			uniformity = 0
		}
		metrics.RegionUniformity = uniformity
	} else {
		metrics.RegionUniformity = 0.5
	}

	// Simple boundary accuracy estimation
	metrics.BoundaryAccuracy = (metrics.IoU + metrics.DiceCoefficient) / 2.0
	metrics.HausdorffDistance = (1.0 - metrics.IoU) * 10.0

	return metrics, nil
}

// CancelProcessing cancels the current processing operation
func (ps *ProcessingService) CancelProcessing() {
	ps.stateRepo.CancelProcessing()
}

// GetProcessingState returns the current processing state
func (ps *ProcessingService) GetProcessingState() models.ProcessingState {
	return ps.stateRepo.GetState()
}

// IsProcessing returns true if processing is currently active
func (ps *ProcessingService) IsProcessing() bool {
	return ps.stateRepo.IsProcessing()
}

// GetAvailableAlgorithms returns list of available algorithms
func (ps *ProcessingService) GetAvailableAlgorithms() []string {
	return ps.algorithmManager.GetAvailableAlgorithms()
}

// ValidateAlgorithmParameters validates parameters for a specific algorithm
func (ps *ProcessingService) ValidateAlgorithmParameters(algorithmName string, parameters map[string]interface{}) error {
	algorithm, err := ps.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		return fmt.Errorf("algorithm not found: %w", err)
	}

	return algorithm.ValidateParameters(parameters)
}

// GetDefaultParameters returns default parameters for an algorithm
func (ps *ProcessingService) GetDefaultParameters(algorithmName string) (map[string]interface{}, error) {
	algorithm, err := ps.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		return nil, fmt.Errorf("algorithm not found: %w", err)
	}

	return algorithm.GetDefaultParameters(), nil
}

// GetProcessingHistory returns the history of processing operations
func (ps *ProcessingService) GetProcessingHistory() []models.ProcessingResult {
	return ps.imageRepo.GetProcessingHistory()
}

// GetLatestResult returns the most recent processing result
func (ps *ProcessingService) GetLatestResult() *models.ProcessingResult {
	history := ps.imageRepo.GetProcessingHistory()
	if len(history) == 0 {
		return nil
	}
	return &history[len(history)-1]
}

// ClearHistory clears the processing history
func (ps *ProcessingService) ClearHistory() {
	ps.imageRepo.ClearProcessedImages()
}

// getPixelIntensity extracts grayscale intensity from a pixel
func (ps *ProcessingService) getPixelIntensity(img image.Image, x, y int) uint8 {
	r, g, b, _ := img.At(x, y).RGBA()
	// Convert to grayscale using luminance formula
	intensity := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256.0
	return uint8(intensity)
}

// abs returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Stats represents memory statistics
type Stats struct {
	AllocCount   int64
	DeallocCount int64
	UsedMemory   int64
}

// ProcessingStats contains processing performance statistics
type ProcessingStats struct {
	TotalProcessed    int
	AverageTime       time.Duration
	TotalMemoryUsed   int64
	SuccessfulRuns    int
	FailedRuns        int
	LastProcessingTime time.Time
}

// GetProcessingStats returns processing performance statistics
func (ps *ProcessingService) GetProcessingStats() ProcessingStats {
	history := ps.imageRepo.GetProcessingHistory()
	
	stats := ProcessingStats{
		TotalProcessed: len(history),
	}

	if len(history) == 0 {
		return stats
	}

	var totalTime time.Duration
	var totalMemory int64
	
	for _, result := range history {
		totalTime += result.ProcessTime
		totalMemory += result.MemoryUsed
		stats.LastProcessingTime = result.ProcessedImage.LoadTime
	}

	stats.SuccessfulRuns = len(history) // All entries in history are successful
	stats.AverageTime = totalTime / time.Duration(len(history))
	stats.TotalMemoryUsed = totalMemory

	return stats
}

// OptimizeMemoryUsage triggers memory optimization
func (ps *ProcessingService) OptimizeMemoryUsage() {
	// Force garbage collection
	runtime.GC()
	runtime.GC() // Double collection for better cleanup
	
	// Get current memory stats
	alloc, dealloc, used := ps.memoryManager.GetStats()
	
	// Log memory optimization (in real implementation, use proper logger)
	_ = alloc + dealloc + used // Suppress unused variable warnings
}

// SetWorkerCount updates the number of worker goroutines
func (ps *ProcessingService) SetWorkerCount(count int) {
	if count <= 0 {
		count = 1
	}
	if count > runtime.NumCPU()*2 {
		count = runtime.NumCPU() * 2
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Create new worker pool
	newPool := make(chan struct{}, count)
	for i := 0; i < count; i++ {
		newPool <- struct{}{}
	}
	
	ps.workerPool = newPool
}

// GetWorkerCount returns the current number of workers
func (ps *ProcessingService) GetWorkerCount() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return cap(ps.workerPool)
}

// Shutdown releases all resources
func (ps *ProcessingService) Shutdown() {
	// Cancel any ongoing processing
	ps.CancelProcessing()
	
	// Clear all data
	ps.imageRepo.ClearAll()
	
	// Final memory optimization
	ps.OptimizeMemoryUsage()
}