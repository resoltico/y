package otsu

import (
	"fmt"
	"math"
	"time"

	"otsu-obliterator/debug"
)

// TwoDMathProcessor handles mathematical computations for 2D Otsu thresholding
type TwoDMathProcessor struct {
	params       map[string]interface{}
	debugManager *debug.Manager
}

// NewTwoDMathProcessor creates a new mathematical processor
func NewTwoDMathProcessor(params map[string]interface{}) *TwoDMathProcessor {
	return &TwoDMathProcessor{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

// FindOptimalThreshold implements 2D Otsu criterion for threshold selection
func (processor *TwoDMathProcessor) FindOptimalThreshold(histData *TwoDHistogramData) (TwoDThreshold, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "threshold_search", time.Since(stepTime))

	if histData == nil {
		return TwoDThreshold{}, fmt.Errorf("histogram data is nil")
	}

	if histData.bins < 2 {
		return TwoDThreshold{}, fmt.Errorf("histogram has insufficient bins: %d", histData.bins)
	}

	// Calculate global statistics for the 2D histogram
	totalStats, err := processor.calculateGlobalStatistics(histData)
	if err != nil {
		return TwoDThreshold{}, err
	}

	// Find threshold using exhaustive search with Otsu criterion
	bestThreshold, err := processor.searchOptimalThreshold(histData, totalStats)
	if err != nil {
		return TwoDThreshold{}, err
	}

	processor.debugManager.LogThresholdCalculation("2D Otsu Math",
		fmt.Sprintf("(%d,%d)", bestThreshold.PixelThreshold, bestThreshold.FeatureThreshold),
		fmt.Sprintf("variance=%.6f", bestThreshold.Variance))

	return bestThreshold, nil
}

// calculateGlobalStatistics computes global 2D histogram statistics
func (processor *TwoDMathProcessor) calculateGlobalStatistics(histData *TwoDHistogramData) (*GlobalStatistics2D, error) {
	stats := &GlobalStatistics2D{}
	bins := histData.bins

	// Calculate moments and total weight
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := histData.histogram[i][j]

			stats.TotalWeight += weight
			stats.WeightedSumI += weight * float64(i)
			stats.WeightedSumJ += weight * float64(j)
			stats.WeightedSumIJ += weight * float64(i) * float64(j)
			stats.WeightedSumI2 += weight * float64(i) * float64(i)
			stats.WeightedSumJ2 += weight * float64(j) * float64(j)
		}
	}

	// Calculate means
	if stats.TotalWeight > 0 {
		stats.MeanI = stats.WeightedSumI / stats.TotalWeight
		stats.MeanJ = stats.WeightedSumJ / stats.TotalWeight
	}

	// Calculate variances
	if stats.TotalWeight > 0 {
		stats.VarianceI = (stats.WeightedSumI2 / stats.TotalWeight) - (stats.MeanI * stats.MeanI)
		stats.VarianceJ = (stats.WeightedSumJ2 / stats.TotalWeight) - (stats.MeanJ * stats.MeanJ)
		stats.CovarianceIJ = (stats.WeightedSumIJ / stats.TotalWeight) - (stats.MeanI * stats.MeanJ)
	}

	if stats.TotalWeight <= 0 {
		return nil, fmt.Errorf("histogram contains no data")
	}

	return stats, nil
}

// searchOptimalThreshold performs exhaustive search for optimal 2D threshold
func (processor *TwoDMathProcessor) searchOptimalThreshold(histData *TwoDHistogramData, globalStats *GlobalStatistics2D) (TwoDThreshold, error) {
	bins := histData.bins
	
	bestThreshold := TwoDThreshold{
		PixelThreshold:   bins / 2,
		FeatureThreshold: bins / 2,
		Variance:         0.0,
	}

	maxBetweenClassVariance := 0.0

	// Determine search step size based on quality setting
	step := 1
	if processor.getStringParam("quality", "Fast") == "Fast" {
		step = 2
	}

	// Exhaustive search over threshold space
	for t1 := 1; t1 < bins-1; t1 += step {
		for t2 := 1; t2 < bins-1; t2 += step {
			// Calculate class statistics for this threshold pair
			classStats, err := processor.calculateClassStatistics(histData, t1, t2, globalStats)
			if err != nil {
				continue // Skip invalid threshold combinations
			}

			// Calculate between-class variance using 2D Otsu criterion
			betweenClassVariance := processor.calculateBetweenClassVariance2D(classStats)

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

	// Refine threshold if coarse search was used
	if step > 1 {
		refinedThreshold, err := processor.refineThreshold(histData, bestThreshold, globalStats)
		if err == nil {
			bestThreshold = refinedThreshold
		}
	}

	return bestThreshold, nil
}

// calculateClassStatistics computes statistics for background and foreground classes
func (processor *TwoDMathProcessor) calculateClassStatistics(histData *TwoDHistogramData, t1, t2 int, globalStats *GlobalStatistics2D) (*ClassStatistics2D, error) {
	stats := &ClassStatistics2D{}
	bins := histData.bins

	// Calculate class weights and first moments
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := histData.histogram[i][j]

			if i <= t1 && j <= t2 {
				// Background class (class 0)
				stats.Weight0 += weight
				stats.WeightedSumI0 += weight * float64(i)
				stats.WeightedSumJ0 += weight * float64(j)
			} else {
				// Foreground class (class 1) - everything else
				stats.Weight1 += weight
				stats.WeightedSumI1 += weight * float64(i)
				stats.WeightedSumJ1 += weight * float64(j)
			}
		}
	}

	// Calculate class means
	if stats.Weight0 > 0 {
		stats.MeanI0 = stats.WeightedSumI0 / stats.Weight0
		stats.MeanJ0 = stats.WeightedSumJ0 / stats.Weight0
	}

	if stats.Weight1 > 0 {
		stats.MeanI1 = stats.WeightedSumI1 / stats.Weight1
		stats.MeanJ1 = stats.WeightedSumJ1 / stats.Weight1
	}

	// Validate class statistics
	totalWeight := stats.Weight0 + stats.Weight1
	if math.Abs(totalWeight-globalStats.TotalWeight) > 1e-6 {
		return nil, fmt.Errorf("class weight sum mismatch: %.6f vs %.6f", totalWeight, globalStats.TotalWeight)
	}

	if stats.Weight0 <= 0 || stats.Weight1 <= 0 {
		return nil, fmt.Errorf("invalid class weights: w0=%.6f, w1=%.6f", stats.Weight0, stats.Weight1)
	}

	return stats, nil
}

// calculateBetweenClassVariance2D computes 2D between-class variance
func (processor *TwoDMathProcessor) calculateBetweenClassVariance2D(stats *ClassStatistics2D) float64 {
	totalWeight := stats.Weight0 + stats.Weight1
	
	if totalWeight <= 0 {
		return 0.0
	}

	// Normalized class weights
	w0 := stats.Weight0 / totalWeight
	w1 := stats.Weight1 / totalWeight

	// Calculate mean differences for both dimensions
	meanDiffI := stats.MeanI1 - stats.MeanI0
	meanDiffJ := stats.MeanJ1 - stats.MeanJ0

	// 2D between-class variance formula
	// This is the trace of the between-class covariance matrix
	betweenClassVariance := w0 * w1 * (meanDiffI*meanDiffI + meanDiffJ*meanDiffJ)

	return betweenClassVariance
}

// refineThreshold performs local search around coarse threshold
func (processor *TwoDMathProcessor) refineThreshold(histData *TwoDHistogramData, coarseThreshold TwoDThreshold, globalStats *GlobalStatistics2D) (TwoDThreshold, error) {
	stepTime := time.Now()
	defer processor.debugManager.LogAlgorithmStep("2D Otsu", "threshold_refinement", time.Since(stepTime))

	bestThreshold := coarseThreshold
	maxVariance := coarseThreshold.Variance

	// Search in a 5x5 neighborhood around coarse threshold
	searchRadius := 2
	bins := histData.bins

	for dt1 := -searchRadius; dt1 <= searchRadius; dt1++ {
		for dt2 := -searchRadius; dt2 <= searchRadius; dt2++ {
			t1 := coarseThreshold.PixelThreshold + dt1
			t2 := coarseThreshold.FeatureThreshold + dt2

			// Check bounds
			if t1 <= 0 || t1 >= bins-1 || t2 <= 0 || t2 >= bins-1 {
				continue
			}

			classStats, err := processor.calculateClassStatistics(histData, t1, t2, globalStats)
			if err != nil {
				continue
			}

			variance := processor.calculateBetweenClassVariance2D(classStats)

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

	return bestThreshold, nil
}

// ValidateThreshold checks if threshold is mathematically valid
func (processor *TwoDMathProcessor) ValidateThreshold(threshold TwoDThreshold, bins int) error {
	if threshold.PixelThreshold < 0 || threshold.PixelThreshold >= bins {
		return fmt.Errorf("pixel threshold %d out of range [0, %d)", threshold.PixelThreshold, bins)
	}

	if threshold.FeatureThreshold < 0 || threshold.FeatureThreshold >= bins {
		return fmt.Errorf("feature threshold %d out of range [0, %d)", threshold.FeatureThreshold, bins)
	}

	if threshold.Variance < 0 {
		return fmt.Errorf("variance cannot be negative: %.6f", threshold.Variance)
	}

	if math.IsNaN(threshold.Variance) || math.IsInf(threshold.Variance, 0) {
		return fmt.Errorf("variance is not finite: %.6f", threshold.Variance)
	}

	return nil
}

// CalculateThresholdQuality assesses threshold quality using separability measures
func (processor *TwoDMathProcessor) CalculateThresholdQuality(histData *TwoDHistogramData, threshold TwoDThreshold, globalStats *GlobalStatistics2D) (map[string]float64, error) {
	quality := make(map[string]float64)

	// Calculate class statistics for the threshold
	classStats, err := processor.calculateClassStatistics(histData, threshold.PixelThreshold, threshold.FeatureThreshold, globalStats)
	if err != nil {
		return nil, err
	}

	// Between-class variance (already calculated)
	quality["between_class_variance"] = threshold.Variance

	// Within-class variance
	withinClassVariance := processor.calculateWithinClassVariance2D(histData, threshold, classStats)
	quality["within_class_variance"] = withinClassVariance

	// Total variance (should equal between + within)
	totalVariance := globalStats.VarianceI + globalStats.VarianceJ
	quality["total_variance"] = totalVariance

	// Separability measure (ratio of between to within class variance)
	if withinClassVariance > 0 {
		quality["separability"] = threshold.Variance / withinClassVariance
	} else {
		quality["separability"] = math.Inf(1)
	}

	// Class balance measure
	totalWeight := classStats.Weight0 + classStats.Weight1
	if totalWeight > 0 {
		balance0 := classStats.Weight0 / totalWeight
		balance1 := classStats.Weight1 / totalWeight
		// Balance entropy (higher is more balanced)
		quality["class_balance"] = -(balance0*math.Log2(balance0) + balance1*math.Log2(balance1))
	}

	return quality, nil
}

// calculateWithinClassVariance2D computes within-class variance for both classes
func (processor *TwoDMathProcessor) calculateWithinClassVariance2D(histData *TwoDHistogramData, threshold TwoDThreshold, classStats *ClassStatistics2D) float64 {
	bins := histData.bins
	withinVar0 := 0.0
	withinVar1 := 0.0

	// Calculate within-class variances
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			weight := histData.histogram[i][j]

			if i <= threshold.PixelThreshold && j <= threshold.FeatureThreshold {
				// Background class
				if classStats.Weight0 > 0 {
					diffI := float64(i) - classStats.MeanI0
					diffJ := float64(j) - classStats.MeanJ0
					withinVar0 += weight * (diffI*diffI + diffJ*diffJ)
				}
			} else {
				// Foreground class
				if classStats.Weight1 > 0 {
					diffI := float64(i) - classStats.MeanI1
					diffJ := float64(j) - classStats.MeanJ1
					withinVar1 += weight * (diffI*diffI + diffJ*diffJ)
				}
			}
		}
	}

	// Normalize by class weights
	if classStats.Weight0 > 0 {
		withinVar0 /= classStats.Weight0
	}
	if classStats.Weight1 > 0 {
		withinVar1 /= classStats.Weight1
	}

	// Weighted average of within-class variances
	totalWeight := classStats.Weight0 + classStats.Weight1
	if totalWeight > 0 {
		return (classStats.Weight0*withinVar0 + classStats.Weight1*withinVar1) / totalWeight
	}

	return 0.0
}

// Parameter access utilities
func (processor *TwoDMathProcessor) getStringParam(name string, defaultValue string) string {
	if value, ok := processor.params[name].(string); ok {
		return value
	}
	return defaultValue
}

// GlobalStatistics2D holds global 2D histogram statistics
type GlobalStatistics2D struct {
	TotalWeight    float64
	WeightedSumI   float64
	WeightedSumJ   float64
	WeightedSumIJ  float64
	WeightedSumI2  float64
	WeightedSumJ2  float64
	MeanI          float64
	MeanJ          float64
	VarianceI      float64
	VarianceJ      float64
	CovarianceIJ   float64
}

// ClassStatistics2D holds statistics for both classes in 2D
type ClassStatistics2D struct {
	Weight0        float64
	Weight1        float64
	WeightedSumI0  float64
	WeightedSumJ0  float64
	WeightedSumI1  float64
	WeightedSumJ1  float64
	MeanI0         float64
	MeanJ0         float64
	MeanI1         float64
	MeanJ1         float64
}