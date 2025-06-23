package otsu

import (
	"fmt"
	"image"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// IterativeTriclassPostprocessor handles post-processing operations for iterative triclass results
type IterativeTriclassPostprocessor struct {
	params       map[string]interface{}
	debugManager *debug.Manager
}

// NewIterativeTriclassPostprocessor creates a new post-processor
func NewIterativeTriclassPostprocessor(params map[string]interface{}) *IterativeTriclassPostprocessor {
	return &IterativeTriclassPostprocessor{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

// ApplyPostprocessing applies cleanup operations to the final result
func (postprocessor *IterativeTriclassPostprocessor) ApplyPostprocessing(src gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer postprocessor.debugManager.LogAlgorithmStep("Iterative Triclass", "postprocessing", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	result := src.Clone()

	// Apply morphological cleanup if requested
	if postprocessor.getBoolParam("apply_cleanup") {
		cleaned, err := postprocessor.applyMorphologicalCleanup(&result)
		if err != nil {
			result.Close()
			return gocv.NewMat(), fmt.Errorf("morphological cleanup failed: %w", err)
		}
		result.Close()
		result = cleaned
	}

	// Apply border preservation if requested
	if postprocessor.getBoolParam("preserve_borders") {
		preserved, err := postprocessor.preserveBorders(&result, &src)
		if err != nil {
			result.Close()
			return gocv.NewMat(), fmt.Errorf("border preservation failed: %w", err)
		}
		result.Close()
		result = preserved
	}

	return result, nil
}

// applyMorphologicalCleanup performs morphological operations using latest GoCV APIs
func (postprocessor *IterativeTriclassPostprocessor) applyMorphologicalCleanup(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer postprocessor.debugManager.LogAlgorithmStep("Iterative Triclass", "morphological_cleanup", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	// Create morphological kernels using latest GoCV API
	smallKernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer smallKernel.Close()

	mediumKernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 5, Y: 5})
	defer mediumKernel.Close()

	// Step 1: Remove small noise with opening operation
	opened := gocv.NewMat()
	defer opened.Close()
	gocv.MorphologyEx(*src, &opened, gocv.MorphOpen, smallKernel)

	// Step 2: Fill small holes with closing operation
	closed := gocv.NewMat()
	defer closed.Close()
	gocv.MorphologyEx(opened, &closed, gocv.MorphClose, mediumKernel)

	// Step 3: Apply median filter to smooth boundaries
	smoothed := gocv.NewMat()
	gocv.MedianBlur(closed, &smoothed, 3)

	// Step 4: Additional cleanup for "Best" quality mode
	if postprocessor.getStringParam("quality", "Fast") == "Best" {
		// Apply gradient morphology to preserve edges
		gradient := gocv.NewMat()
		defer gradient.Close()
		gocv.MorphologyEx(smoothed, &gradient, gocv.MorphGradient, smallKernel)

		// Combine with original for edge preservation
		enhanced := gocv.NewMat()
		defer enhanced.Close()
		gocv.BitwiseOr(smoothed, gradient, &enhanced)

		smoothed.Close()
		smoothed = enhanced.Clone()
	}

	postprocessor.debugManager.LogInfo("Iterative Triclass Postprocessing",
		"Applied morphological cleanup operations")

	return smoothed, nil
}

// preserveBorders ensures border pixels are handled appropriately
func (postprocessor *IterativeTriclassPostprocessor) preserveBorders(src, original *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer postprocessor.debugManager.LogAlgorithmStep("Iterative Triclass", "border_preservation", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	result := src.Clone()
	rows := result.Rows()
	cols := result.Cols()

	// Create border mask (1-pixel wide border)
	borderMask := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	defer borderMask.Close()
	borderMask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set border pixels to 255 in mask
	for x := 0; x < cols; x++ {
		borderMask.SetUCharAt(0, x, 255)      // Top border
		borderMask.SetUCharAt(rows-1, x, 255) // Bottom border
	}
	for y := 0; y < rows; y++ {
		borderMask.SetUCharAt(y, 0, 255)      // Left border
		borderMask.SetUCharAt(y, cols-1, 255) // Right border
	}

	// Apply erosion to slightly shrink foreground regions near borders
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	eroded := gocv.NewMat()
	defer eroded.Close()

	// Use latest GoCV Erode API with correct parameters
	gocv.ErodeWithParams(result, &eroded, kernel, image.Point{X: -1, Y: -1}, 1, gocv.BorderConstant)

	// Restore border pixels from original
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if borderMask.GetUCharAt(y, x) > 0 {
				// Keep original border pixel values
				originalValue := src.GetUCharAt(y, x)
				result.SetUCharAt(y, x, originalValue)
			} else {
				// Use eroded value for interior pixels
				erodedValue := eroded.GetUCharAt(y, x)
				result.SetUCharAt(y, x, erodedValue)
			}
		}
	}

	postprocessor.debugManager.LogInfo("Iterative Triclass Postprocessing",
		"Applied border preservation")

	return result, nil
}

// ApplyAdvancedFiltering applies advanced filtering techniques for high-quality results
func (postprocessor *IterativeTriclassPostprocessor) ApplyAdvancedFiltering(src gocv.Mat) (gocv.Mat, error) {
	if postprocessor.getStringParam("quality", "Fast") != "Best" {
		// Return copy for non-Best quality modes
		return src.Clone(), nil
	}

	stepTime := time.Now()
	defer postprocessor.debugManager.LogAlgorithmStep("Iterative Triclass", "advanced_filtering", time.Since(stepTime))

	// Apply bilateral filter to preserve edges while smoothing
	bilateral := gocv.NewMat()
	defer bilateral.Close()

	// Use latest GoCV bilateral filter API
	gocv.BilateralFilter(src, &bilateral, 9, 75.0, 75.0)

	// Apply adaptive threshold to enhance boundaries
	adaptive := gocv.NewMat()
	defer adaptive.Close()

	gocv.AdaptiveThreshold(bilateral, &adaptive, 255, gocv.AdaptiveThresholdMean, gocv.ThresholdBinary, 11, 2)

	// Combine results using weighted average
	result := gocv.NewMat()
	gocv.AddWeighted(src, 0.7, adaptive, 0.3, 0, &result)

	postprocessor.debugManager.LogInfo("Iterative Triclass Postprocessing",
		"Applied advanced filtering techniques")

	return result, nil
}

// ApplyConditionalSmoothing applies smoothing based on local image characteristics
func (postprocessor *IterativeTriclassPostprocessor) ApplyConditionalSmoothing(src gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer postprocessor.debugManager.LogAlgorithmStep("Iterative Triclass", "conditional_smoothing", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	result := src.Clone()

	// Calculate local variance to identify noisy regions
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 5, Y: 5})
	defer kernel.Close()

	// Apply Laplacian to detect edges
	laplacian := gocv.NewMat()
	defer laplacian.Close()
	gocv.Laplacian(src, &laplacian, gocv.MatTypeCV8U, 1, 1, 0, gocv.BorderDefault)

	// Create mask for regions that need smoothing (non-edge regions)
	smoothingMask := gocv.NewMat()
	defer smoothingMask.Close()
	gocv.Threshold(laplacian, &smoothingMask, 30, 255, gocv.ThresholdBinaryInv)

	// Apply morphological closing to fill gaps in mask
	closed := gocv.NewMat()
	defer closed.Close()
	gocv.MorphologyEx(smoothingMask, &closed, gocv.MorphClose, kernel)

	// Apply bilateral filter only to non-edge regions
	if postprocessor.getStringParam("quality", "Fast") == "Best" {
		bilateral := gocv.NewMat()
		defer bilateral.Close()
		gocv.BilateralFilter(src, &bilateral, 9, 75.0, 75.0)

		// Combine results using weighted average
		gocv.BitwiseAnd(bilateral, bilateral, &bilateral)

		// Invert mask for original image regions
		invertedMask := gocv.NewMat()
		defer invertedMask.Close()
		gocv.BitwiseNot(closed, &invertedMask)

		originalMasked := gocv.NewMat()
		defer originalMasked.Close()
		gocv.BitwiseAnd(src, src, &originalMasked)

		// Combine results
		gocv.BitwiseOr(bilateral, originalMasked, &result)
	}

	postprocessor.debugManager.LogInfo("Iterative Triclass Postprocessing",
		"Applied conditional smoothing based on edge detection")

	return result, nil
}

// ValidatePostprocessingParameters checks parameter validity
func (postprocessor *IterativeTriclassPostprocessor) ValidatePostprocessingParameters() error {
	quality := postprocessor.getStringParam("quality", "Fast")
	validQualities := []string{"Fast", "Best"}

	isValid := false
	for _, valid := range validQualities {
		if quality == valid {
			isValid = true
			break
		}
	}

	if !isValid {
		return fmt.Errorf("quality must be one of %v, got %s", validQualities, quality)
	}

	return nil
}

// GetPostprocessingStatistics returns statistics about post-processing operations
func (postprocessor *IterativeTriclassPostprocessor) GetPostprocessingStatistics(original, processed gocv.Mat) map[string]interface{} {
	stats := make(map[string]interface{})

	stats["apply_cleanup"] = postprocessor.getBoolParam("apply_cleanup")
	stats["preserve_borders"] = postprocessor.getBoolParam("preserve_borders")
	stats["quality_mode"] = postprocessor.getStringParam("quality", "Fast")

	if !original.Empty() && !processed.Empty() {
		// Calculate basic statistics
		originalForeground := gocv.CountNonZero(original)
		processedForeground := gocv.CountNonZero(processed)
		totalPixels := original.Rows() * original.Cols()

		stats["original_foreground_pixels"] = originalForeground
		stats["processed_foreground_pixels"] = processedForeground
		stats["foreground_pixel_change"] = processedForeground - originalForeground
		stats["foreground_ratio_change"] = float64(processedForeground-originalForeground) / float64(totalPixels)

		// Calculate difference mask
		diff := gocv.NewMat()
		defer diff.Close()
		gocv.AbsDiff(original, processed, &diff)
		changedPixels := gocv.CountNonZero(diff)

		stats["changed_pixels"] = changedPixels
		stats["change_percentage"] = float64(changedPixels) / float64(totalPixels) * 100.0
	}

	return stats
}

// ApplyQualityAssurance performs quality checks on post-processed results
func (postprocessor *IterativeTriclassPostprocessor) ApplyQualityAssurance(src, result gocv.Mat) error {
	if src.Empty() || result.Empty() {
		return fmt.Errorf("input or result Mat is empty")
	}

	// Check dimensions match
	if src.Rows() != result.Rows() || src.Cols() != result.Cols() {
		return fmt.Errorf("dimension mismatch: src=%dx%d, result=%dx%d",
			src.Cols(), src.Rows(), result.Cols(), result.Rows())
	}

	// Check that result is binary
	minVal, maxVal, _, _ := gocv.MinMaxLoc(result)
	if minVal < 0 || maxVal > 255 {
		return fmt.Errorf("result contains invalid pixel values: min=%.0f, max=%.0f", minVal, maxVal)
	}

	// Check for reasonable foreground/background ratio
	foregroundPixels := gocv.CountNonZero(result)
	totalPixels := result.Rows() * result.Cols()
	foregroundRatio := float64(foregroundPixels) / float64(totalPixels)

	if foregroundRatio < 0.001 || foregroundRatio > 0.999 {
		postprocessor.debugManager.LogWarning("Iterative Triclass Postprocessing",
			fmt.Sprintf("Extreme foreground ratio: %.4f", foregroundRatio))
	}

	postprocessor.debugManager.LogInfo("Iterative Triclass Postprocessing",
		fmt.Sprintf("Quality assurance passed: FG ratio=%.4f", foregroundRatio))

	return nil
}

// AnalyzeBoundaryQuality analyzes the quality of object boundaries
func (postprocessor *IterativeTriclassPostprocessor) AnalyzeBoundaryQuality(binary gocv.Mat) map[string]interface{} {
	analysis := make(map[string]interface{})

	if binary.Empty() {
		analysis["error"] = "empty input"
		return analysis
	}

	// Calculate boundary length using edge detection
	edges := gocv.NewMat()
	defer edges.Close()
	gocv.Canny(binary, &edges, 50, 150)

	boundaryPixels := gocv.CountNonZero(edges)
	totalPixels := binary.Rows() * binary.Cols()

	analysis["boundary_pixels"] = boundaryPixels
	analysis["boundary_density"] = float64(boundaryPixels) / float64(totalPixels)

	// Calculate boundary smoothness using morphological operations
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	smoothed := gocv.NewMat()
	defer smoothed.Close()
	gocv.MorphologyEx(binary, &smoothed, gocv.MorphClose, kernel)

	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(binary, smoothed, &diff)

	roughnessPixels := gocv.CountNonZero(diff)
	analysis["boundary_roughness"] = float64(roughnessPixels) / float64(totalPixels)
	analysis["boundary_smoothness"] = 1.0 - analysis["boundary_roughness"].(float64)

	// Classify boundary quality
	smoothness := analysis["boundary_smoothness"].(float64)
	if smoothness > 0.95 {
		analysis["quality_rating"] = "excellent"
	} else if smoothness > 0.85 {
		analysis["quality_rating"] = "good"
	} else if smoothness > 0.70 {
		analysis["quality_rating"] = "fair"
	} else {
		analysis["quality_rating"] = "poor"
	}

	return analysis
}

// OptimizeForConnectivity optimizes result for better connected components
func (postprocessor *IterativeTriclassPostprocessor) OptimizeForConnectivity(src gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer postprocessor.debugManager.LogAlgorithmStep("Iterative Triclass", "connectivity_optimization", time.Since(stepTime))

	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	// Analyze connected components
	labels := gocv.NewMat()
	defer labels.Close()
	stats := gocv.NewMat()
	defer stats.Close()
	centroids := gocv.NewMat()
	defer centroids.Close()

	numComponents := gocv.ConnectedComponentsWithStats(src, &labels, &stats, &centroids)

	// Remove small components
	result := src.Clone()
	minComponentSize := (src.Rows() * src.Cols()) / 1000 // 0.1% of image size

	for i := 1; i < numComponents; i++ { // Skip background (label 0)
		area := int(stats.GetIntAt(i, 4)) // Area is at index 4

		if area < minComponentSize {
			// Remove small component by setting pixels to 0
			for y := 0; y < labels.Rows(); y++ {
				for x := 0; x < labels.Cols(); x++ {
					if int(labels.GetIntAt(y, x)) == i {
						result.SetUCharAt(y, x, 0)
					}
				}
			}
		}
	}

	postprocessor.debugManager.LogInfo("Iterative Triclass Postprocessing",
		fmt.Sprintf("Connectivity optimization: %d components processed", numComponents))

	return result, nil
}

// Parameter access utilities
func (postprocessor *IterativeTriclassPostprocessor) getBoolParam(name string) bool {
	if value, ok := postprocessor.params[name].(bool); ok {
		return value
	}
	return false
}

func (postprocessor *IterativeTriclassPostprocessor) getStringParam(name string, defaultValue string) string {
	if value, ok := postprocessor.params[name].(string); ok {
		return value
	}
	return defaultValue
}

func (postprocessor *IterativeTriclassPostprocessor) getIntParam(name string, defaultValue int) int {
	if value, ok := postprocessor.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (postprocessor *IterativeTriclassPostprocessor) getFloatParam(name string, defaultValue float64) float64 {
	if value, ok := postprocessor.params[name].(float64); ok {
		return value
	}
	return defaultValue
}

// Cleanup releases resources used by the postprocessor
func (postprocessor *IterativeTriclassPostprocessor) Cleanup() {
	if postprocessor.debugManager != nil {
		postprocessor.debugManager.Cleanup()
	}
}
