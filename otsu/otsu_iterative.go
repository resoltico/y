package otsu

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

type IterativeTriclassProcessor struct {
	params       map[string]interface{}
	debugManager *debug.Manager
}

func NewIterativeTriclassProcessor(params map[string]interface{}) *IterativeTriclassProcessor {
	return &IterativeTriclassProcessor{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

func (processor *IterativeTriclassProcessor) Process(src gocv.Mat) (gocv.Mat, error) {
	// Validate input Mat thoroughly
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	if src.Rows() <= 0 || src.Cols() <= 0 {
		return gocv.NewMat(), fmt.Errorf("input Mat has invalid dimensions: %dx%d", src.Cols(), src.Rows())
	}

	// Create a safe working copy immediately to avoid memory issues
	safeCopy := src.Clone()
	defer safeCopy.Close()

	if safeCopy.Empty() {
		return gocv.NewMat(), fmt.Errorf("failed to create safe copy of input Mat")
	}

	// Debug initial input using safe copy
	processor.debugManager.LogTriclassStart(safeCopy, processor.params)
	processor.debugManager.LogMatPixelAnalysis("TriclassInput", safeCopy)

	// Convert to grayscale if needed
	gray := gocv.NewMat()
	defer gray.Close()

	if safeCopy.Channels() == 3 {
		gocv.CvtColor(safeCopy, &gray, gocv.ColorBGRToGray)
	} else {
		safeCopy.CopyTo(&gray)
	}

	if gray.Empty() {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed")
	}

	processor.debugManager.LogMatPixelAnalysis("TriclassGrayscale", gray)

	// Apply preprocessing if requested
	working := gocv.NewMat()
	defer working.Close()

	if processor.getBoolParam("apply_preprocessing") {
		processor.applyPreprocessing(&gray, &working)
		if working.Empty() {
			return gocv.NewMat(), fmt.Errorf("preprocessing failed")
		}
		processor.debugManager.LogMatPixelAnalysis("TriclassPreprocessed", working)
	} else {
		gray.CopyTo(&working)
	}

	// Iterative triclass processing
	result, err := processor.performIterativeTriclass(&working)
	if err != nil {
		return gocv.NewMat(), err
	}

	// Apply cleanup if requested
	if processor.getBoolParam("apply_cleanup") {
		cleaned := gocv.NewMat()
		defer result.Close()
		processor.applyCleanup(&result, &cleaned)
		if cleaned.Empty() {
			return gocv.NewMat(), fmt.Errorf("cleanup failed")
		}
		processor.debugManager.LogMatPixelAnalysis("TriclassCleanedResult", cleaned)
		return cleaned, nil
	}

	processor.debugManager.LogMatPixelAnalysis("TriclassFinalResult", result)
	return result, nil
}

func (processor *IterativeTriclassProcessor) performIterativeTriclass(working *gocv.Mat) (gocv.Mat, error) {
	maxIterations := processor.getIntParam("max_iterations")
	convergenceEpsilon := processor.getFloatParam("convergence_epsilon")
	minTBDFraction := processor.getFloatParam("minimum_tbd_fraction")

	// Initialize final result
	result := gocv.NewMatWithSize(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	result.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Start with all background

	// Current working region
	currentRegion := gocv.NewMat()
	defer currentRegion.Close()
	working.CopyTo(&currentRegion)

	var previousThreshold float64 = -1
	var iterationThresholds []float64
	var iterationConvergence []float64

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Check if current region has any pixels to process
		nonZeroPixels := gocv.CountNonZero(currentRegion)
		if nonZeroPixels == 0 {
			processor.debugManager.LogTriclassIteration(iteration, previousThreshold, 0, 0, 0, 0)
			break
		}

		// Calculate threshold for current region
		threshold := processor.calculateThresholdForRegion(&currentRegion)
		iterationThresholds = append(iterationThresholds, threshold)

		// Check convergence
		convergence := math.Abs(threshold - previousThreshold)
		iterationConvergence = append(iterationConvergence, convergence)

		if previousThreshold >= 0 && convergence < convergenceEpsilon {
			processor.debugManager.LogTriclassIteration(iteration, threshold, convergence, 0, 0, 0)
			break
		}
		previousThreshold = threshold

		// Segment current region into three classes
		foregroundMask, backgroundMask, tbdMask := processor.segmentRegion(&currentRegion, threshold)

		// Count pixels in each class
		foregroundCount := gocv.CountNonZero(foregroundMask)
		backgroundCount := gocv.CountNonZero(backgroundMask)
		tbdCount := gocv.CountNonZero(tbdMask)

		processor.debugManager.LogTriclassIteration(iteration, threshold, convergence,
			foregroundCount, backgroundCount, tbdCount)

		// Update final result with current classifications
		processor.updateResult(&result, &foregroundMask, &backgroundMask)

		// Check if TBD region is too small
		totalPixels := currentRegion.Rows() * currentRegion.Cols()
		tbdFraction := float64(tbdCount) / float64(totalPixels)

		if tbdFraction < minTBDFraction {
			foregroundMask.Close()
			backgroundMask.Close()
			tbdMask.Close()
			break
		}

		// Update current region to only include TBD pixels
		newRegion := gocv.NewMat()
		processor.extractTBDRegion(working, &tbdMask, &newRegion)
		currentRegion.Close()
		currentRegion = newRegion

		foregroundMask.Close()
		backgroundMask.Close()
		tbdMask.Close()
	}

	// Log debug information
	totalPixels := result.Rows() * result.Cols()
	foregroundPixels := gocv.CountNonZero(result)
	backgroundPixels := totalPixels - foregroundPixels

	debugInfo := &debug.TriclassDebugInfo{
		InputMatDimensions:   fmt.Sprintf("%dx%d", working.Cols(), working.Rows()),
		InputMatChannels:     working.Channels(),
		InputMatType:         working.Type(),
		OutputMatDimensions:  fmt.Sprintf("%dx%d", result.Cols(), result.Rows()),
		OutputMatChannels:    result.Channels(),
		OutputMatType:        result.Type(),
		IterationCount:       len(iterationThresholds),
		FinalThreshold:       previousThreshold,
		TotalPixels:          totalPixels,
		ForegroundPixels:     foregroundPixels,
		BackgroundPixels:     backgroundPixels,
		TBDPixels:            0, // Final result has no TBD pixels
		ProcessingSteps:      []string{"grayscale", "preprocessing", "iterative_segmentation", "final_result"},
		IterationThresholds:  iterationThresholds,
		IterationConvergence: iterationConvergence,
	}

	processor.debugManager.LogTriclassResult(debugInfo)

	return result, nil
}

func (processor *IterativeTriclassProcessor) calculateThresholdForRegion(region *gocv.Mat) float64 {
	// Build histogram for the region
	histogram := processor.calculateHistogram(region)

	method := processor.getStringParam("initial_threshold_method")
	switch method {
	case "mean":
		return processor.calculateMeanThreshold(histogram)
	case "median":
		return processor.calculateMedianThreshold(histogram)
	default: // "otsu"
		return processor.calculateOtsuThreshold(histogram)
	}
}

func (processor *IterativeTriclassProcessor) calculateHistogram(src *gocv.Mat) []int {
	histBins := processor.getIntParam("histogram_bins")
	histogram := make([]int, histBins)

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := src.GetUCharAt(y, x)

			// Only include non-zero pixels (active region)
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

	return histogram
}

func (processor *IterativeTriclassProcessor) calculateOtsuThreshold(histogram []int) float64 {
	histBins := len(histogram)
	total := 0

	// Calculate total pixel count
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return 127.5 // Default middle value
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

		// Between-class variance
		varBetween := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = float64(t) * 255.0 / float64(histBins-1)
		}
	}

	return bestThreshold
}

func (processor *IterativeTriclassProcessor) calculateMeanThreshold(histogram []int) float64 {
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

func (processor *IterativeTriclassProcessor) calculateMedianThreshold(histogram []int) float64 {
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

func (processor *IterativeTriclassProcessor) segmentRegion(region *gocv.Mat, threshold float64) (gocv.Mat, gocv.Mat, gocv.Mat) {
	rows := region.Rows()
	cols := region.Cols()

	foreground := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	background := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	tbd := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	// Initialize all masks to 0
	foreground.SetTo(gocv.NewScalar(0, 0, 0, 0))
	background.SetTo(gocv.NewScalar(0, 0, 0, 0))
	tbd.SetTo(gocv.NewScalar(0, 0, 0, 0))

	gapFactor := processor.getFloatParam("lower_upper_gap_factor")

	// Create adaptive thresholds
	lowerThreshold := threshold * (1.0 - gapFactor)
	upperThreshold := threshold * (1.0 + gapFactor)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := float64(region.GetUCharAt(y, x))

			// Only process active pixels
			if pixelValue > 0 {
				if pixelValue > upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if pixelValue < lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd
}

func (processor *IterativeTriclassProcessor) updateResult(result, foregroundMask, backgroundMask *gocv.Mat) {
	rows := result.Rows()
	cols := result.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if foregroundMask.GetUCharAt(y, x) > 0 {
				result.SetUCharAt(y, x, 255)
			}
			// Background pixels remain 0 (already initialized)
		}
	}
}

func (processor *IterativeTriclassProcessor) extractTBDRegion(original, tbdMask, result *gocv.Mat) {
	*result = gocv.NewMatWithSize(original.Rows(), original.Cols(), gocv.MatTypeCV8UC1)
	result.SetTo(gocv.NewScalar(0, 0, 0, 0))

	rows := original.Rows()
	cols := original.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdMask.GetUCharAt(y, x) > 0 {
				result.SetUCharAt(y, x, original.GetUCharAt(y, x))
			}
		}
	}
}

func (processor *IterativeTriclassProcessor) applyPreprocessing(src, dst *gocv.Mat) {
	// Apply CLAHE for contrast enhancement
	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	clahe.Apply(*src, dst)

	// Apply denoising
	denoised := gocv.NewMat()
	defer denoised.Close()

	gocv.FastNlMeansDenoising(*dst, &denoised)
	denoised.CopyTo(dst)
}

func (processor *IterativeTriclassProcessor) applyCleanup(src, dst *gocv.Mat) {
	// Apply morphological operations to clean up the result
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	// Opening to remove small noise
	opened := gocv.NewMat()
	defer opened.Close()
	gocv.MorphologyEx(*src, &opened, gocv.MorphOpen, kernel)

	// Closing to fill small holes
	gocv.MorphologyEx(opened, dst, gocv.MorphClose, kernel)

	// Apply median filter to smooth boundaries
	medianFiltered := gocv.NewMat()
	defer medianFiltered.Close()
	gocv.MedianBlur(*dst, &medianFiltered, 3)
	medianFiltered.CopyTo(dst)
}

func (processor *IterativeTriclassProcessor) getIntParam(name string) int {
	if value, ok := processor.params[name].(int); ok {
		return value
	}
	return 0
}

func (processor *IterativeTriclassProcessor) getFloatParam(name string) float64 {
	if value, ok := processor.params[name].(float64); ok {
		return value
	}
	return 0.0
}

func (processor *IterativeTriclassProcessor) getBoolParam(name string) bool {
	if value, ok := processor.params[name].(bool); ok {
		return value
	}
	return false
}

func (processor *IterativeTriclassProcessor) getStringParam(name string) string {
	if value, ok := processor.params[name].(string); ok {
		return value
	}
	return ""
}
