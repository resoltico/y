package otsu

import (
	"fmt"
	"image"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// IterativeTriclassProcessor handles the complete iterative triclass thresholding workflow
// This replaces the original IterativeTriclassCore with better modularization and mathematical accuracy
type IterativeTriclassProcessor struct {
	params             map[string]interface{}
	debugManager       *debug.Manager
	mathProcessor      *IterativeTriclassMathProcessor
	convergenceMonitor *IterativeConvergenceMonitor
	postprocessor      *IterativeTriclassPostprocessor
	iterationState     *IterationState
}

// IterationState tracks the current state of iterative processing
type IterationState struct {
	CurrentRegion   gocv.Mat
	FinalResult     gocv.Mat
	IterationCount  int
	TotalPixels     int
	ProcessedPixels int
	ActivePixels    int
	IsInitialized   bool
}

// NewIterativeTriclassProcessor creates a new iterative triclass processor
func NewIterativeTriclassProcessor(params map[string]interface{}) *IterativeTriclassProcessor {
	debugManager := debug.NewManager()

	return &IterativeTriclassProcessor{
		params:             params,
		debugManager:       debugManager,
		mathProcessor:      NewIterativeTriclassMath(params),
		convergenceMonitor: NewIterativeConvergenceMonitor(params),
		postprocessor:      NewIterativeTriclassPostprocessor(params),
		iterationState:     &IterationState{},
	}
}

// Process applies iterative triclass thresholding with enhanced mathematical accuracy
func (processor *IterativeTriclassProcessor) Process(src gocv.Mat) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	// Validate input dimensions with more stringent checks
	if src.Rows() <= 0 || src.Cols() <= 0 {
		return gocv.NewMat(), fmt.Errorf("input Mat has invalid dimensions: %dx%d", src.Cols(), src.Rows())
	}

	if src.Rows() < 5 || src.Cols() < 5 {
		return gocv.NewMat(), fmt.Errorf("image too small for iterative processing: %dx%d (minimum 5x5)", src.Cols(), src.Rows())
	}

	processor.debugManager.LogAlgorithmStart("Iterative Triclass Enhanced", processor.params)
	startTime := time.Now()

	// Step 1: Validate parameters before processing
	if err := processor.mathProcessor.ValidateIterationParameters(); err != nil {
		return gocv.NewMat(), fmt.Errorf("parameter validation failed: %w", err)
	}

	// Step 2: Create working copy and convert to grayscale
	working := src.Clone()
	defer working.Close()

	gray, err := processor.prepareGrayscaleImage(&working)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed: %w", err)
	}
	defer gray.Close()

	// Step 3: Apply preprocessing if enabled
	preprocessed, err := processor.applyPreprocessing(&gray)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("preprocessing failed: %w", err)
	}
	defer preprocessed.Close()

	// Step 4: Initialize iteration state
	err = processor.initializeIterationState(&preprocessed)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("iteration initialization failed: %w", err)
	}
	defer processor.cleanupIterationState()

	// Step 5: Perform iterative triclass segmentation
	err = processor.performIterativeSegmentation(&preprocessed)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("iterative segmentation failed: %w", err)
	}

	// Step 6: Apply post-processing
	result, err := processor.postprocessor.ApplyPostprocessing(processor.iterationState.FinalResult)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("post-processing failed: %w", err)
	}

	// Step 7: Log final results and analysis
	totalTime := time.Since(startTime)
	processor.logFinalResults(totalTime)

	processor.debugManager.LogAlgorithmComplete("Iterative Triclass Enhanced", totalTime,
		fmt.Sprintf("%dx%d", result.Cols(), result.Rows()))

	return result, nil
}

// prepareGrayscaleImage converts input to grayscale with enhanced validation
func (processor *IterativeTriclassProcessor) prepareGrayscaleImage(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("Iterative Triclass Enhanced", "grayscale_conversion", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	gray := gocv.NewMat()
	channels := src.Channels()

	switch channels {
	case 1:
		// Already grayscale - create copy
		src.CopyTo(&gray)
	case 3:
		// Convert BGR to GRAY using latest GoCV API
		gocv.CvtColor(*src, &gray, gocv.ColorBGRToGray)
	case 4:
		// Convert BGRA to GRAY using latest GoCV API
		gocv.CvtColor(*src, &gray, gocv.ColorBGRAToGray)
	default:
		return gocv.NewMat(), fmt.Errorf("unsupported channel count: %d", channels)
	}

	if gray.Empty() {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed")
	}

	processor.debugManager.LogInfo("Iterative Triclass Enhanced",
		fmt.Sprintf("Converted %d-channel image to grayscale", channels))

	return gray, nil
}

// applyPreprocessing applies CLAHE and denoising if enabled
func (processor *IterativeTriclassProcessor) applyPreprocessing(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("Iterative Triclass Enhanced", "preprocessing", time.Since(stepTime))

	result := gocv.NewMat()

	if processor.getBoolParam("apply_preprocessing") {
		// Apply CLAHE for contrast enhancement using latest GoCV API
		clahe := gocv.NewCLAHEWithParams(2.0, image.Point{X: 8, Y: 8})
		defer clahe.Close()

		enhanced := gocv.NewMat()
		defer enhanced.Close()
		clahe.Apply(*src, &enhanced)

		// Apply denoising for better threshold calculation
		if processor.getStringParam("quality", "Fast") == "Best" {
			// Use latest denoising API with optimized parameters
			gocv.FastNlMeansDenoisingWithParams(enhanced, &result, 3.0, 7, 21)
		} else {
			enhanced.CopyTo(&result)
		}
	} else {
		src.CopyTo(&result)
	}

	return result, nil
}

// initializeIterationState prepares data structures for iterative processing
func (processor *IterativeTriclassProcessor) initializeIterationState(src *gocv.Mat) error {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("Iterative Triclass Enhanced", "state_initialization", time.Since(stepTime))

	if src.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	processor.iterationState = &IterationState{
		TotalPixels:     src.Rows() * src.Cols(),
		ProcessedPixels: 0,
		ActivePixels:    src.Rows() * src.Cols(),
		IterationCount:  0,
		IsInitialized:   true,
	}

	// Initialize current region with the entire image
	processor.iterationState.CurrentRegion = src.Clone()

	// Initialize final result as all background (0)
	processor.iterationState.FinalResult = gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	processor.iterationState.FinalResult.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Reset convergence monitor for new processing
	processor.convergenceMonitor.Reset()

	processor.debugManager.LogInfo("Iterative Triclass Enhanced",
		fmt.Sprintf("Initialized iteration state: %dx%d, %d total pixels",
			src.Cols(), src.Rows(), processor.iterationState.TotalPixels))

	return nil
}

// performIterativeSegmentation executes the main iterative triclass algorithm
func (processor *IterativeTriclassProcessor) performIterativeSegmentation(src *gocv.Mat) error {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("Iterative Triclass Enhanced", "iterative_segmentation", time.Since(stepTime))

	maxIterations := processor.getIntParam("max_iterations", 10)
	var previousThreshold float64 = -1.0

	for iteration := 0; iteration < maxIterations; iteration++ {
		iterStartTime := time.Now()

		// Check if current region has sufficient pixels to process
		activePixels := gocv.CountNonZero(processor.iterationState.CurrentRegion)
		if activePixels == 0 {
			processor.debugManager.LogInfo("Iterative Triclass Enhanced",
				fmt.Sprintf("No active pixels remaining at iteration %d", iteration))
			break
		}

		// Calculate threshold for current region
		threshold, err := processor.calculateIterationThreshold()
		if err != nil {
			return fmt.Errorf("threshold calculation failed at iteration %d: %w", iteration, err)
		}

		// Check convergence using mathematical processor
		converged, convergenceValue, err := processor.mathProcessor.CheckConvergence(threshold, previousThreshold, iteration)
		if err != nil {
			return fmt.Errorf("convergence check failed at iteration %d: %w", iteration, err)
		}

		// Perform triclass segmentation
		foreground, background, tbd, err := processor.performTriclassSegmentation(threshold)
		if err != nil {
			return fmt.Errorf("triclass segmentation failed at iteration %d: %w", iteration, err)
		}

		// Count pixels in each class
		foregroundCount := gocv.CountNonZero(foreground)
		backgroundCount := gocv.CountNonZero(background)
		tbdCount := gocv.CountNonZero(tbd)

		// Check TBD termination condition
		shouldTerminate, tbdFraction := processor.mathProcessor.CheckTBDTermination(tbdCount, processor.iterationState.TotalPixels)

		// Record iteration in convergence monitor
		processor.convergenceMonitor.RecordIteration(iteration, threshold, convergenceValue, tbdFraction,
			time.Since(iterStartTime), foregroundCount, backgroundCount, tbdCount)

		// Update final result with current classifications
		processor.updateFinalResult(&foreground, &background)

		// Check convergence or termination conditions
		if converged {
			processor.debugManager.LogInfo("Iterative Triclass Enhanced",
				fmt.Sprintf("Converged at iteration %d", iteration))
			processor.assignRemainingTBDPixels(&tbd, threshold)
			foreground.Close()
			background.Close()
			tbd.Close()
			break
		}

		if shouldTerminate {
			processor.debugManager.LogInfo("Iterative Triclass Enhanced",
				fmt.Sprintf("TBD termination at iteration %d", iteration))
			processor.assignRemainingTBDPixels(&tbd, threshold)
			foreground.Close()
			background.Close()
			tbd.Close()
			break
		}

		// Update current region to only include TBD pixels
		err = processor.updateCurrentRegion(&tbd)
		foreground.Close()
		background.Close()
		tbd.Close()

		if err != nil {
			return fmt.Errorf("region update failed at iteration %d: %w", iteration, err)
		}

		previousThreshold = threshold
		processor.iterationState.IterationCount++
	}

	return nil
}

// calculateIterationThreshold computes threshold for current active region
func (processor *IterativeTriclassProcessor) calculateIterationThreshold() (float64, error) {
	// Build histogram for active region only
	histogram, err := processor.buildRegionHistogram()
	if err != nil {
		return 0, fmt.Errorf("histogram construction failed: %w", err)
	}

	// Calculate threshold using mathematical processor
	return processor.mathProcessor.CalculateRegionThreshold(histogram)
}

// buildRegionHistogram constructs histogram only for active (non-zero) pixels
func (processor *IterativeTriclassProcessor) buildRegionHistogram() ([]int, error) {
	histBins := processor.getIntParam("histogram_bins", 64)
	histogram := make([]int, histBins)

	if processor.iterationState.CurrentRegion.Empty() {
		return histogram, fmt.Errorf("current region is empty")
	}

	rows := processor.iterationState.CurrentRegion.Rows()
	cols := processor.iterationState.CurrentRegion.Cols()

	// Build histogram only from active pixels (non-zero values)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := processor.iterationState.CurrentRegion.GetUCharAt(y, x)

			// Only include active pixels in histogram
			if pixelValue > 0 {
				bin := int(float64(pixelValue) * float64(histBins-1) / 255.0)

				// Clamp bin to valid range
				if bin < 0 {
					bin = 0
				} else if bin >= histBins {
					bin = histBins - 1
				}

				histogram[bin]++
			}
		}
	}

	return histogram, nil
}

// performTriclassSegmentation segments current region into three classes
func (processor *IterativeTriclassProcessor) performTriclassSegmentation(threshold float64) (gocv.Mat, gocv.Mat, gocv.Mat, error) {
	if processor.iterationState.CurrentRegion.Empty() {
		return gocv.NewMat(), gocv.NewMat(), gocv.NewMat(), fmt.Errorf("current region is empty")
	}

	rows := processor.iterationState.CurrentRegion.Rows()
	cols := processor.iterationState.CurrentRegion.Cols()

	// Create output masks
	foreground := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	background := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	tbd := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	// Initialize all masks to zero
	foreground.SetTo(gocv.NewScalar(0, 0, 0, 0))
	background.SetTo(gocv.NewScalar(0, 0, 0, 0))
	tbd.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Calculate adaptive thresholds using mathematical processor
	lowerThreshold, upperThreshold, err := processor.mathProcessor.CalculateTriclassThresholds(threshold)
	if err != nil {
		foreground.Close()
		background.Close()
		tbd.Close()
		return gocv.NewMat(), gocv.NewMat(), gocv.NewMat(), err
	}

	// Perform triclass segmentation
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := float64(processor.iterationState.CurrentRegion.GetUCharAt(y, x))

			// Only process active pixels
			if pixelValue > 0 {
				if pixelValue >= upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if pixelValue <= lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					// To-be-determined region
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd, nil
}

// updateFinalResult incorporates current iteration results into final segmentation
func (processor *IterativeTriclassProcessor) updateFinalResult(foreground, background *gocv.Mat) {
	rows := processor.iterationState.FinalResult.Rows()
	cols := processor.iterationState.FinalResult.Cols()

	foregroundPixels := 0
	backgroundPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Set foreground pixels to 255
			if foreground.GetUCharAt(y, x) > 0 {
				processor.iterationState.FinalResult.SetUCharAt(y, x, 255)
				foregroundPixels++
			}
			// Count background pixels for statistics
			if background.GetUCharAt(y, x) > 0 {
				backgroundPixels++
			}
			// Background pixels remain 0 (already initialized)
		}
	}

	// Update processed pixels count
	processor.iterationState.ProcessedPixels += foregroundPixels + backgroundPixels
}

// updateCurrentRegion prepares the next iteration region from TBD pixels
func (processor *IterativeTriclassProcessor) updateCurrentRegion(tbdMask *gocv.Mat) error {
	// Create new region containing only TBD pixels with their original values
	newRegion := gocv.NewMatWithSize(processor.iterationState.CurrentRegion.Rows(),
		processor.iterationState.CurrentRegion.Cols(), gocv.MatTypeCV8UC1)
	newRegion.SetTo(gocv.NewScalar(0, 0, 0, 0))

	rows := processor.iterationState.CurrentRegion.Rows()
	cols := processor.iterationState.CurrentRegion.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdMask.GetUCharAt(y, x) > 0 {
				// Copy original pixel value to new region
				originalValue := processor.iterationState.CurrentRegion.GetUCharAt(y, x)
				newRegion.SetUCharAt(y, x, originalValue)
			}
		}
	}

	// Replace current region
	processor.iterationState.CurrentRegion.Close()
	processor.iterationState.CurrentRegion = newRegion

	// Update active pixel count
	processor.iterationState.ActivePixels = gocv.CountNonZero(newRegion)

	return nil
}

// assignRemainingTBDPixels assigns final TBD pixels using simple threshold
func (processor *IterativeTriclassProcessor) assignRemainingTBDPixels(tbdMask *gocv.Mat, threshold float64) {
	rows := tbdMask.Rows()
	cols := tbdMask.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdMask.GetUCharAt(y, x) > 0 {
				// Get original pixel value
				originalValue := float64(processor.iterationState.CurrentRegion.GetUCharAt(y, x))

				// Simple threshold assignment
				if originalValue >= threshold {
					processor.iterationState.FinalResult.SetUCharAt(y, x, 255) // Foreground
				}
				// Background pixels remain 0
			}
		}
	}
}

// logFinalResults logs results and convergence analysis
func (processor *IterativeTriclassProcessor) logFinalResults(processingTime time.Duration) {
	if processor.iterationState == nil || !processor.iterationState.IsInitialized {
		return
	}

	// Calculate final statistics
	totalPixels := processor.iterationState.TotalPixels
	foregroundPixels := gocv.CountNonZero(processor.iterationState.FinalResult)
	backgroundPixels := totalPixels - foregroundPixels

	// Generate convergence report
	convergenceReport := processor.convergenceMonitor.GenerateConvergenceReport()
	processor.debugManager.LogInfo("Iterative Triclass Final", convergenceReport)

	// Log processing summary
	summary := fmt.Sprintf(`Iterative Triclass Processing Summary:
- Total Processing Time: %v
- Total Iterations: %d
- Total Pixels: %d
- Final Foreground: %d (%.2f%%)
- Final Background: %d (%.2f%%)
- Processing Efficiency: %.2f%%`,
		processingTime,
		processor.iterationState.IterationCount,
		totalPixels,
		foregroundPixels, float64(foregroundPixels)/float64(totalPixels)*100,
		backgroundPixels, float64(backgroundPixels)/float64(totalPixels)*100,
		float64(processor.iterationState.ProcessedPixels)/float64(totalPixels)*100)

	processor.debugManager.LogInfo("Iterative Triclass Summary", summary)
}

// cleanupIterationState releases memory allocated for iteration processing
func (processor *IterativeTriclassProcessor) cleanupIterationState() {
	if processor.iterationState != nil && processor.iterationState.IsInitialized {
		if !processor.iterationState.CurrentRegion.Empty() {
			processor.iterationState.CurrentRegion.Close()
		}
		// Note: FinalResult is returned, so don't close it here
		processor.iterationState.IsInitialized = false
	}
}

// GetConvergenceHistory returns detailed convergence information
func (processor *IterativeTriclassProcessor) GetConvergenceHistory() []ConvergenceRecord {
	return processor.convergenceMonitor.GetConvergenceHistory()
}

// GetProcessingStatistics returns comprehensive processing statistics
func (processor *IterativeTriclassProcessor) GetProcessingStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["algorithm"] = "Iterative_Triclass_Enhanced"
	stats["quality_mode"] = processor.getStringParam("quality", "Fast")
	stats["initial_threshold_method"] = processor.getStringParam("initial_threshold_method", "otsu")
	stats["convergence_epsilon"] = processor.getFloatParam("convergence_epsilon", 1.0)
	stats["max_iterations"] = processor.getIntParam("max_iterations", 10)
	stats["minimum_tbd_fraction"] = processor.getFloatParam("minimum_tbd_fraction", 0.01)
	stats["lower_upper_gap_factor"] = processor.getFloatParam("lower_upper_gap_factor", 0.5)
	stats["apply_preprocessing"] = processor.getBoolParam("apply_preprocessing")
	stats["apply_cleanup"] = processor.getBoolParam("apply_cleanup")
	stats["preserve_borders"] = processor.getBoolParam("preserve_borders")

	if processor.iterationState != nil && processor.iterationState.IsInitialized {
		stats["current_iteration"] = processor.iterationState.IterationCount
		stats["total_pixels"] = processor.iterationState.TotalPixels
		stats["processed_pixels"] = processor.iterationState.ProcessedPixels
		stats["active_pixels"] = processor.iterationState.ActivePixels
	}

	// Add convergence analysis
	convergenceAnalysis := processor.convergenceMonitor.AnalyzeConvergenceBehavior()
	for key, value := range convergenceAnalysis {
		stats["convergence_"+key] = value
	}

	return stats
}

// ValidateParameters checks if all parameters are mathematically valid
func (processor *IterativeTriclassProcessor) ValidateParameters() error {
	return processor.mathProcessor.ValidateIterationParameters()
}

// Parameter access utilities
func (processor *IterativeTriclassProcessor) getIntParam(name string, defaultValue int) int {
	if value, ok := processor.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (processor *IterativeTriclassProcessor) getFloatParam(name string, defaultValue float64) float64 {
	if value, ok := processor.params[name].(float64); ok {
		return value
	}
	return defaultValue
}

func (processor *IterativeTriclassProcessor) getBoolParam(name string) bool {
	if value, ok := processor.params[name].(bool); ok {
		return value
	}
	return false
}

func (processor *IterativeTriclassProcessor) getStringParam(name string, defaultValue string) string {
	if value, ok := processor.params[name].(string); ok {
		return value
	}
	return defaultValue
}

// Cleanup releases resources used by the processor
func (processor *IterativeTriclassProcessor) Cleanup() {
	processor.cleanupIterationState()

	if processor.debugManager != nil {
		processor.debugManager.Cleanup()
	}
}
