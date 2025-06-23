package otsu

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// TwoDOtsuProcessor handles the complete 2D Otsu thresholding workflow
// This replaces the original TwoDOtsuCore with better modularization and mathematical accuracy
type TwoDOtsuProcessor struct {
	params            map[string]interface{}
	debugManager      *debug.Manager
	preprocessor      *TwoDPreprocessor
	histogramBuilder  *TwoDHistogramBuilder
	mathProcessor     *TwoDMathProcessor
}

// NewTwoDOtsuProcessor creates a new 2D Otsu processor with all modular components
func NewTwoDOtsuProcessor(params map[string]interface{}) *TwoDOtsuProcessor {
	debugManager := debug.NewManager()
	
	return &TwoDOtsuProcessor{
		params:           params,
		debugManager:     debugManager,
		preprocessor:     NewTwoDPreprocessor(params),
		histogramBuilder: NewTwoDHistogramBuilder(params),
		mathProcessor:    NewTwoDMathProcessor(params),
	}
}

// Process applies 2D Otsu thresholding with enhanced mathematical accuracy
func (processor *TwoDOtsuProcessor) Process(src gocv.Mat) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	// Validate input dimensions
	if src.Rows() <= 0 || src.Cols() <= 0 {
		return gocv.NewMat(), fmt.Errorf("input Mat has invalid dimensions: %dx%d", src.Cols(), src.Rows())
	}

	processor.debugManager.LogAlgorithmStart("2D Otsu Enhanced", processor.params)
	startTime := time.Now()

	// Step 1: Create working copy to avoid modifying input
	working := src.Clone()
	defer working.Close()

	// Step 2: Convert to grayscale if needed
	gray, err := processor.preprocessor.PrepareGrayscaleImage(&working)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed: %w", err)
	}
	defer gray.Close()

	// Step 3: Apply preprocessing (CLAHE, denoising) if enabled
	preprocessed, err := processor.preprocessor.ApplyPreprocessing(&gray)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("preprocessing failed: %w", err)
	}
	defer preprocessed.Close()

	// Step 4: Calculate neighborhood features
	neighborhood, err := processor.preprocessor.CalculateNeighborhoodFeatures(&preprocessed)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	// Step 5: Build 2D histogram
	histogramData, err := processor.histogramBuilder.BuildHistogram(&preprocessed, &neighborhood)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("histogram construction failed: %w", err)
	}

	// Step 6: Find optimal threshold using mathematical processor
	threshold, err := processor.mathProcessor.FindOptimalThreshold(histogramData)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("threshold calculation failed: %w", err)
	}

	// Step 7: Validate threshold mathematically
	err = processor.mathProcessor.ValidateThreshold(threshold, histogramData.bins)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("threshold validation failed: %w", err)
	}

	// Step 8: Apply binary threshold to create result
	result, err := processor.applyBinaryThreshold(&preprocessed, &neighborhood, threshold, histogramData)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("binary thresholding failed: %w", err)
	}

	// Step 9: Log processing completion with statistics
	totalTime := time.Since(startTime)
	processor.debugManager.LogAlgorithmComplete("2D Otsu Enhanced", totalTime,
		fmt.Sprintf("%dx%d", result.Cols(), result.Rows()))

	// Step 10: Log threshold quality analysis
	processor.logThresholdQuality(histogramData, threshold)

	return result, nil
}

// applyBinaryThreshold creates binary image using calculated 2D threshold
func (processor *TwoDOtsuProcessor) applyBinaryThreshold(src, neighborhood *gocv.Mat, threshold TwoDThreshold, histData *TwoDHistogramData) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu Enhanced", "binary_threshold_application", time.Since(stepTime))

	if src.Empty() || neighborhood.Empty() {
		return gocv.NewMat(), fmt.Errorf("source or neighborhood Mat is empty")
	}

	// Create result Mat
	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)

	// Get parameters
	pixelWeight := processor.getFloatParam("pixel_weight_factor", 0.5)
	
	// Convert threshold indices back to intensity values
	pixelThresholdVal := float64(threshold.PixelThreshold) * 255.0 / float64(histData.bins-1)
	featureThresholdVal := float64(threshold.FeatureThreshold) * 255.0 / float64(histData.bins-1)

	// Apply 2D thresholding with mathematical accuracy
	rows := src.Rows()
	cols := src.Cols()
	
	foregroundCount := 0
	backgroundCount := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelVal := float64(src.GetUCharAt(y, x))
			neighVal := float64(neighborhood.GetUCharAt(y, x))

			// Calculate blended feature using same formula as histogram building
			blendedFeature := pixelWeight*pixelVal + (1.0-pixelWeight)*neighVal

			// Apply 2D threshold criterion - both conditions must be met for foreground
			if pixelVal > pixelThresholdVal && blendedFeature > featureThresholdVal {
				result.SetUCharAt(y, x, 255) // Foreground
				foregroundCount++
			} else {
				result.SetUCharAt(y, x, 0) // Background
				backgroundCount++
			}
		}
	}

	// Log binary threshold statistics
	totalPixels := foregroundCount + backgroundCount
	foregroundRatio := float64(foregroundCount) / float64(totalPixels)
	
	processor.debugManager.LogInfo("2D Otsu Enhanced", 
		fmt.Sprintf("Binary threshold results: FG=%d (%.2f%%), BG=%d (%.2f%%)", 
			foregroundCount, foregroundRatio*100, backgroundCount, (1.0-foregroundRatio)*100))

	return result, nil
}

// logThresholdQuality performs quality analysis of the calculated threshold
func (processor *TwoDOtsuProcessor) logThresholdQuality(histData *TwoDHistogramData, threshold TwoDThreshold) {
	// Calculate global statistics for quality assessment
	globalStats := &GlobalStatistics2D{}
	bins := histData.bins

	// Calculate global histogram statistics
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := histData.histogram[i][j]
			globalStats.TotalWeight += weight
			globalStats.WeightedSumI += weight * float64(i)
			globalStats.WeightedSumJ += weight * float64(j)
		}
	}

	if globalStats.TotalWeight > 0 {
		globalStats.MeanI = globalStats.WeightedSumI / globalStats.TotalWeight
		globalStats.MeanJ = globalStats.WeightedSumJ / globalStats.TotalWeight
	}

	// Calculate threshold quality metrics
	quality, err := processor.mathProcessor.CalculateThresholdQuality(histData, threshold, globalStats)
	if err != nil {
		processor.debugManager.LogWarning("2D Otsu Enhanced", 
			fmt.Sprintf("Quality analysis failed: %v", err))
		return
	}

	// Log quality metrics
	qualityReport := fmt.Sprintf(`2D Otsu Threshold Quality Analysis:
- Between-class variance: %.6f
- Within-class variance: %.6f  
- Separability ratio: %.6f
- Class balance entropy: %.6f
- Threshold coordinates: (%d, %d)
- Threshold intensities: (%.1f, %.1f)`,
		quality["between_class_variance"],
		quality["within_class_variance"],
		quality["separability"],
		quality["class_balance"],
		threshold.PixelThreshold,
		threshold.FeatureThreshold,
		float64(threshold.PixelThreshold)*255.0/float64(bins-1),
		float64(threshold.FeatureThreshold)*255.0/float64(bins-1))

	processor.debugManager.LogInfo("2D Otsu Quality", qualityReport)
}

// ValidateParameters checks if all parameters are mathematically valid
func (processor *TwoDOtsuProcessor) ValidateParameters() error {
	// Validate preprocessing parameters
	if err := processor.preprocessor.ValidatePreprocessingParameters(); err != nil {
		return fmt.Errorf("preprocessing parameter validation failed: %w", err)
	}

	// Validate histogram parameters
	histBins := processor.getIntParam("histogram_bins", 64)
	if histBins < 8 || histBins > 512 {
		return fmt.Errorf("histogram_bins must be between 8 and 512, got %d", histBins)
	}

	pixelWeight := processor.getFloatParam("pixel_weight_factor", 0.5)
	if pixelWeight < 0.0 || pixelWeight > 1.0 {
		return fmt.Errorf("pixel_weight_factor must be between 0.0 and 1.0, got %.3f", pixelWeight)
	}

	smoothingSigma := processor.getFloatParam("smoothing_sigma", 0.0)
	if smoothingSigma < 0.0 || smoothingSigma > 10.0 {
		return fmt.Errorf("smoothing_sigma must be between 0.0 and 10.0, got %.3f", smoothingSigma)
	}

	return nil
}

// GetProcessingStatistics returns comprehensive processing statistics
func (processor *TwoDOtsuProcessor) GetProcessingStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["algorithm"] = "2D_Otsu_Enhanced"
	stats["quality_mode"] = processor.getStringParam("quality", "Fast")
	stats["window_size"] = processor.getIntParam("window_size", 7)
	stats["histogram_bins"] = processor.getIntParam("histogram_bins", 64)
	stats["neighbourhood_metric"] = processor.getStringParam("neighbourhood_metric", "mean")
	stats["pixel_weight_factor"] = processor.getFloatParam("pixel_weight_factor", 0.5)
	stats["apply_contrast_enhancement"] = processor.getBoolParam("apply_contrast_enhancement")
	stats["use_log_histogram"] = processor.getBoolParam("use_log_histogram")
	stats["normalize_histogram"] = processor.getBoolParam("normalize_histogram")
	stats["smoothing_sigma"] = processor.getFloatParam("smoothing_sigma", 0.0)

	return stats
}

// GetHistogramStatistics returns detailed histogram analysis
func (processor *TwoDOtsuProcessor) GetHistogramStatistics(histData *TwoDHistogramData) map[string]interface{} {
	if histData == nil {
		return make(map[string]interface{})
	}
	
	return processor.histogramBuilder.GetHistogramStatistics(histData)
}

// Parameter access utilities with type safety
func (processor *TwoDOtsuProcessor) getIntParam(name string, defaultValue int) int {
	if value, ok := processor.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (processor *TwoDOtsuProcessor) getFloatParam(name string, defaultValue float64) float64 {
	if value, ok := processor.params[name].(float64); ok {
		return value
	}
	return defaultValue
}

func (processor *TwoDOtsuProcessor) getBoolParam(name string) bool {
	if value, ok := processor.params[name].(bool); ok {
		return value
	}
	return false
}

func (processor *TwoDOtsuProcessor) getStringParam(name string, defaultValue string) string {
	if value, ok := processor.params[name].(string); ok {
		return value
	}
	return defaultValue
}

// Cleanup releases resources used by the processor
func (processor *TwoDOtsuProcessor) Cleanup() {
	if processor.debugManager != nil {
		processor.debugManager.Cleanup()
	}
} 