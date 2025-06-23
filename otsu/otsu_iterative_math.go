package otsu

import (
	"fmt"
	"math"
	"time"

	"otsu-obliterator/debug"
)

// IterativeTriclassMathProcessor handles mathematical computations for iterative triclass thresholding
type IterativeTriclassMathProcessor struct {
	params       map[string]interface{}
	debugManager *debug.Manager
}

// NewIterativeTriclassMath creates a new mathematical processor for triclass algorithm
func NewIterativeTriclassMath(params map[string]interface{}) *IterativeTriclassMathProcessor {
	return &IterativeTriclassMathProcessor{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

// CalculateRegionThreshold computes threshold for active region using specified method
func (mathProc *IterativeTriclassMathProcessor) CalculateRegionThreshold(histogram []int) (float64, error) {
	stepTime := time.Now()
	defer mathProc.debugManager.LogAlgorithmStep("Iterative Triclass", "threshold_calculation", time.Since(stepTime))

	if len(histogram) == 0 {
		return 0, fmt.Errorf("histogram is empty")
	}

	method := mathProc.getStringParam("initial_threshold_method", "otsu")

	switch method {
	case "otsu":
		return mathProc.calculateOtsuThreshold(histogram)
	case "mean":
		return mathProc.calculateMeanThreshold(histogram)
	case "median":
		return mathProc.calculateMedianThreshold(histogram)
	default:
		return mathProc.calculateOtsuThreshold(histogram)
	}
}

// calculateOtsuThreshold implements classic 1D Otsu thresholding
func (mathProc *IterativeTriclassMathProcessor) calculateOtsuThreshold(histogram []int) (float64, error) {
	histBins := len(histogram)

	// Calculate total pixel count
	total := 0
	for _, count := range histogram {
		total += count
	}

	if total == 0 {
		return 127.5, fmt.Errorf("histogram contains no pixels")
	}

	// Calculate weighted sum for mean calculation
	sum := 0.0
	for i, count := range histogram {
		sum += float64(i) * float64(count)
	}

	// Variables for Otsu's method
	sumB := 0.0 // Sum of background
	wB := 0     // Weight of background
	maxVariance := 0.0
	bestThreshold := 127.5

	// Iterate through all possible thresholds
	for t := 0; t < histBins; t++ {
		wB += histogram[t]
		if wB == 0 {
			continue
		}

		wF := total - wB // Weight of foreground
		if wF == 0 {
			break
		}

		sumB += float64(t) * float64(histogram[t])

		// Calculate means
		mB := sumB / float64(wB)         // Mean of background
		mF := (sum - sumB) / float64(wF) // Mean of foreground

		// Calculate between-class variance
		variance := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if variance > maxVariance {
			maxVariance = variance
			bestThreshold = float64(t) * 255.0 / float64(histBins-1)
		}
	}

	mathProc.debugManager.LogThresholdCalculation("Iterative Triclass Otsu", bestThreshold,
		fmt.Sprintf("variance=%.6f", maxVariance))

	return bestThreshold, nil
}

// calculateMeanThreshold computes mean-based threshold
func (mathProc *IterativeTriclassMathProcessor) calculateMeanThreshold(histogram []int) (float64, error) {
	histBins := len(histogram)
	totalPixels := 0
	weightedSum := 0.0

	for i, count := range histogram {
		totalPixels += count
		weightedSum += float64(i) * float64(count)
	}

	if totalPixels == 0 {
		return 127.5, fmt.Errorf("histogram contains no pixels")
	}

	meanBin := weightedSum / float64(totalPixels)
	threshold := meanBin * 255.0 / float64(histBins-1)

	mathProc.debugManager.LogThresholdCalculation("Iterative Triclass Mean", threshold, "mean_based")

	return threshold, nil
}

// calculateMedianThreshold computes median-based threshold
func (mathProc *IterativeTriclassMathProcessor) calculateMedianThreshold(histogram []int) (float64, error) {
	histBins := len(histogram)
	totalPixels := 0

	for _, count := range histogram {
		totalPixels += count
	}

	if totalPixels == 0 {
		return 127.5, fmt.Errorf("histogram contains no pixels")
	}

	halfPixels := totalPixels / 2
	cumSum := 0

	for i, count := range histogram {
		cumSum += count
		if cumSum >= halfPixels {
			threshold := float64(i) * 255.0 / float64(histBins-1)
			mathProc.debugManager.LogThresholdCalculation("Iterative Triclass Median", threshold, "median_based")
			return threshold, nil
		}
	}

	return 127.5, fmt.Errorf("median calculation failed")
}

// CalculateTriclassThresholds computes lower and upper thresholds for triclass segmentation
func (mathProc *IterativeTriclassMathProcessor) CalculateTriclassThresholds(baseThreshold float64) (float64, float64, error) {
	gapFactor := mathProc.getFloatParam("lower_upper_gap_factor", 0.5)

	if gapFactor < 0.0 || gapFactor > 1.0 {
		return 0, 0, fmt.Errorf("gap factor must be between 0.0 and 1.0, got %.3f", gapFactor)
	}

	// Calculate adaptive thresholds
	lowerThreshold := baseThreshold * (1.0 - gapFactor)
	upperThreshold := baseThreshold * (1.0 + gapFactor)

	// Ensure thresholds are within valid intensity range [0, 255]
	lowerThreshold = math.Max(0.0, math.Min(255.0, lowerThreshold))
	upperThreshold = math.Max(0.0, math.Min(255.0, upperThreshold))

	// Ensure upper > lower threshold
	if upperThreshold <= lowerThreshold {
		upperThreshold = lowerThreshold + 1.0
		if upperThreshold > 255.0 {
			upperThreshold = 255.0
			lowerThreshold = 254.0
		}
	}

	mathProc.debugManager.LogThresholdCalculation("Iterative Triclass Bounds",
		fmt.Sprintf("lower=%.2f, upper=%.2f", lowerThreshold, upperThreshold),
		fmt.Sprintf("gap_factor=%.3f", gapFactor))

	return lowerThreshold, upperThreshold, nil
}

// CheckConvergence determines if the iterative process has converged
func (mathProc *IterativeTriclassMathProcessor) CheckConvergence(currentThreshold, previousThreshold float64, iteration int) (bool, float64, error) {
	if iteration == 0 {
		return false, math.Inf(1), nil // First iteration, no convergence yet
	}

	convergenceEpsilon := mathProc.getFloatParam("convergence_epsilon", 1.0)
	maxIterations := mathProc.getIntParam("max_iterations", 10)

	convergenceValue := math.Abs(currentThreshold - previousThreshold)

	// Check convergence criteria
	converged := convergenceValue < convergenceEpsilon

	// Force convergence if maximum iterations reached
	if iteration >= maxIterations {
		converged = true
		mathProc.debugManager.LogInfo("Iterative Triclass Math",
			fmt.Sprintf("Forced convergence at iteration %d (max: %d)", iteration, maxIterations))
	}

	mathProc.debugManager.LogConvergenceInfo("Iterative Triclass", iteration, currentThreshold, convergenceValue, converged)

	return converged, convergenceValue, nil
}

// CheckTBDTermination determines if TBD region is small enough to terminate
func (mathProc *IterativeTriclassMathProcessor) CheckTBDTermination(tbdPixelCount, totalPixels int) (bool, float64) {
	if totalPixels == 0 {
		return true, 0.0
	}

	tbdFraction := float64(tbdPixelCount) / float64(totalPixels)
	minTBDFraction := mathProc.getFloatParam("minimum_tbd_fraction", 0.01)

	shouldTerminate := tbdFraction < minTBDFraction

	if shouldTerminate {
		mathProc.debugManager.LogInfo("Iterative Triclass Math",
			fmt.Sprintf("TBD termination: fraction %.6f < threshold %.6f", tbdFraction, minTBDFraction))
	}

	return shouldTerminate, tbdFraction
}

// CalculateClassStatistics computes statistics for triclass segmentation results
func (mathProc *IterativeTriclassMathProcessor) CalculateClassStatistics(foregroundCount, backgroundCount, tbdCount int) map[string]interface{} {
	totalPixels := foregroundCount + backgroundCount + tbdCount

	stats := make(map[string]interface{})
	stats["total_pixels"] = totalPixels
	stats["foreground_count"] = foregroundCount
	stats["background_count"] = backgroundCount
	stats["tbd_count"] = tbdCount

	if totalPixels > 0 {
		stats["foreground_ratio"] = float64(foregroundCount) / float64(totalPixels)
		stats["background_ratio"] = float64(backgroundCount) / float64(totalPixels)
		stats["tbd_ratio"] = float64(tbdCount) / float64(totalPixels)
	} else {
		stats["foreground_ratio"] = 0.0
		stats["background_ratio"] = 0.0
		stats["tbd_ratio"] = 0.0
	}

	// Calculate class balance entropy
	if totalPixels > 0 {
		entropy := 0.0
		ratios := []float64{
			float64(foregroundCount) / float64(totalPixels),
			float64(backgroundCount) / float64(totalPixels),
			float64(tbdCount) / float64(totalPixels),
		}

		for _, ratio := range ratios {
			if ratio > 0 {
				entropy -= ratio * math.Log2(ratio)
			}
		}
		stats["class_entropy"] = entropy
	} else {
		stats["class_entropy"] = 0.0
	}

	return stats
}

// ValidateIterationParameters checks mathematical validity of iteration parameters
func (mathProc *IterativeTriclassMathProcessor) ValidateIterationParameters() error {
	// Check convergence epsilon
	epsilon := mathProc.getFloatParam("convergence_epsilon", 1.0)
	if epsilon <= 0.0 || epsilon > 50.0 {
		return fmt.Errorf("convergence_epsilon must be between 0.0 and 50.0, got %.3f", epsilon)
	}

	// Check maximum iterations
	maxIter := mathProc.getIntParam("max_iterations", 10)
	if maxIter < 1 || maxIter > 100 {
		return fmt.Errorf("max_iterations must be between 1 and 100, got %d", maxIter)
	}

	// Check minimum TBD fraction
	minTBD := mathProc.getFloatParam("minimum_tbd_fraction", 0.01)
	if minTBD < 0.0001 || minTBD > 0.5 {
		return fmt.Errorf("minimum_tbd_fraction must be between 0.0001 and 0.5, got %.6f", minTBD)
	}

	// Check gap factor
	gapFactor := mathProc.getFloatParam("lower_upper_gap_factor", 0.5)
	if gapFactor < 0.0 || gapFactor > 1.0 {
		return fmt.Errorf("lower_upper_gap_factor must be between 0.0 and 1.0, got %.3f", gapFactor)
	}

	return nil
}

// CalculateConvergenceRate estimates convergence rate from iteration history
func (mathProc *IterativeTriclassMathProcessor) CalculateConvergenceRate(convergenceHistory []float64) float64 {
	if len(convergenceHistory) < 2 {
		return 0.0
	}

	// Calculate average convergence rate over last few iterations
	n := len(convergenceHistory)
	windowSize := int(math.Min(float64(n), 5)) // Use last 5 iterations

	totalRate := 0.0
	count := 0

	for i := n - windowSize; i < n-1; i++ {
		if convergenceHistory[i] > 0 {
			rate := convergenceHistory[i+1] / convergenceHistory[i]
			totalRate += rate
			count++
		}
	}

	if count > 0 {
		return totalRate / float64(count)
	}

	return 0.0
}

// EstimateRemainingIterations predicts iterations needed for convergence
func (mathProc *IterativeTriclassMathProcessor) EstimateRemainingIterations(currentConvergence float64, convergenceRate float64) int {
	epsilon := mathProc.getFloatParam("convergence_epsilon", 1.0)
	maxIter := mathProc.getIntParam("max_iterations", 10)

	if convergenceRate <= 0 || convergenceRate >= 1.0 || currentConvergence <= epsilon {
		return 0
	}

	// Geometric series estimation: C_n = C_0 * r^n
	// Solve for n when C_n = epsilon
	estimatedIterations := math.Log(epsilon/currentConvergence) / math.Log(convergenceRate)

	estimated := int(math.Ceil(estimatedIterations))

	// Clamp to reasonable bounds
	if estimated < 0 {
		return 0
	}
	if estimated > maxIter {
		return maxIter
	}

	return estimated
}

// Parameter access utilities
func (mathProc *IterativeTriclassMathProcessor) getStringParam(name string, defaultValue string) string {
	if value, ok := mathProc.params[name].(string); ok {
		return value
	}
	return defaultValue
}

func (mathProc *IterativeTriclassMathProcessor) getIntParam(name string, defaultValue int) int {
	if value, ok := mathProc.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (mathProc *IterativeTriclassMathProcessor) getFloatParam(name string, defaultValue float64) float64 {
	if value, ok := mathProc.params[name].(float64); ok {
		return value
	}
	return defaultValue
}
