package otsu

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

type IterativeTriclassProcessor struct {
	params map[string]interface{}
}

func NewIterativeTriclassProcessor(params map[string]interface{}) *IterativeTriclassProcessor {
	return &IterativeTriclassProcessor{
		params: params,
	}
}

func (processor *IterativeTriclassProcessor) Process(src gocv.Mat) (gocv.Mat, error) {
	defer src.Close()

	// Convert to grayscale if needed
	gray := gocv.NewMat()
	defer gray.Close()

	if src.Channels() == 3 {
		gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	} else {
		src.CopyTo(&gray)
	}

	// Apply preprocessing if requested
	working := gocv.NewMat()
	defer working.Close()

	if processor.getBoolParam("apply_preprocessing") {
		processor.applyPreprocessing(&gray, &working)
	} else {
		gray.CopyTo(&working)
	}

	// Initialize result masks
	foregroundMask := gocv.NewMat()
	defer foregroundMask.Close()
	backgroundMask := gocv.NewMat()
	defer backgroundMask.Close()

	foregroundMask = gocv.NewMatWithSize(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	backgroundMask = gocv.NewMatWithSize(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	foregroundMask.SetTo(gocv.NewScalar(0, 0, 0, 0))
	backgroundMask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Current region to process (initially the whole image)
	currentRegion := gocv.NewMat()
	defer currentRegion.Close()
	working.CopyTo(&currentRegion)

	// Iterative processing
	maxIterations := processor.getIntParam("max_iterations")
	convergenceEpsilon := processor.getFloatParam("convergence_epsilon")
	minTBDFraction := processor.getFloatParam("minimum_tbd_fraction")

	var previousThreshold float64 = -1

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Calculate histogram for current region
		histogram := processor.calculateHistogram(&currentRegion)

		// Find initial threshold
		threshold := processor.findInitialThreshold(histogram)

		// Check convergence
		if previousThreshold >= 0 && math.Abs(threshold-previousThreshold) < convergenceEpsilon {
			break
		}
		previousThreshold = threshold

		// Calculate class means
		meanLower, meanUpper := processor.calculateClassMeans(&currentRegion, threshold)

		// Create triclass segmentation
		newForeground, newBackground, newTBD := processor.createTriclassSegmentation(
			&currentRegion, meanLower, meanUpper)

		// Check TBD region size
		tbdPixels := processor.countNonZeroPixels(&newTBD)
		totalPixels := currentRegion.Rows() * currentRegion.Cols()
		tbdFraction := float64(tbdPixels) / float64(totalPixels)

		if tbdFraction < minTBDFraction {
			// TBD region too small, stop iteration
			processor.addToMask(&foregroundMask, &newForeground)
			processor.addToMask(&backgroundMask, &newBackground)
			break
		}

		// Add current foreground and background to final masks
		processor.addToMask(&foregroundMask, &newForeground)
		processor.addToMask(&backgroundMask, &newBackground)

		// Update current region to TBD only
		processor.maskRegion(&working, &newTBD, &currentRegion)

		newForeground.Close()
		newBackground.Close()
		newTBD.Close()
	}

	// Create final result
	result := gocv.NewMat()
	processor.createFinalResult(&foregroundMask, &backgroundMask, &result)

	// Apply cleanup if requested
	if processor.getBoolParam("apply_cleanup") {
		cleaned := gocv.NewMat()
		processor.applyCleanup(&result, &cleaned)
		result.Close()
		result = cleaned
	}

	return result, nil
}

func (processor *IterativeTriclassProcessor) calculateHistogram(src *gocv.Mat) []int {
	histBins := processor.getIntParam("histogram_bins")
	histogram := make([]int, histBins)

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := src.GetUCharAt(y, x)
			bin := int(float64(pixelValue) * float64(histBins-1) / 255.0)

			if bin < 0 {
				bin = 0
			} else if bin >= histBins {
				bin = histBins - 1
			}

			histogram[bin]++
		}
	}

	return histogram
}

func (processor *IterativeTriclassProcessor) findInitialThreshold(histogram []int) float64 {
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

func (processor *IterativeTriclassProcessor) calculateOtsuThreshold(histogram []int) float64 {
	histBins := len(histogram)
	total := 0

	// Calculate total pixel count
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return float64(histBins) / 2.0
	}

	// Calculate cumulative sum and weighted sum
	sum := 0.0
	for i := 0; i < histBins; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	wF := 0
	maxVariance := 0.0
	bestThreshold := 0.0

	for t := 0; t < histBins; t++ {
		wB += histogram[t]
		if wB == 0 {
			continue
		}

		wF = total - wB
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
			bestThreshold = float64(t)
		}
	}

	// Convert back to pixel value (0-255)
	return bestThreshold * 255.0 / float64(histBins-1)
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
		return 127.5 // Middle value
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

func (processor *IterativeTriclassProcessor) calculateClassMeans(src *gocv.Mat, threshold float64) (float64, float64) {
	rows := src.Rows()
	cols := src.Cols()

	lowerSum, lowerCount := 0.0, 0
	upperSum, upperCount := 0.0, 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := float64(src.GetUCharAt(y, x))

			if pixelValue <= threshold {
				lowerSum += pixelValue
				lowerCount++
			} else {
				upperSum += pixelValue
				upperCount++
			}
		}
	}

	meanLower := 0.0
	meanUpper := 255.0

	if lowerCount > 0 {
		meanLower = lowerSum / float64(lowerCount)
	}
	if upperCount > 0 {
		meanUpper = upperSum / float64(upperCount)
	}

	return meanLower, meanUpper
}

func (processor *IterativeTriclassProcessor) createTriclassSegmentation(src *gocv.Mat, meanLower, meanUpper float64) (gocv.Mat, gocv.Mat, gocv.Mat) {
	rows := src.Rows()
	cols := src.Cols()

	foreground := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	background := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	tbd := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	foreground.SetTo(gocv.NewScalar(0, 0, 0, 0))
	background.SetTo(gocv.NewScalar(0, 0, 0, 0))
	tbd.SetTo(gocv.NewScalar(0, 0, 0, 0))

	gapFactor := processor.getFloatParam("lower_upper_gap_factor")

	// Adjust bounds based on gap factor
	adjustedLower := meanLower * (1.0 - gapFactor)
	adjustedUpper := meanUpper * (1.0 + gapFactor)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := float64(src.GetUCharAt(y, x))

			if pixelValue > adjustedUpper {
				foreground.SetUCharAt(y, x, 255) // Foreground
			} else if pixelValue < adjustedLower {
				background.SetUCharAt(y, x, 255) // Background
			} else {
				tbd.SetUCharAt(y, x, 255) // To-be-determined
			}
		}
	}

	return foreground, background, tbd
}

func (processor *IterativeTriclassProcessor) countNonZeroPixels(src *gocv.Mat) int {
	return gocv.CountNonZero(*src)
}

func (processor *IterativeTriclassProcessor) addToMask(dest, src *gocv.Mat) {
	gocv.BitwiseOr(*dest, *src, dest)
}

func (processor *IterativeTriclassProcessor) maskRegion(original, mask, result *gocv.Mat) {
	*result = gocv.NewMatWithSize(original.Rows(), original.Cols(), gocv.MatTypeCV8UC1)
	result.SetTo(gocv.NewScalar(0, 0, 0, 0))

	rows := original.Rows()
	cols := original.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if mask.GetUCharAt(y, x) > 0 {
				result.SetUCharAt(y, x, original.GetUCharAt(y, x))
			}
		}
	}
}

func (processor *IterativeTriclassProcessor) createFinalResult(foregroundMask, backgroundMask, result *gocv.Mat) {
	rows := foregroundMask.Rows()
	cols := foregroundMask.Cols()

	*result = gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if foregroundMask.GetUCharAt(y, x) > 0 {
				result.SetUCharAt(y, x, 255) // Foreground
			} else {
				result.SetUCharAt(y, x, 0) // Background
			}
		}
	}
}

func (processor *IterativeTriclassProcessor) applyPreprocessing(src, dst *gocv.Mat) {
	// Apply CLAHE for contrast enhancement
	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	clahe.Apply(*src, dst)

	// Apply denoising with simplified parameters
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
