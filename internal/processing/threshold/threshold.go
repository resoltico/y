package threshold

import (
	"context"
	"fmt"
	"math"

	"otsu-obliterator/internal/opencv/safe"
)

// Otsu2DCalculator implements 2D Otsu thresholding
type Otsu2DCalculator struct{}

func NewOtsu2DCalculator() *Otsu2DCalculator {
	return &Otsu2DCalculator{}
}

func (o *Otsu2DCalculator) Calculate(histogram [][]float64) ([2]float64, error) {
	if len(histogram) == 0 || len(histogram[0]) == 0 {
		return [2]float64{}, fmt.Errorf("empty histogram")
	}

	return o.find2DOtsuThresholdDecomposed(histogram), nil
}

func (o *Otsu2DCalculator) find2DOtsuThresholdDecomposed(histogram [][]float64) [2]float64 {
	histBins := len(histogram)
	bestThreshold := [2]float64{float64(histBins) / 2.0, float64(histBins) / 2.0}
	maxVariance := 0.0

	// Calculate total statistics with double precision
	totalSum := 0.0
	totalCount := 0.0
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			weight := histogram[i][j]
			totalSum += float64(i*histBins+j) * weight
			totalCount += weight
		}
	}

	if totalCount < 1e-10 {
		return bestThreshold
	}

	invTotalCount := 1.0 / totalCount
	subPixelStep := 0.1

	// Search with sub-pixel precision using bilinear interpolation
	for t1 := 1.0; t1 < float64(histBins-1); t1 += subPixelStep {
		for t2 := 1.0; t2 < float64(histBins-1); t2 += subPixelStep {
			variance := o.calculateBetweenClassVariance(histogram, t1, t2, totalSum, totalCount, invTotalCount)
			if variance > maxVariance {
				maxVariance = variance
				bestThreshold = [2]float64{t1, t2}
			}
		}
	}

	return bestThreshold
}

func (o *Otsu2DCalculator) calculateBetweenClassVariance(histogram [][]float64, t1, t2, totalSum, totalCount, invTotalCount float64) float64 {
	histBins := len(histogram)
	var w0, w1, sum0, sum1 float64

	t1Int := int(t1)
	t2Int := int(t2)

	// Calculate class 0 statistics with bilinear interpolation
	for i := 0; i <= t1Int && i < histBins; i++ {
		for j := 0; j <= t2Int && j < histBins; j++ {
			if float64(i) <= t1 && float64(j) <= t2 {
				weight := histogram[i][j]

				// Apply bilinear interpolation for sub-pixel precision
				interpFactor := 1.0
				if i == t1Int && float64(i) < t1 {
					interpFactor *= (t1 - float64(i))
				}
				if j == t2Int && float64(j) < t2 {
					interpFactor *= (t2 - float64(j))
				}

				weightInterp := weight * interpFactor
				w0 += weightInterp
				sum0 += float64(i*histBins+j) * weightInterp
			}
		}
	}

	// Calculate class 1 statistics
	for i := t1Int + 1; i < histBins; i++ {
		for j := t2Int + 1; j < histBins; j++ {
			weight := histogram[i][j]
			w1 += weight
			sum1 += float64(i*histBins+j) * weight
		}
	}

	// Check for numerical stability
	if w0 < 1e-10 || w1 < 1e-10 {
		return 0.0
	}

	mean0 := sum0 / w0
	mean1 := sum1 / w1
	meanDiff := mean0 - mean1

	// Normalize weights
	w0 *= invTotalCount
	w1 *= invTotalCount

	return w0 * w1 * meanDiff * meanDiff
}

// BilinearApplier applies threshold using bilinear interpolation
type BilinearApplier struct{}

func NewBilinearApplier() *BilinearApplier {
	return &BilinearApplier{}
}

func (b *BilinearApplier) Apply(src, neighborhood *safe.Mat, thresholds [2]float64) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	err = b.applyThresholdBilinear(src, neighborhood, result, thresholds)
	if err != nil {
		result.Close()
		return nil, err
	}

	return result, nil
}

func (b *BilinearApplier) applyThresholdBilinear(src, neighborhood, dst *safe.Mat, threshold [2]float64) error {
	rows := src.Rows()
	cols := src.Cols()
	histBins := 256 // Standard 8-bit range
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := src.GetUCharAt(y, x)
			if err != nil {
				return err
			}

			neighValue, err := neighborhood.GetUCharAt(y, x)
			if err != nil {
				return err
			}

			pixelBin := float64(pixelValue) * binScale
			neighBin := float64(neighValue) * binScale

			// Use bilinear interpolation for sub-pixel threshold comparison
			var value uint8
			if pixelBin > threshold[0] && neighBin > threshold[1] {
				value = 255
			} else {
				value = 0
			}

			if err := dst.SetUCharAt(y, x, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// TriclassCalculator implements iterative triclass thresholding
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
	histogram := t.calculateHistogram(region)

	switch method {
	case "mean":
		return t.calculateMeanThreshold(histogram)
	case "median":
		return t.calculateMedianThreshold(histogram)
	case "triangle":
		return t.calculateTriangleThreshold(histogram)
	default:
		return t.calculateOtsuThreshold(histogram)
	}
}

func (t *TriclassCalculator) calculateHistogram(src *safe.Mat) []int {
	histogram := make([]int, 256)
	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if pixelValue, err := src.GetUCharAt(y, x); err == nil && pixelValue > 0 {
				histogram[pixelValue]++
			}
		}
	}

	return histogram
}

func (t *TriclassCalculator) calculateOtsuThreshold(histogram []int) float64 {
	total := 0
	for _, count := range histogram {
		total += count
	}

	if total == 0 {
		return 127.5
	}

	sum := 0.0
	for i, count := range histogram {
		sum += float64(i) * float64(count)
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5

	for i := 0; i < 256; i++ {
		wB += histogram[i]
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += float64(i) * float64(histogram[i])
		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)

		varBetween := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = float64(i)
		}
	}

	return bestThreshold
}

func (t *TriclassCalculator) calculateMeanThreshold(histogram []int) float64 {
	totalPixels := 0
	weightedSum := 0.0

	for i, count := range histogram {
		totalPixels += count
		weightedSum += float64(i) * float64(count)
	}

	if totalPixels == 0 {
		return 127.5
	}

	return weightedSum / float64(totalPixels)
}

func (t *TriclassCalculator) calculateMedianThreshold(histogram []int) float64 {
	totalPixels := 0
	for _, count := range histogram {
		totalPixels += count
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := totalPixels / 2
	cumSum := 0

	for i, count := range histogram {
		cumSum += count
		if cumSum >= halfPixels {
			return float64(i)
		}
	}

	return 127.5
}

func (t *TriclassCalculator) calculateTriangleThreshold(histogram []int) float64 {
	// Find histogram peak
	maxCount := 0
	peakIndex := 0
	for i, count := range histogram {
		if count > maxCount {
			maxCount = count
			peakIndex = i
		}
	}

	// Find histogram endpoints
	leftEnd := 0
	rightEnd := 255

	for i := 0; i < 256; i++ {
		if histogram[i] > 0 {
			leftEnd = i
			break
		}
	}

	for i := 255; i >= 0; i-- {
		if histogram[i] > 0 {
			rightEnd = i
			break
		}
	}

	// Triangle method: find point with maximum distance to line
	maxDistance := 0.0
	bestThreshold := float64(peakIndex)

	// Determine which end to use
	farEnd := rightEnd
	if peakIndex-leftEnd > rightEnd-peakIndex {
		farEnd = leftEnd
	}

	// Calculate line parameters
	x1, y1 := float64(peakIndex), float64(maxCount)
	x2, y2 := float64(farEnd), float64(histogram[farEnd])

	if x1 != x2 {
		for i := min(peakIndex, farEnd); i <= max(peakIndex, farEnd); i++ {
			distance := math.Abs((y2-y1)*float64(i)-(x2-x1)*float64(histogram[i])+x2*y1-y2*x1) /
				math.Sqrt((y2-y1)*(y2-y1)+(x2-x1)*(x2-x1))

			if distance > maxDistance {
				maxDistance = distance
				bestThreshold = float64(i)
			}
		}
	}

	return bestThreshold
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
	lowerThreshold := threshold * (1.0 - classSeparation)
	upperThreshold := threshold * (1.0 + classSeparation)

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}