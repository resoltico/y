package otsu

import (
	"fmt"
	"math"

	"gocv.io/x/gocv"
)

// calculateRegionThreshold computes the threshold for the current active region
func (core *IterativeTriclassCore) calculateRegionThreshold() (float64, error) {
	stepTime := core.debugManager.StartTiming("triclass_threshold_calculation")
	defer core.debugManager.EndTiming("triclass_threshold_calculation", stepTime)

	// Build histogram for active region only
	histogram, err := core.buildRegionHistogram()
	if err != nil {
		return 0, err
	}

	method := core.getStringParam("initial_threshold_method")
	var threshold float64

	switch method {
	case "mean":
		threshold = core.calculateMeanThreshold(histogram)
	case "median":
		threshold = core.calculateMedianThreshold(histogram)
	case "otsu":
		threshold = core.calculateOtsuThreshold(histogram)
	default:
		threshold = core.calculateOtsuThreshold(histogram)
	}

	core.debugManager.LogThresholdCalculation("Iterative Triclass", threshold, method)
	return threshold, nil
}

// buildRegionHistogram constructs histogram only for active (non-zero) pixels
func (core *IterativeTriclassCore) buildRegionHistogram() ([]int, error) {
	histBins := core.getIntParam("histogram_bins")
	histogram := make([]int, histBins)

	if core.iterationData.CurrentRegion.Empty() {
		return histogram, fmt.Errorf("current region is empty")
	}

	rows := core.iterationData.CurrentRegion.Rows()
	cols := core.iterationData.CurrentRegion.Cols()

	// Build histogram only from active pixels (non-zero values)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := core.iterationData.CurrentRegion.GetUCharAt(y, x)

			// Only include active pixels in histogram
			if pixelValue > 0 {
				bin := int(float64(pixelValue) * float64(histBins-1) / 255.0)
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

// calculateOtsuThreshold implements standard Otsu thresholding on histogram
func (core *IterativeTriclassCore) calculateOtsuThreshold(histogram []int) float64 {
	histBins := len(histogram)
	total := 0

	// Calculate total pixel count
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return 127.5 // Default threshold if no active pixels
	}

	// Calculate cumulative sum and weighted sum
	sum := 0.0
	for i := 0; i < histBins; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5

	for t := 0; t < histBins; t++ {
		wB += histogram[t]
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += float64(t) * float64(histogram[t])

		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)

		// Between-class variance (Otsu criterion)
		variance := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if variance > maxVariance {
			maxVariance = variance
			bestThreshold = float64(t) * 255.0 / float64(histBins-1)
		}
	}

	return bestThreshold
}

// calculateMeanThreshold computes mean-based threshold
func (core *IterativeTriclassCore) calculateMeanThreshold(histogram []int) float64 {
	histBins := len(histogram)
	totalPixels := 0
	weightedSum := 0.0

	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
		weightedSum += float64(i) * float64(histogram[i])
	}

	if totalPixels == 0 {
		return 127.5
	}

	meanBin := weightedSum / float64(totalPixels)
	return meanBin * 255.0 / float64(histBins-1)
}

// calculateMedianThreshold computes median-based threshold
func (core *IterativeTriclassCore) calculateMedianThreshold(histogram []int) float64 {
	histBins := len(histogram)
	totalPixels := 0

	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := totalPixels / 2
	cumSum := 0

	for i := 0; i < histBins; i++ {
		cumSum += histogram[i]
		if cumSum >= halfPixels {
			return float64(i) * 255.0 / float64(histBins-1)
		}
	}

	return 127.5
}

// performTriclassSegmentation segments the current region into three classes
func (core *IterativeTriclassCore) performTriclassSegmentation(threshold float64) (gocv.Mat, gocv.Mat, gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("triclass_segmentation")
	defer core.debugManager.EndTiming("triclass_segmentation", stepTime)

	if core.iterationData.CurrentRegion.Empty() {
		return gocv.NewMat(), gocv.NewMat(), gocv.NewMat(), fmt.Errorf("current region is empty")
	}

	rows := core.iterationData.CurrentRegion.Rows()
	cols := core.iterationData.CurrentRegion.Cols()

	// Create output masks
	foreground := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	background := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	tbd := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	// Initialize all masks to zero
	foreground.SetTo(gocv.NewScalar(0, 0, 0, 0))
	background.SetTo(gocv.NewScalar(0, 0, 0, 0))
	tbd.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Calculate adaptive thresholds with gap factor
	gapFactor := core.getFloatParam("lower_upper_gap_factor")
	lowerThreshold := threshold * (1.0 - gapFactor)
	upperThreshold := threshold * (1.0 + gapFactor)

	// Ensure thresholds are within valid range
	lowerThreshold = math.Max(0.0, math.Min(255.0, lowerThreshold))
	upperThreshold = math.Max(0.0, math.Min(255.0, upperThreshold))

	// Ensure upper > lower
	if upperThreshold <= lowerThreshold {
		upperThreshold = lowerThreshold + 1.0
	}

	// Perform triclass segmentation
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := float64(core.iterationData.CurrentRegion.GetUCharAt(y, x))

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
func (core *IterativeTriclassCore) updateFinalResult(foreground, background *gocv.Mat) {
	stepTime := core.debugManager.StartTiming("triclass_result_update")
	defer core.debugManager.EndTiming("triclass_result_update", stepTime)

	rows := core.iterationData.FinalResult.Rows()
	cols := core.iterationData.FinalResult.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Set foreground pixels to 255
			if foreground.GetUCharAt(y, x) > 0 {
				core.iterationData.FinalResult.SetUCharAt(y, x, 255)
			}
			// Background pixels remain 0 (already initialized)
		}
	}

	// Update processed pixels count
	foregroundPixels := gocv.CountNonZero(*foreground)
	backgroundPixels := gocv.CountNonZero(*background)
	core.iterationData.ProcessedPixels += foregroundPixels + backgroundPixels
}

// updateCurrentRegion prepares the next iteration region from TBD pixels
func (core *IterativeTriclassCore) updateCurrentRegion(tbdMask *gocv.Mat) error {
	stepTime := core.debugManager.StartTiming("triclass_region_update")
	defer core.debugManager.EndTiming("triclass_region_update", stepTime)

	// Create new region containing only TBD pixels with their original values
	newRegion := gocv.NewMatWithSize(core.iterationData.CurrentRegion.Rows(), 
		core.iterationData.CurrentRegion.Cols(), gocv.MatTypeCV8UC1)
	newRegion.SetTo(gocv.NewScalar(0, 0, 0, 0))

	rows := core.iterationData.CurrentRegion.Rows()
	cols := core.iterationData.CurrentRegion.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdMask.GetUCharAt(y, x) > 0 {
				// Copy original pixel value to new region
				originalValue := core.iterationData.CurrentRegion.GetUCharAt(y, x)
				newRegion.SetUCharAt(y, x, originalValue)
			}
		}
	}

	// Replace current region
	core.iterationData.CurrentRegion.Close()
	core.iterationData.CurrentRegion = newRegion

	// Update active pixel count
	core.iterationData.ActivePixels = gocv.CountNonZero(newRegion)

	return nil
}

// assignRemainingTBDPixels assigns final TBD pixels using simple threshold
func (core *IterativeTriclassCore) assignRemainingTBDPixels(tbdMask *gocv.Mat, threshold float64) {
	stepTime := core.debugManager.StartTiming("triclass_final_assignment")
	defer core.debugManager.EndTiming("triclass_final_assignment", stepTime)

	rows := tbdMask.Rows()
	cols := tbdMask.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdMask.GetUCharAt(y, x) > 0 {
				// Get original pixel value
				originalValue := float64(core.iterationData.CurrentRegion.GetUCharAt(y, x))
				
				// Simple threshold assignment
				if originalValue >= threshold {
					core.iterationData.FinalResult.SetUCharAt(y, x, 255) // Foreground
				}
				// Background pixels remain 0
			}
		}
	}
}

// GetConvergenceLog returns the convergence information for analysis
func (core *IterativeTriclassCore) GetConvergenceLog() []ConvergenceInfo {
	return core.convergenceLog
}

// GetIterationCount returns the number of iterations performed
func (core *IterativeTriclassCore) GetIterationCount() int {
	if core.iterationData != nil {
		return core.iterationData.IterationCount
	}
	return 0
}

// GetProcessingStatistics returns statistics about the processing
func (core *IterativeTriclassCore) GetProcessingStatistics() map[string]interface{} {
	if core.iterationData == nil {
		return make(map[string]interface{})
	}

	finalForegroundPixels := gocv.CountNonZero(core.iterationData.FinalResult)
	finalBackgroundPixels := core.iterationData.TotalPixels - finalForegroundPixels

	return map[string]interface{}{
		"total_pixels":          core.iterationData.TotalPixels,
		"processed_pixels":      core.iterationData.ProcessedPixels,
		"final_foreground":      finalForegroundPixels,
		"final_background":      finalBackgroundPixels,
		"iteration_count":       core.iterationData.IterationCount,
		"convergence_achieved":  len(core.convergenceLog) > 0,
		"final_convergence":     core.getFinalConvergence(),
	}
}

// getFinalConvergence returns the final convergence value
func (core *IterativeTriclassCore) getFinalConvergence() float64 {
	if len(core.convergenceLog) == 0 {
		return 0.0
	}
	return core.convergenceLog[len(core.convergenceLog)-1].ConvergenceValue
}
