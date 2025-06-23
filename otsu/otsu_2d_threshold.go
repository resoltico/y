package otsu

import (
	"fmt"
	"math"

	"gocv.io/x/gocv"
)

// TwoDThreshold represents a 2D threshold point
type TwoDThreshold struct {
	PixelThreshold   int
	FeatureThreshold int
	Variance         float64
}

// findOptimalThreshold implements the mathematical 2D Otsu criterion
func (core *TwoDOtsuCore) findOptimalThreshold() TwoDThreshold {
	stepTime := core.debugManager.StartTiming("2d_otsu_threshold_search")
	defer core.debugManager.EndTiming("2d_otsu_threshold_search", stepTime)

	bins := core.histogramData.bins
	bestThreshold := TwoDThreshold{
		PixelThreshold:   bins / 2,
		FeatureThreshold: bins / 2,
		Variance:         0.0,
	}

	// Calculate total statistics for the image
	totalStats := core.calculateTotalStatistics()

	// Exhaustive search for optimal threshold using 2D Otsu criterion
	maxBetweenClassVariance := 0.0

	// Use step size for performance optimization in Fast mode
	step := 1
	if core.getStringParam("quality") == "Fast" {
		step = 2
	}

	for t1 := 1; t1 < bins-1; t1 += step {
		for t2 := 1; t2 < bins-1; t2 += step {
			// Calculate class statistics for this threshold
			classStats := core.calculateClassStatistics(t1, t2, totalStats)

			// Calculate between-class variance (Otsu criterion)
			betweenClassVariance := core.calculateBetweenClassVariance(classStats)

			if betweenClassVariance > maxBetweenClassVariance {
				maxBetweenClassVariance = betweenClassVariance
				bestThreshold = TwoDThreshold{
					PixelThreshold:   t1,
					FeatureThreshold: t2,
					Variance:         betweenClassVariance,
				}
			}
		}
	}

	// If Fast mode was used, refine around the best threshold
	if step > 1 {
		bestThreshold = core.refineThreshold(bestThreshold, totalStats)
	}

	core.debugManager.LogThresholdCalculation("2D Otsu Threshold", 
		fmt.Sprintf("(%d,%d)", bestThreshold.PixelThreshold, bestThreshold.FeatureThreshold), 
		fmt.Sprintf("variance=%.6f", bestThreshold.Variance))

	return bestThreshold
}

// TotalStatistics holds global image statistics
type TotalStatistics struct {
	TotalWeight   float64
	TotalMoment1  float64 // First moment (mean)
	TotalMoment2  float64 // Second moment for variance calculation
	PixelMean     float64
	FeatureMean   float64
}

// calculateTotalStatistics computes global statistics for the 2D histogram
func (core *TwoDOtsuCore) calculateTotalStatistics() TotalStatistics {
	bins := core.histogramData.bins
	stats := TotalStatistics{}

	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := core.histogramData.histogram[i][j]
			
			stats.TotalWeight += weight
			stats.TotalMoment1 += weight * float64(i*bins+j)
			stats.TotalMoment2 += weight * float64(i*bins+j) * float64(i*bins+j)
			
			// Separate pixel and feature statistics
			stats.PixelMean += weight * float64(i)
			stats.FeatureMean += weight * float64(j)
		}
	}

	if stats.TotalWeight > 0 {
		stats.PixelMean /= stats.TotalWeight
		stats.FeatureMean /= stats.TotalWeight
	}

	return stats
}

// ClassStatistics holds statistics for foreground and background classes
type ClassStatistics struct {
	ForegroundWeight float64
	BackgroundWeight float64
	ForegroundMean   [2]float64 // [pixel_mean, feature_mean]
	BackgroundMean   [2]float64 // [pixel_mean, feature_mean]
	ForegroundVar    [2]float64 // [pixel_var, feature_var]
	BackgroundVar    [2]float64 // [pixel_var, feature_var]
}

// calculateClassStatistics computes statistics for both classes given a threshold
func (core *TwoDOtsuCore) calculateClassStatistics(t1, t2 int, totalStats TotalStatistics) ClassStatistics {
	bins := core.histogramData.bins
	stats := ClassStatistics{}

	// Calculate class weights and moments
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := core.histogramData.histogram[i][j]

			if i <= t1 && j <= t2 {
				// Background class
				stats.BackgroundWeight += weight
				stats.BackgroundMean[0] += weight * float64(i)
				stats.BackgroundMean[1] += weight * float64(j)
			} else {
				// Foreground class (everything else)
				stats.ForegroundWeight += weight
				stats.ForegroundMean[0] += weight * float64(i)
				stats.ForegroundMean[1] += weight * float64(j)
			}
		}
	}

	// Calculate means
	if stats.BackgroundWeight > 0 {
		stats.BackgroundMean[0] /= stats.BackgroundWeight
		stats.BackgroundMean[1] /= stats.BackgroundWeight
	}

	if stats.ForegroundWeight > 0 {
		stats.ForegroundMean[0] /= stats.ForegroundWeight
		stats.ForegroundMean[1] /= stats.ForegroundWeight
	}

	// Calculate variances
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := core.histogramData.histogram[i][j]

			if i <= t1 && j <= t2 && stats.BackgroundWeight > 0 {
				// Background variance
				pixelDiff := float64(i) - stats.BackgroundMean[0]
				featureDiff := float64(j) - stats.BackgroundMean[1]
				stats.BackgroundVar[0] += weight * pixelDiff * pixelDiff
				stats.BackgroundVar[1] += weight * featureDiff * featureDiff
			} else if stats.ForegroundWeight > 0 {
				// Foreground variance
				pixelDiff := float64(i) - stats.ForegroundMean[0]
				featureDiff := float64(j) - stats.ForegroundMean[1]
				stats.ForegroundVar[0] += weight * pixelDiff * pixelDiff
				stats.ForegroundVar[1] += weight * featureDiff * featureDiff
			}
		}
	}

	if stats.BackgroundWeight > 0 {
		stats.BackgroundVar[0] /= stats.BackgroundWeight
		stats.BackgroundVar[1] /= stats.BackgroundWeight
	}

	if stats.ForegroundWeight > 0 {
		stats.ForegroundVar[0] /= stats.ForegroundWeight
		stats.ForegroundVar[1] /= stats.ForegroundWeight
	}

	return stats
}

// calculateBetweenClassVariance implements the 2D Otsu between-class variance formula
func (core *TwoDOtsuCore) calculateBetweenClassVariance(stats ClassStatistics) float64 {
	if stats.BackgroundWeight <= 0 || stats.ForegroundWeight <= 0 {
		return 0.0
	}

	totalWeight := stats.BackgroundWeight + stats.ForegroundWeight
	if totalWeight <= 0 {
		return 0.0
	}

	// Normalized weights
	w0 := stats.BackgroundWeight / totalWeight
	w1 := stats.ForegroundWeight / totalWeight

	// Calculate between-class variance for both pixel and feature dimensions
	pixelMeanDiff := stats.ForegroundMean[0] - stats.BackgroundMean[0]
	featureMeanDiff := stats.ForegroundMean[1] - stats.BackgroundMean[1]

	// 2D between-class variance (Fisher discriminant in 2D)
	betweenClassVariance := w0 * w1 * (pixelMeanDiff*pixelMeanDiff + featureMeanDiff*featureMeanDiff)

	return betweenClassVariance
}

// refineThreshold performs local search around the coarse threshold
func (core *TwoDOtsuCore) refineThreshold(coarseThreshold TwoDThreshold, totalStats TotalStatistics) TwoDThreshold {
	stepTime := core.debugManager.StartTiming("2d_otsu_threshold_refinement")
	defer core.debugManager.EndTiming("2d_otsu_threshold_refinement", stepTime)

	bestThreshold := coarseThreshold
	maxVariance := coarseThreshold.Variance

	// Search in a 5x5 neighborhood around the coarse threshold
	searchRadius := 2
	bins := core.histogramData.bins

	for dt1 := -searchRadius; dt1 <= searchRadius; dt1++ {
		for dt2 := -searchRadius; dt2 <= searchRadius; dt2++ {
			t1 := coarseThreshold.PixelThreshold + dt1
			t2 := coarseThreshold.FeatureThreshold + dt2

			// Check bounds
			if t1 <= 0 || t1 >= bins-1 || t2 <= 0 || t2 >= bins-1 {
				continue
			}

			classStats := core.calculateClassStatistics(t1, t2, totalStats)
			variance := core.calculateBetweenClassVariance(classStats)

			if variance > maxVariance {
				maxVariance = variance
				bestThreshold = TwoDThreshold{
					PixelThreshold:   t1,
					FeatureThreshold: t2,
					Variance:         variance,
				}
			}
		}
	}

	return bestThreshold
}

// applyBinaryThreshold creates the final binary image using the calculated threshold
func (core *TwoDOtsuCore) applyBinaryThreshold(src, neighborhood *gocv.Mat, threshold TwoDThreshold) gocv.Mat {
	stepTime := core.debugManager.StartTiming("2d_otsu_binary_threshold_application")
	defer core.debugManager.EndTiming("2d_otsu_binary_threshold_application", stepTime)

	result := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	
	bins := core.histogramData.bins
	pixelWeight := core.getFloatParam("pixel_weight_factor")
	
	// Convert threshold back to intensity values
	pixelThresholdVal := float64(threshold.PixelThreshold) * 255.0 / float64(bins-1)
	featureThresholdVal := float64(threshold.FeatureThreshold) * 255.0 / float64(bins-1)

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelVal := float64(src.GetUCharAt(y, x))
			neighVal := float64(neighborhood.GetUCharAt(y, x))

			// Calculate the same blended feature as used in histogram building
			blendedFeature := pixelWeight*pixelVal + (1.0-pixelWeight)*neighVal

			// Apply 2D threshold
			if pixelVal > pixelThresholdVal && blendedFeature > featureThresholdVal {
				result.SetUCharAt(y, x, 255) // Foreground
			} else {
				result.SetUCharAt(y, x, 0)   // Background
			}
		}
	}

	core.debugManager.LogAlgorithmStep("2D Otsu", "binary_threshold_applied", stepTime)
	return result
}

// GetThresholdInfo returns human-readable threshold information
func (threshold *TwoDThreshold) GetThresholdInfo() map[string]interface{} {
	return map[string]interface{}{
		"pixel_threshold":   threshold.PixelThreshold,
		"feature_threshold": threshold.FeatureThreshold,
		"variance":          threshold.Variance,
		"threshold_type":    "2D_Otsu",
	}
}

// IsValid checks if the threshold is mathematically valid
func (threshold *TwoDThreshold) IsValid(maxBins int) bool {
	return threshold.PixelThreshold >= 0 && 
		   threshold.PixelThreshold < maxBins &&
		   threshold.FeatureThreshold >= 0 && 
		   threshold.FeatureThreshold < maxBins &&
		   threshold.Variance >= 0.0
}