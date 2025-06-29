package threshold

import (
	"context"
	"fmt"
	"math"

	"otsu-obliterator/internal/opencv/safe"
)

type TriclassCalculator struct{}

func NewTriclassCalculator() *TriclassCalculator {
	return &TriclassCalculator{}
}

func (t *TriclassCalculator) ProcessIterative(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	maxIterations := t.getIntParam(params, "max_iterations", 8)
	convergencePrecision := t.getFloatParam(params, "convergence_precision", 1.0)
	minTBDFraction := t.getFloatParam(params, "minimum_tbd_fraction", 0.01)

	result, err := safe.NewMat(input.Rows(), input.Cols(), input.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	currentRegion, err := input.Clone()
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("failed to clone input Mat: %w", err)
	}
	defer currentRegion.Close()

	previousThreshold := -1.0
	totalPixels := float64(currentRegion.Rows() * currentRegion.Cols())
	stableIterations := 0
	const requiredStableIterations = 2

	for iteration := 0; iteration < maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		nonZeroPixels := t.countNonZeroPixels(currentRegion)
		if nonZeroPixels == 0 {
			break
		}

		threshold := t.calculateThresholdAdaptive(currentRegion, params)

		// Convergence detection with stability requirement
		convergence := math.Abs(threshold - previousThreshold)

		if convergence < convergencePrecision {
			stableIterations++
			if stableIterations >= requiredStableIterations {
				break
			}
		} else {
			stableIterations = 0
		}

		previousThreshold = threshold

		foregroundMask, backgroundMask, tbdMask, err := t.segmentRegionAdaptiveGaps(currentRegion, threshold, params)
		if err != nil {
			return nil, fmt.Errorf("segmentation failed at iteration %d: %w", iteration, err)
		}

		tbdCount := t.countNonZeroPixels(tbdMask)
		tbdFraction := float64(tbdCount) / totalPixels

		t.updateResult(result, foregroundMask)

		foregroundMask.Close()
		backgroundMask.Close()

		if tbdFraction < minTBDFraction {
			tbdMask.Close()
			break
		}

		newRegion, err := t.extractTBDRegion(input, tbdMask)
		tbdMask.Close()
		if err != nil {
			return nil, fmt.Errorf("TBD region extraction failed: %w", err)
		}

		currentRegion.Close()
		currentRegion = newRegion
	}

	return result, nil
}

func (t *TriclassCalculator) calculateThresholdAdaptive(region *safe.Mat, params map[string]interface{}) float64 {
	method := t.getStringParam(params, "initial_threshold_method", "otsu")
	histBins := t.getIntParam(params, "histogram_bins", 0)

	if histBins == 0 {
		histBins = t.calculateAdaptiveHistogramBins(region)
	}

	histogram := t.calculateHistogram(region, histBins)

	// Detect histogram characteristics for method selection
	if method == "otsu" && !t.isHistogramBimodal(histogram) {
		method = "triangle"
	}

	switch method {
	case "mean":
		return t.calculateMeanThresholdPrecise(histogram, histBins)
	case "median":
		return t.calculateMedianThresholdPrecise(histogram, histBins)
	case "triangle":
		return t.calculateTriangleThreshold(histogram, histBins)
	default:
		return t.calculateOtsuThresholdStable(histogram, histBins)
	}
}

func (t *TriclassCalculator) segmentRegionAdaptiveGaps(region *safe.Mat, threshold float64, params map[string]interface{}) (*safe.Mat, *safe.Mat, *safe.Mat, error) {
	rows := region.Rows()
	cols := region.Cols()

	foreground, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create foreground Mat: %w", err)
	}

	background, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		return nil, nil, nil, fmt.Errorf("failed to create background Mat: %w", err)
	}

	tbd, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		background.Close()
		return nil, nil, nil, fmt.Errorf("failed to create TBD Mat: %w", err)
	}

	classSeparation := t.getFloatParam(params, "class_separation", 0.5)

	// Adaptive gap calculation based on threshold position
	adaptiveGap := classSeparation
	if threshold < 64 {
		adaptiveGap *= 1.3 // Increase gap for dark regions
	} else if threshold > 192 {
		adaptiveGap *= 0.7 // Decrease gap for bright regions
	}

	// Calculate thresholds with numerical stability
	lowerThreshold := threshold * (1.0 - adaptiveGap)
	upperThreshold := threshold * (1.0 + adaptiveGap)

	// Ensure valid threshold range
	lowerThreshold = math.Max(0, lowerThreshold)
	upperThreshold = math.Min(255, upperThreshold)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := region.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			if pixelValue > 0 {
				pixelFloat := float64(pixelValue)
				if pixelFloat > upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if pixelFloat < lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd, nil
}

func (t *TriclassCalculator) calculateOtsuThresholdStable(histogram []int, histBins int) float64 {
	total := 0
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return 127.5
	}

	sum := 0.0
	for i := 0; i < histBins; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5
	invTotal := 1.0 / float64(total)
	binToValue := 255.0 / float64(histBins-1)

	// Sub-pixel precision search with stability checks
	subPixelStep := 0.1
	for t := 0.0; t < float64(histBins); t += subPixelStep {
		tInt := int(t)
		if tInt >= histBins {
			break
		}

		// Interpolated weight calculation
		weight := float64(histogram[tInt])
		if tInt+1 < histBins {
			fraction := t - float64(tInt)
			weight = weight*(1.0-fraction) + float64(histogram[tInt+1])*fraction
		}

		wB += int(weight)
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += t * weight

		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)
		meanDiff := mB - mF

		// Check for numerical stability
		if math.Abs(meanDiff) < 1e-10 {
			continue
		}

		varBetween := float64(wB) * float64(wF) * invTotal * meanDiff * meanDiff

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = t * binToValue
		}
	}

	return bestThreshold
}

func (t *TriclassCalculator) calculateTriangleThreshold(histogram []int, histBins int) float64 {
	// Find histogram peak
	maxCount := 0
	peakIndex := 0
	for i := 0; i < histBins; i++ {
		if histogram[i] > maxCount {
			maxCount = histogram[i]
			peakIndex = i
		}
	}

	// Find histogram endpoints
	leftEnd := 0
	rightEnd := histBins - 1

	for i := 0; i < histBins; i++ {
		if histogram[i] > 0 {
			leftEnd = i
			break
		}
	}

	for i := histBins - 1; i >= 0; i-- {
		if histogram[i] > 0 {
			rightEnd = i
			break
		}
	}

	// Triangle method: find point with maximum distance to line connecting peak and far end
	var maxDistance float64
	bestThreshold := float64(peakIndex)

	// Determine which end to use based on histogram skew
	farEnd := rightEnd
	if peakIndex-leftEnd > rightEnd-peakIndex {
		farEnd = leftEnd
	}

	// Calculate line parameters
	x1, y1 := float64(peakIndex), float64(maxCount)
	x2, y2 := float64(farEnd), float64(histogram[farEnd])

	if x1 != x2 {
		for i := min(peakIndex, farEnd); i <= max(peakIndex, farEnd); i++ {
			// Distance from point to line
			distance := math.Abs((y2-y1)*float64(i)-(x2-x1)*float64(histogram[i])+x2*y1-y2*x1) /
				math.Sqrt((y2-y1)*(y2-y1)+(x2-x1)*(x2-x1))

			if distance > maxDistance {
				maxDistance = distance
				bestThreshold = float64(i)
			}
		}
	}

	return bestThreshold * 255.0 / float64(histBins-1)
}

func (t *TriclassCalculator) calculateMeanThresholdPrecise(histogram []int, histBins int) float64 {
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

func (t *TriclassCalculator) calculateMedianThresholdPrecise(histogram []int, histBins int) float64 {
	totalPixels := 0
	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := float64(totalPixels) / 2.0
	cumSum := 0.0

	for i := 0; i < histBins; i++ {
		cumSum += float64(histogram[i])
		if cumSum >= halfPixels {
			// Sub-pixel interpolation for median
			if i > 0 && cumSum > halfPixels {
				excess := cumSum - halfPixels
				fraction := excess / float64(histogram[i])
				interpolatedBin := float64(i) - fraction
				return interpolatedBin * 255.0 / float64(histBins-1)
			}
			return float64(i) * 255.0 / float64(histBins-1)
		}
	}

	return 127.5
}

// Helper functions
func (t *TriclassCalculator) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (t *TriclassCalculator) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (t *TriclassCalculator) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}

func (t *TriclassCalculator) countNonZeroPixels(mat *safe.Mat) int {
	rows := mat.Rows()
	cols := mat.Cols()
	count := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := mat.GetUCharAt(y, x); err == nil && value > 0 {
				count++
			}
		}
	}

	return count
}

func (t *TriclassCalculator) calculateHistogram(src *safe.Mat, histBins int) []int {
	histogram := make([]int, histBins)
	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if pixelValue, err := src.GetUCharAt(y, x); err == nil && pixelValue > 0 {
				bin := int(float64(pixelValue) * binScale)
				bin = max(0, min(bin, histBins-1))
				histogram[bin]++
			}
		}
	}

	return histogram
}

func (t *TriclassCalculator) updateResult(result, foregroundMask *safe.Mat) {
	rows := result.Rows()
	cols := result.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := foregroundMask.GetUCharAt(y, x); err == nil && value > 0 {
				result.SetUCharAt(y, x, 255)
			}
		}
	}
}

func (t *TriclassCalculator) extractTBDRegion(original, tbdMask *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(original.Rows(), original.Cols(), original.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create TBD region Mat: %w", err)
	}

	rows := original.Rows()
	cols := original.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdValue, err := tbdMask.GetUCharAt(y, x); err == nil && tbdValue > 0 {
				if origValue, err := original.GetUCharAt(y, x); err == nil {
					result.SetUCharAt(y, x, origValue)
				}
			}
		}
	}

	return result, nil
}

func (t *TriclassCalculator) calculateAdaptiveHistogramBins(region *safe.Mat) int {
	rows := region.Rows()
	cols := region.Cols()
	totalPixels := rows * cols

	// Calculate dynamic range and noise level
	var minVal, maxVal uint8 = 255, 0
	nonZeroPixels := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val, _ := region.GetUCharAt(y, x)
			if val > 0 {
				nonZeroPixels++
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
			}
		}
	}

	if nonZeroPixels == 0 {
		return 32
	}

	dynamicRange := int(maxVal - minVal)
	noiseLevel := t.estimateNoiseLevel(region)

	// Adaptive calculation
	baseBins := 32
	if dynamicRange < 20 {
		baseBins = 16
	} else if dynamicRange > 100 {
		baseBins = 64
	}

	// Adjust for noise level
	if noiseLevel > 10.0 {
		baseBins = max(baseBins/2, 8)
	}

	// Adjust for region size
	if nonZeroPixels < totalPixels/4 {
		baseBins = max(baseBins/2, 8)
	}

	return baseBins
}

func (t *TriclassCalculator) estimateNoiseLevel(src *safe.Mat) float64 {
	rows := src.Rows()
	cols := src.Cols()

	// Use Laplacian operator to estimate noise
	var sumSq float64
	count := 0

	for y := 1; y < rows-1; y++ {
		for x := 1; x < cols-1; x++ {
			center, err := src.GetUCharAt(y, x)
			if err != nil || center == 0 {
				continue
			}

			top, _ := src.GetUCharAt(y-1, x)
			bottom, _ := src.GetUCharAt(y+1, x)
			left, _ := src.GetUCharAt(y, x-1)
			right, _ := src.GetUCharAt(y, x+1)

			laplacian := float64(center)*4 - float64(top) - float64(bottom) - float64(left) - float64(right)
			sumSq += laplacian * laplacian
			count++
		}
	}

	if count > 0 {
		return math.Sqrt(sumSq/float64(count)) / 6.0
	}
	return 5.0 // Default noise level
}

func (t *TriclassCalculator) isHistogramBimodal(histogram []int) bool {
	histBins := len(histogram)

	// Smooth histogram to reduce noise in peak detection
	smoothed := make([]float64, histBins)
	for i := 0; i < histBins; i++ {
		sum := 0.0
		count := 0
		for j := max(0, i-2); j <= min(histBins-1, i+2); j++ {
			sum += float64(histogram[j])
			count++
		}
		smoothed[i] = sum / float64(count)
	}

	// Find local maxima
	peaks := 0
	for i := 1; i < histBins-1; i++ {
		if smoothed[i] > smoothed[i-1] && smoothed[i] > smoothed[i+1] && smoothed[i] > 0 {
			peaks++
		}
	}

	return peaks >= 2
}