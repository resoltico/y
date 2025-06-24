package otsu

import (
	"image"
	"math"

	"gocv.io/x/gocv"
)

type TwoDOtsuProcessor struct {
	params        map[string]interface{}
	memoryManager MemoryManagerInterface
}

func NewTwoDOtsuProcessor(params map[string]interface{}, memoryManager MemoryManagerInterface) *TwoDOtsuProcessor {
	return &TwoDOtsuProcessor{
		params:        params,
		memoryManager: memoryManager,
	}
}

func (processor *TwoDOtsuProcessor) Process(src gocv.Mat) (gocv.Mat, error) {
	defer processor.memoryManager.ReleaseMat(src)

	// Convert to grayscale if needed
	gray := processor.memoryManager.GetMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(gray)

	if src.Channels() == 3 {
		gocv.CvtColor(src, &gray, gocv.ColorBGRToGray)
	} else {
		src.CopyTo(&gray)
	}

	// Apply contrast enhancement if requested
	if processor.getBoolParam("apply_contrast_enhancement") {
		enhanced := processor.memoryManager.GetMat(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
		defer processor.memoryManager.ReleaseMat(enhanced)

		processor.applyCLAHE(&gray, &enhanced)
		processor.memoryManager.ReleaseMat(gray)
		gray = enhanced.Clone()
	}

	// Calculate neighborhood averages
	neighborhood := processor.memoryManager.GetMat(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(neighborhood)

	err := processor.calculateNeighborhoodMean(&gray, &neighborhood)
	if err != nil {
		return gocv.NewMat(), err
	}

	// Build 2D histogram
	histogram := processor.build2DHistogram(&gray, &neighborhood)

	// Apply smoothing if requested
	smoothingSigma := processor.getFloatParam("smoothing_sigma")
	if smoothingSigma > 0.0 {
		processor.smoothHistogram(histogram, smoothingSigma)
	}

	// Apply log scaling if requested
	if processor.getBoolParam("use_log_histogram") {
		processor.applyLogScaling(histogram)
	}

	// Normalize if requested
	if processor.getBoolParam("normalize_histogram") {
		processor.normalizeHistogram(histogram)
	}

	// Find threshold using 2D Otsu
	threshold := processor.find2DOtsuThreshold(histogram)

	// Apply threshold
	result := processor.memoryManager.GetMat(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	processor.applyThreshold(&gray, &neighborhood, &result, threshold)

	return result, nil
}

func (processor *TwoDOtsuProcessor) calculateNeighborhoodMean(src, dst *gocv.Mat) error {
	windowSize := processor.getIntParam("window_size")
	metric := processor.getStringParam("neighbourhood_metric")

	kernel := gocv.GetStructuringElement(gocv.MorphRect,
		image.Point{X: windowSize, Y: windowSize})
	defer kernel.Close()

	switch metric {
	case "mean":
		gocv.Blur(*src, dst, image.Point{X: windowSize, Y: windowSize})
	case "median":
		gocv.MedianBlur(*src, dst, windowSize)
	case "gaussian":
		sigma := float64(windowSize) / 3.0
		gocv.GaussianBlur(*src, dst, image.Point{X: windowSize, Y: windowSize},
			sigma, sigma, gocv.BorderDefault)
	default:
		gocv.Blur(*src, dst, image.Point{X: windowSize, Y: windowSize})
	}

	return nil
}

func (processor *TwoDOtsuProcessor) build2DHistogram(src, neighborhood *gocv.Mat) [][]float64 {
	histBins := processor.getIntParam("histogram_bins")
	pixelWeightFactor := processor.getFloatParam("pixel_weight_factor")

	histogram := make([][]float64, histBins)
	for i := range histogram {
		histogram[i] = make([]float64, histBins)
	}

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := src.GetUCharAt(y, x)
			neighValue := neighborhood.GetUCharAt(y, x)

			// Blend pixel intensity and neighborhood mean
			feature := pixelWeightFactor*float64(pixelValue) +
				(1.0-pixelWeightFactor)*float64(neighValue)

			// Map to histogram bins
			pixelBin := int(float64(pixelValue) * float64(histBins-1) / 255.0)
			neighBin := int(feature * float64(histBins-1) / 255.0)

			// Clamp to valid range
			if pixelBin < 0 {
				pixelBin = 0
			} else if pixelBin >= histBins {
				pixelBin = histBins - 1
			}

			if neighBin < 0 {
				neighBin = 0
			} else if neighBin >= histBins {
				neighBin = histBins - 1
			}

			histogram[pixelBin][neighBin]++
		}
	}

	return histogram
}

func (processor *TwoDOtsuProcessor) smoothHistogram(histogram [][]float64, sigma float64) {
	if sigma <= 0.0 {
		return
	}

	histBins := len(histogram)
	kernelSize := int(sigma*3)*2 + 1

	// Create Gaussian kernel
	kernel := make([][]float64, kernelSize)
	for i := range kernel {
		kernel[i] = make([]float64, kernelSize)
	}

	center := kernelSize / 2
	sum := 0.0

	for i := 0; i < kernelSize; i++ {
		for j := 0; j < kernelSize; j++ {
			x := float64(i - center)
			y := float64(j - center)
			value := math.Exp(-(x*x + y*y) / (2.0 * sigma * sigma))
			kernel[i][j] = value
			sum += value
		}
	}

	// Normalize kernel
	for i := 0; i < kernelSize; i++ {
		for j := 0; j < kernelSize; j++ {
			kernel[i][j] /= sum
		}
	}

	// Apply convolution
	smoothed := make([][]float64, histBins)
	for i := range smoothed {
		smoothed[i] = make([]float64, histBins)
	}

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			value := 0.0

			for ki := 0; ki < kernelSize; ki++ {
				for kj := 0; kj < kernelSize; kj++ {
					hi := i + ki - center
					hj := j + kj - center

					if hi >= 0 && hi < histBins && hj >= 0 && hj < histBins {
						value += histogram[hi][hj] * kernel[ki][kj]
					}
				}
			}

			smoothed[i][j] = value
		}
	}

	// Copy back
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			histogram[i][j] = smoothed[i][j]
		}
	}
}

func (processor *TwoDOtsuProcessor) applyLogScaling(histogram [][]float64) {
	histBins := len(histogram)

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			if histogram[i][j] > 0 {
				histogram[i][j] = math.Log(1.0 + histogram[i][j])
			}
		}
	}
}

func (processor *TwoDOtsuProcessor) normalizeHistogram(histogram [][]float64) {
	histBins := len(histogram)
	total := 0.0

	// Calculate total
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			total += histogram[i][j]
		}
	}

	// Normalize
	if total > 0 {
		for i := 0; i < histBins; i++ {
			for j := 0; j < histBins; j++ {
				histogram[i][j] /= total
			}
		}
	}
}

func (processor *TwoDOtsuProcessor) find2DOtsuThreshold(histogram [][]float64) [2]int {
	histBins := len(histogram)
	bestThreshold := [2]int{histBins / 2, histBins / 2}
	maxVariance := 0.0

	// Calculate total sum and count
	totalSum := 0.0
	totalCount := 0.0

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			totalSum += float64(i*histBins+j) * histogram[i][j]
			totalCount += histogram[i][j]
		}
	}

	if totalCount == 0 {
		return bestThreshold
	}

	// Search for threshold
	for t1 := 1; t1 < histBins-1; t1++ {
		for t2 := 1; t2 < histBins-1; t2++ {
			// Calculate class statistics
			w0, w1 := 0.0, 0.0
			sum0, sum1 := 0.0, 0.0

			// Background class (i <= t1, j <= t2)
			for i := 0; i <= t1; i++ {
				for j := 0; j <= t2; j++ {
					weight := histogram[i][j]
					w0 += weight
					sum0 += float64(i*histBins+j) * weight
				}
			}

			// Foreground class (i > t1, j > t2)
			for i := t1 + 1; i < histBins; i++ {
				for j := t2 + 1; j < histBins; j++ {
					weight := histogram[i][j]
					w1 += weight
					sum1 += float64(i*histBins+j) * weight
				}
			}

			if w0 > 0 && w1 > 0 {
				mean0 := sum0 / w0
				mean1 := sum1 / w1

				// Between-class variance
				variance := w0 * w1 * (mean0 - mean1) * (mean0 - mean1)

				if variance > maxVariance {
					maxVariance = variance
					bestThreshold = [2]int{t1, t2}
				}
			}
		}
	}

	return bestThreshold
}

func (processor *TwoDOtsuProcessor) applyThreshold(src, neighborhood, dst *gocv.Mat, threshold [2]int) {
	histBins := processor.getIntParam("histogram_bins")
	pixelWeightFactor := processor.getFloatParam("pixel_weight_factor")

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := src.GetUCharAt(y, x)
			neighValue := neighborhood.GetUCharAt(y, x)

			// Calculate feature value same as in histogram building
			feature := pixelWeightFactor*float64(pixelValue) +
				(1.0-pixelWeightFactor)*float64(neighValue)

			// Map to histogram bins
			pixelBin := int(float64(pixelValue) * float64(histBins-1) / 255.0)
			neighBin := int(feature * float64(histBins-1) / 255.0)

			// Clamp to valid range
			if pixelBin < 0 {
				pixelBin = 0
			} else if pixelBin >= histBins {
				pixelBin = histBins - 1
			}

			if neighBin < 0 {
				neighBin = 0
			} else if neighBin >= histBins {
				neighBin = histBins - 1
			}

			// Apply threshold
			if pixelBin > threshold[0] && neighBin > threshold[1] {
				dst.SetUCharAt(y, x, 255) // Foreground
			} else {
				dst.SetUCharAt(y, x, 0) // Background
			}
		}
	}
}

func (processor *TwoDOtsuProcessor) applyCLAHE(src, dst *gocv.Mat) {
	// Create CLAHE with basic parameters
	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	// Apply CLAHE
	clahe.Apply(*src, dst)
}

func (processor *TwoDOtsuProcessor) getIntParam(name string) int {
	if value, ok := processor.params[name].(int); ok {
		return value
	}
	return 0
}

func (processor *TwoDOtsuProcessor) getFloatParam(name string) float64 {
	if value, ok := processor.params[name].(float64); ok {
		return value
	}
	return 0.0
}

func (processor *TwoDOtsuProcessor) getBoolParam(name string) bool {
	if value, ok := processor.params[name].(bool); ok {
		return value
	}
	return false
}

func (processor *TwoDOtsuProcessor) getStringParam(name string) string {
	if value, ok := processor.params[name].(string); ok {
		return value
	}
	return ""
}
