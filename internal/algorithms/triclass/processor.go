package triclass

import (
	"context"
	"fmt"
	"image"
	"math"

	"otsu-obliterator/internal/opencv/conversion"
	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

type Processor struct {
	name string
}

func NewProcessor() *Processor {
	return &Processor{
		name: "Iterative Triclass",
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"initial_threshold_method": "otsu", // Initial threshold calculation method
		"histogram_bins":           0,      // Auto-calculated based on region characteristics
		"convergence_precision":    1.0,    // Convergence threshold (0.5-2.0)
		"max_iterations":           8,      // Maximum iterations (5-15)
		"minimum_tbd_fraction":     0.01,   // Minimum "to be determined" fraction
		"class_separation":         0.5,    // Adaptive gap factor
		"preprocessing":            true,   // Advanced preprocessing with guided filtering
		"result_cleanup":           true,   // Morphological cleanup
		"preserve_borders":         false,  // Border preservation
		"noise_robustness":         true,   // Non-local means denoising
		"guided_filtering":         true,   // Edge-preserving guided filter
		"guided_radius":            6,      // Guided filter radius (1-8)
		"guided_epsilon":           0.15,   // Guided filter regularization
		"parallel_processing":      true,   // Use OpenCV parallel processing
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if method, ok := params["initial_threshold_method"].(string); ok {
		if method != "otsu" && method != "mean" && method != "median" && method != "triangle" {
			return fmt.Errorf("initial_threshold_method must be 'otsu', 'mean', 'median', or 'triangle', got: %s", method)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins != 0 && (histBins < 8 || histBins > 256) {
			return fmt.Errorf("histogram_bins must be 0 (auto) or between 8 and 256, got: %d", histBins)
		}
	}

	if precision, ok := params["convergence_precision"].(float64); ok {
		if precision < 0.5 || precision > 2.0 {
			return fmt.Errorf("convergence_precision must be between 0.5 and 2.0, got: %f", precision)
		}
	}

	if maxIter, ok := params["max_iterations"].(int); ok {
		if maxIter < 3 || maxIter > 15 {
			return fmt.Errorf("max_iterations must be between 3 and 15, got: %d", maxIter)
		}
	}

	if fraction, ok := params["minimum_tbd_fraction"].(float64); ok {
		if fraction < 0.001 || fraction > 0.2 {
			return fmt.Errorf("minimum_tbd_fraction must be between 0.001 and 0.2, got: %f", fraction)
		}
	}

	if separation, ok := params["class_separation"].(float64); ok {
		if separation < 0.1 || separation > 0.8 {
			return fmt.Errorf("class_separation must be between 0.1 and 0.8, got: %f", separation)
		}
	}

	if radius, ok := params["guided_radius"].(int); ok {
		if radius < 1 || radius > 8 {
			return fmt.Errorf("guided_radius must be between 1 and 8, got: %d", radius)
		}
	}

	if epsilon, ok := params["guided_epsilon"].(float64); ok {
		if epsilon < 0.01 || epsilon > 0.5 {
			return fmt.Errorf("guided_epsilon must be between 0.01 and 0.5, got: %f", epsilon)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return p.ProcessWithContext(context.Background(), input, params)
}

func (p *Processor) ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(input, "Iterative Triclass processing"); err != nil {
		return nil, err
	}

	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Enable parallel processing if requested
	if p.getBoolParam(params, "parallel_processing") {
		gocv.SetNumThreads(0) // Use all available threads
	} else {
		gocv.SetNumThreads(1)
	}

	gray, err := conversion.ConvertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to grayscale: %w", err)
	}
	defer gray.Close()

	working := gray
	if p.getBoolParam(params, "preprocessing") {
		preprocessed, err := p.applyAdvancedPreprocessing(gray, params)
		if err != nil {
			return nil, fmt.Errorf("preprocessing failed: %w", err)
		}
		working = preprocessed
		defer preprocessed.Close()
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result, err := p.performIterativeTriclassAdaptive(ctx, working, params)
	if err != nil {
		return nil, fmt.Errorf("iterative processing failed: %w", err)
	}

	if p.getBoolParam(params, "result_cleanup") {
		cleaned, err := p.applyAdvancedCleanup(result)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("cleanup failed: %w", err)
		}
		result.Close()
		result = cleaned
	}

	return result, nil
}

// performIterativeTriclassAdaptive implements iterative triclass with adaptive convergence and gap calculation
func (p *Processor) performIterativeTriclassAdaptive(ctx context.Context, working *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	maxIterations := p.getIntParam(params, "max_iterations")
	convergencePrecision := p.getFloatParam(params, "convergence_precision")
	minTBDFraction := p.getFloatParam(params, "minimum_tbd_fraction")

	result, err := safe.NewMat(working.Rows(), working.Cols(), working.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	currentRegion, err := working.Clone()
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("failed to clone working Mat: %w", err)
	}
	defer currentRegion.Close()

	previousThreshold := -1.0
	totalPixels := float64(currentRegion.Rows() * currentRegion.Cols())
	convergenceHistory := make([]float64, 0, maxIterations)

	// Stability tracking for convergence detection
	stableIterations := 0
	const requiredStableIterations = 2

	for iteration := 0; iteration < maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		nonZeroPixels := p.countNonZeroPixels(currentRegion)
		if nonZeroPixels == 0 {
			break
		}

		threshold := p.calculateThresholdAdaptive(currentRegion, params)

		// Convergence detection with stability requirement
		convergence := math.Abs(threshold - previousThreshold)
		convergenceHistory = append(convergenceHistory, convergence)

		if convergence < convergencePrecision {
			stableIterations++
			if stableIterations >= requiredStableIterations {
				break
			}
		} else {
			stableIterations = 0
		}

		previousThreshold = threshold

		foregroundMask, backgroundMask, tbdMask, err := p.segmentRegionAdaptiveGaps(currentRegion, threshold, params)
		if err != nil {
			return nil, fmt.Errorf("segmentation failed at iteration %d: %w", iteration, err)
		}

		tbdCount := p.countNonZeroPixels(tbdMask)
		tbdFraction := float64(tbdCount) / totalPixels

		p.updateResult(result, foregroundMask)

		foregroundMask.Close()
		backgroundMask.Close()

		if tbdFraction < minTBDFraction {
			tbdMask.Close()
			break
		}

		newRegion, err := p.extractTBDRegion(working, tbdMask)
		tbdMask.Close()
		if err != nil {
			return nil, fmt.Errorf("TBD region extraction failed: %w", err)
		}

		currentRegion.Close()
		currentRegion = newRegion
	}

	return result, nil
}

// applyAdvancedPreprocessing applies guided filtering and non-local means denoising
func (p *Processor) applyAdvancedPreprocessing(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	var processed *safe.Mat
	var err error

	// Apply guided filtering for edge-preserving smoothing
	if p.getBoolParam(params, "guided_filtering") {
		guided, err := p.applyGuidedFilter(src, params)
		if err != nil {
			return nil, err
		}
		processed = guided
	} else {
		processed, err = src.Clone()
		if err != nil {
			return nil, err
		}
	}

	// Apply non-local means denoising if noise robustness is enabled
	if p.getBoolParam(params, "noise_robustness") {
		denoised, err := p.applyNonLocalMeansDenoising(processed)
		if err != nil {
			processed.Close()
			return nil, err
		}
		processed.Close()
		return denoised, nil
	}

	return processed, nil
}

// applyGuidedFilter implements edge-preserving guided filtering with optimized parameters
func (p *Processor) applyGuidedFilter(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	radius := p.getIntParam(params, "guided_radius")
	epsilon := p.getFloatParam(params, "guided_epsilon")

	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()

	// Build integral images for efficient box filtering
	integralI, integralI2, integralP, integralIP := p.buildGuidedFilterIntegrals(src)
	defer integralI.Close()
	defer integralI2.Close()
	defer integralP.Close()
	defer integralIP.Close()

	// Apply guided filter with box filter approximation
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-radius)
			x1 := max(0, x-radius)
			y2 := min(rows-1, y+radius)
			x2 := min(cols-1, x+radius)

			area := float64((y2 - y1 + 1) * (x2 - x1 + 1))

			// Calculate local statistics using integral images
			meanI := p.getIntegralSum(integralI, y1, x1, y2, x2) / area
			meanI2 := p.getIntegralSum(integralI2, y1, x1, y2, x2) / area
			meanP := p.getIntegralSum(integralP, y1, x1, y2, x2) / area
			meanIP := p.getIntegralSum(integralIP, y1, x1, y2, x2) / area

			varI := meanI2 - meanI*meanI
			covIP := meanIP - meanI*meanP

			a := covIP / (varI + epsilon)
			b := meanP - a*meanI

			pixelVal, _ := src.GetUCharAt(y, x)
			filteredVal := a*float64(pixelVal) + b

			result.SetUCharAt(y, x, uint8(math.Max(0, math.Min(255, filteredVal))))
		}
	}

	return result, nil
}

// applyNonLocalMeansDenoising applies advanced denoising with adaptive parameters
func (p *Processor) applyNonLocalMeansDenoising(src *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()

	// Apply non-local means denoising with adaptive parameters
	// h=10 for moderate denoising, templateWindowSize=7, searchWindowSize=21
	gocv.FastNlMeansDenoisingWithParams(srcMat, &resultMat, 10.0, 7, 21)

	return result, nil
}

// calculateThresholdAdaptive uses automatic method selection with histogram analysis
func (p *Processor) calculateThresholdAdaptive(region *safe.Mat, params map[string]interface{}) float64 {
	method := p.getStringParam(params, "initial_threshold_method")
	histBins := p.getIntParam(params, "histogram_bins")

	if histBins == 0 {
		histBins = p.calculateAdaptiveHistogramBins(region)
	}

	histogram := p.calculateHistogram(region, histBins)

	// Detect histogram characteristics for method selection
	if method == "otsu" && !p.isHistogramBimodal(histogram) {
		// Fall back to triangle method for skewed unimodal histograms
		method = "triangle"
	}

	switch method {
	case "mean":
		return p.calculateMeanThresholdPrecise(histogram, histBins)
	case "median":
		return p.calculateMedianThresholdPrecise(histogram, histBins)
	case "triangle":
		return p.calculateTriangleThreshold(histogram, histBins)
	default:
		return p.calculateOtsuThresholdStable(histogram, histBins)
	}
}

// calculateAdaptiveHistogramBins determines optimal bin count based on region characteristics
func (p *Processor) calculateAdaptiveHistogramBins(region *safe.Mat) int {
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
	noiseLevel := p.estimateNoiseLevel(region)

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

// segmentRegionAdaptiveGaps uses adaptive gap calculation based on image characteristics
func (p *Processor) segmentRegionAdaptiveGaps(region *safe.Mat, threshold float64, params map[string]interface{}) (*safe.Mat, *safe.Mat, *safe.Mat, error) {
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

	classSeparation := p.getFloatParam(params, "class_separation")

	// Adaptive gap calculation based on threshold position and local variance
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

// calculateOtsuThresholdStable implements stable Otsu calculation with numerical checks
func (p *Processor) calculateOtsuThresholdStable(histogram []int, histBins int) float64 {
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

// calculateTriangleThreshold implements triangle thresholding for skewed histograms
func (p *Processor) calculateTriangleThreshold(histogram []int, histBins int) float64 {
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

// calculateMeanThresholdPrecise calculates mean with sub-pixel interpolation
func (p *Processor) calculateMeanThresholdPrecise(histogram []int, histBins int) float64 {
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

// calculateMedianThresholdPrecise calculates median with sub-pixel interpolation
func (p *Processor) calculateMedianThresholdPrecise(histogram []int, histBins int) float64 {
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

// isHistogramBimodal detects if histogram is bimodal for Otsu applicability
func (p *Processor) isHistogramBimodal(histogram []int) bool {
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

// applyAdvancedCleanup applies morphological operations with adaptive kernels
func (p *Processor) applyAdvancedCleanup(src *safe.Mat) (*safe.Mat, error) {
	// Adaptive kernel size based on image dimensions
	rows := src.Rows()
	cols := src.Cols()

	smallKernelSize := 3
	largeKernelSize := 5

	// Adjust kernel sizes for large images
	if rows*cols > 1000000 {
		smallKernelSize = 5
		largeKernelSize = 7
	}

	kernel3 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: smallKernelSize, Y: smallKernelSize})
	defer kernel3.Close()

	kernel5 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: largeKernelSize, Y: largeKernelSize})
	defer kernel5.Close()

	// Opening operation to remove small noise
	opened, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	srcMat := src.GetMat()
	openedMat := opened.GetMat()
	gocv.MorphologyEx(srcMat, &openedMat, gocv.MorphOpen, kernel3)

	// Closing operation to fill small gaps
	closed, err := safe.NewMat(opened.Rows(), opened.Cols(), opened.Type())
	if err != nil {
		return nil, err
	}
	defer closed.Close()

	closedMat := closed.GetMat()
	gocv.MorphologyEx(openedMat, &closedMat, gocv.MorphClose, kernel5)

	// Final median filtering for additional noise reduction
	result, err := safe.NewMat(closed.Rows(), closed.Cols(), closed.Type())
	if err != nil {
		return nil, err
	}

	resultMat := result.GetMat()
	gocv.MedianBlur(closedMat, &resultMat, smallKernelSize)

	return result, nil
}

// Helper functions

func (p *Processor) buildGuidedFilterIntegrals(src *safe.Mat) (*safe.Mat, *safe.Mat, *safe.Mat, *safe.Mat) {
	rows := src.Rows()
	cols := src.Cols()

	integralI, _ := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	integralI2, _ := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	integralP, _ := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	integralIP, _ := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)

	// Initialize borders
	for i := 0; i <= rows; i++ {
		integralI.GetMat().SetDoubleAt(i, 0, 0.0)
		integralI2.GetMat().SetDoubleAt(i, 0, 0.0)
		integralP.GetMat().SetDoubleAt(i, 0, 0.0)
		integralIP.GetMat().SetDoubleAt(i, 0, 0.0)
	}
	for j := 0; j <= cols; j++ {
		integralI.GetMat().SetDoubleAt(0, j, 0.0)
		integralI2.GetMat().SetDoubleAt(0, j, 0.0)
		integralP.GetMat().SetDoubleAt(0, j, 0.0)
		integralIP.GetMat().SetDoubleAt(0, j, 0.0)
	}

	// Build integral images
	for y := 1; y <= rows; y++ {
		for x := 1; x <= cols; x++ {
			pixelVal, _ := src.GetUCharAt(y-1, x-1)
			I := float64(pixelVal)
			P := I // For guided filter, guide image equals input image

			// Calculate integral sums
			prevRowI := integralI.GetMat().GetDoubleAt(y-1, x)
			prevColI := integralI.GetMat().GetDoubleAt(y, x-1)
			prevDiagI := integralI.GetMat().GetDoubleAt(y-1, x-1)
			integralI.GetMat().SetDoubleAt(y, x, I+prevRowI+prevColI-prevDiagI)

			prevRowI2 := integralI2.GetMat().GetDoubleAt(y-1, x)
			prevColI2 := integralI2.GetMat().GetDoubleAt(y, x-1)
			prevDiagI2 := integralI2.GetMat().GetDoubleAt(y-1, x-1)
			integralI2.GetMat().SetDoubleAt(y, x, I*I+prevRowI2+prevColI2-prevDiagI2)

			prevRowP := integralP.GetMat().GetDoubleAt(y-1, x)
			prevColP := integralP.GetMat().GetDoubleAt(y, x-1)
			prevDiagP := integralP.GetMat().GetDoubleAt(y-1, x-1)
			integralP.GetMat().SetDoubleAt(y, x, P+prevRowP+prevColP-prevDiagP)

			prevRowIP := integralIP.GetMat().GetDoubleAt(y-1, x)
			prevColIP := integralIP.GetMat().GetDoubleAt(y, x-1)
			prevDiagIP := integralIP.GetMat().GetDoubleAt(y-1, x-1)
			integralIP.GetMat().SetDoubleAt(y, x, I*P+prevRowIP+prevColIP-prevDiagIP)
		}
	}

	return integralI, integralI2, integralP, integralIP
}

func (p *Processor) getIntegralSum(integral *safe.Mat, y1, x1, y2, x2 int) float64 {
	sum := integral.GetMat().GetDoubleAt(y2+1, x2+1)
	sum -= integral.GetMat().GetDoubleAt(y1, x2+1)
	sum -= integral.GetMat().GetDoubleAt(y2+1, x1)
	sum += integral.GetMat().GetDoubleAt(y1, x1)
	return sum
}

func (p *Processor) estimateNoiseLevel(src *safe.Mat) float64 {
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

func (p *Processor) countNonZeroPixels(mat *safe.Mat) int {
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

func (p *Processor) calculateHistogram(src *safe.Mat, histBins int) []int {
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

func (p *Processor) updateResult(result, foregroundMask *safe.Mat) {
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

func (p *Processor) extractTBDRegion(original, tbdMask *safe.Mat) (*safe.Mat, error) {
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

func (p *Processor) getBoolParam(params map[string]interface{}, key string) bool {
	if value, ok := params[key].(bool); ok {
		return value
	}
	return false
}

func (p *Processor) getIntParam(params map[string]interface{}, key string) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return 0
}

func (p *Processor) getFloatParam(params map[string]interface{}, key string) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return 0.0
}

func (p *Processor) getStringParam(params map[string]interface{}, key string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return ""
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
