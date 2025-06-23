package otsu

import (
	"fmt"
	"image"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// TwoDPreprocessor handles image preprocessing for 2D Otsu thresholding
type TwoDPreprocessor struct {
	params       map[string]interface{}
	debugManager *debug.Manager
}

// NewTwoDPreprocessor creates a new preprocessing handler
func NewTwoDPreprocessor(params map[string]interface{}) *TwoDPreprocessor {
	return &TwoDPreprocessor{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

// PrepareGrayscaleImage converts input to grayscale using latest GoCV APIs
func (processor *TwoDPreprocessor) PrepareGrayscaleImage(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "grayscale_conversion", time.Since(stepTime))

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

	processor.debugManager.LogInfo("2D Otsu Preprocessing",
		fmt.Sprintf("Converted %d-channel image to grayscale", channels))

	return gray, nil
}

// ApplyPreprocessing applies CLAHE and denoising if enabled
func (processor *TwoDPreprocessor) ApplyPreprocessing(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "preprocessing", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	result := gocv.NewMat()

	if processor.getBoolParam("apply_contrast_enhancement") {
		enhanced, err := processor.applyCLAHE(src)
		if err != nil {
			return gocv.NewMat(), err
		}
		defer enhanced.Close()

		// Apply denoising if Best quality mode is selected
		if processor.getStringParam("quality", "Fast") == "Best" {
			denoised, err := processor.applyDenoising(&enhanced)
			if err != nil {
				return gocv.NewMat(), err
			}
			defer denoised.Close()
			denoised.CopyTo(&result)
		} else {
			enhanced.CopyTo(&result)
		}
	} else {
		src.CopyTo(&result)
	}

	return result, nil
}

// applyCLAHE applies Contrast Limited Adaptive Histogram Equalization
func (processor *TwoDPreprocessor) applyCLAHE(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "clahe_enhancement", time.Since(stepTime))

	// Create CLAHE with latest GoCV API parameters
	clipLimit := 2.0
	tileGridSize := image.Point{X: 8, Y: 8}

	clahe := gocv.NewCLAHEWithParams(clipLimit, tileGridSize)
	defer clahe.Close()

	result := gocv.NewMat()
	clahe.Apply(*src, &result)

	processor.debugManager.LogInfo("2D Otsu Preprocessing",
		fmt.Sprintf("Applied CLAHE with clip limit %.1f and tile size %dx%d",
			clipLimit, tileGridSize.X, tileGridSize.Y))

	return result, nil
}

// applyDenoising applies advanced denoising using latest GoCV APIs
func (processor *TwoDPreprocessor) applyDenoising(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "denoising", time.Since(stepTime))

	result := gocv.NewMat()

	// Use FastNlMeansDenoising with optimized parameters for latest GoCV
	h := float32(3.0)       // Filter strength
	templateWindowSize := 7 // Template patch size
	searchWindowSize := 21  // Search area size

	gocv.FastNlMeansDenoisingWithParams(*src, &result, h, templateWindowSize, searchWindowSize)

	processor.debugManager.LogInfo("2D Otsu Preprocessing",
		fmt.Sprintf("Applied denoising with h=%.1f, template=%d, search=%d",
			h, templateWindowSize, searchWindowSize))

	return result, nil
}

// CalculateNeighborhoodFeatures computes neighborhood-based features using latest GoCV APIs
func (processor *TwoDPreprocessor) CalculateNeighborhoodFeatures(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "neighborhood_calculation", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	windowSize := processor.getIntParam("window_size", 7)

	// Ensure odd window size for symmetric operations
	if windowSize%2 == 0 {
		windowSize++
	}

	// Clamp window size to reasonable bounds
	if windowSize < 3 {
		windowSize = 3
	} else if windowSize > 31 {
		windowSize = 31
	}

	metric := processor.getStringParam("neighbourhood_metric", "mean")

	switch metric {
	case "mean":
		return processor.calculateMeanNeighborhood(src, windowSize)
	case "median":
		return processor.calculateMedianNeighborhood(src, windowSize)
	case "gaussian":
		return processor.calculateGaussianNeighborhood(src, windowSize)
	default:
		// Default to mean if unknown metric
		return processor.calculateMeanNeighborhood(src, windowSize)
	}
}

// calculateMeanNeighborhood computes mean-based neighborhood features
func (processor *TwoDPreprocessor) calculateMeanNeighborhood(src *gocv.Mat, windowSize int) (gocv.Mat, error) {
	result := gocv.NewMat()

	// Use BoxFilter for mean calculation with latest GoCV API
	gocv.BoxFilter(*src, &result, -1, image.Point{X: windowSize, Y: windowSize})

	processor.debugManager.LogInfo("2D Otsu Preprocessing",
		fmt.Sprintf("Calculated mean neighborhood with window size %d", windowSize))

	return result, nil
}

// calculateMedianNeighborhood computes median-based neighborhood features
func (processor *TwoDPreprocessor) calculateMedianNeighborhood(src *gocv.Mat, windowSize int) (gocv.Mat, error) {
	result := gocv.NewMat()

	// Median filtering with bounds checking for GoCV limitations
	if windowSize > 31 {
		windowSize = 31 // OpenCV limitation for median filter
	}

	gocv.MedianBlur(*src, &result, windowSize)

	processor.debugManager.LogInfo("2D Otsu Preprocessing",
		fmt.Sprintf("Calculated median neighborhood with window size %d", windowSize))

	return result, nil
}

// calculateGaussianNeighborhood computes Gaussian-weighted neighborhood features
func (processor *TwoDPreprocessor) calculateGaussianNeighborhood(src *gocv.Mat, windowSize int) (gocv.Mat, error) {
	result := gocv.NewMat()

	// Calculate sigma based on window size using standard formula
	sigma := float64(windowSize) / 6.0

	// Apply Gaussian blur with latest GoCV API
	ksize := image.Point{X: windowSize, Y: windowSize}

	gocv.GaussianBlur(*src, &result, ksize, sigma, sigma, gocv.BorderDefault)

	processor.debugManager.LogInfo("2D Otsu Preprocessing",
		fmt.Sprintf("Calculated Gaussian neighborhood with window size %d, sigma %.2f",
			windowSize, sigma))

	return result, nil
}

// ValidatePreprocessingParameters checks if preprocessing parameters are valid
func (processor *TwoDPreprocessor) ValidatePreprocessingParameters() error {
	windowSize := processor.getIntParam("window_size", 7)

	if windowSize < 3 || windowSize > 31 {
		return fmt.Errorf("window_size must be between 3 and 31, got %d", windowSize)
	}

	if windowSize%2 == 0 {
		return fmt.Errorf("window_size must be odd, got %d", windowSize)
	}

	metric := processor.getStringParam("neighbourhood_metric", "mean")
	validMetrics := []string{"mean", "median", "gaussian"}

	isValid := false
	for _, valid := range validMetrics {
		if metric == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("neighbourhood_metric must be one of %v, got %s", validMetrics, metric)
	}

	return nil
}

// GetPreprocessingStatistics returns preprocessing statistics
func (processor *TwoDPreprocessor) GetPreprocessingStatistics(original, processed *gocv.Mat) map[string]interface{} {
	stats := make(map[string]interface{})

	stats["contrast_enhancement"] = processor.getBoolParam("apply_contrast_enhancement")
	stats["quality_mode"] = processor.getStringParam("quality", "Fast")
	stats["window_size"] = processor.getIntParam("window_size", 7)
	stats["neighbourhood_metric"] = processor.getStringParam("neighbourhood_metric", "mean")

	// Calculate basic image statistics
	if !original.Empty() && !processed.Empty() {
		// Use simple mean calculation
		origSum := 0.0
		procSum := 0.0
		totalPixels := original.Rows() * original.Cols()

		for y := 0; y < original.Rows(); y++ {
			for x := 0; x < original.Cols(); x++ {
				origSum += float64(original.GetUCharAt(y, x))
				procSum += float64(processed.GetUCharAt(y, x))
			}
		}

		origMean := origSum / float64(totalPixels)
		procMean := procSum / float64(totalPixels)

		stats["original_mean"] = origMean
		stats["processed_mean"] = procMean
		stats["mean_change"] = procMean - origMean
	}

	return stats
}

// Parameter access utilities
func (processor *TwoDPreprocessor) getIntParam(name string, defaultValue int) int {
	if value, ok := processor.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (processor *TwoDPreprocessor) getStringParam(name string, defaultValue string) string {
	if value, ok := processor.params[name].(string); ok {
		return value
	}
	return defaultValue
}

func (processor *TwoDPreprocessor) getBoolParam(name string) bool {
	if value, ok := processor.params[name].(bool); ok {
		return value
	}
	return false
}
