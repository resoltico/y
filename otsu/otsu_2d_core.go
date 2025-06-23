package otsu

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// TwoDOtsuCore handles the core 2D Otsu thresholding algorithm
type TwoDOtsuCore struct {
	params        map[string]interface{}
	debugManager  *debug.Manager
	histogramData *TwoDHistogramData
}

// TwoDHistogramData encapsulates 2D histogram construction and processing
type TwoDHistogramData struct {
	histogram     [][]float64
	pixelCounts   [][]int
	bins          int
	totalPixels   int
	smoothed      bool
	normalized    bool
	logScaled     bool
}

// NewTwoDOtsuCore creates a new 2D Otsu processor with debug support
func NewTwoDOtsuCore(params map[string]interface{}) *TwoDOtsuCore {
	return &TwoDOtsuCore{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

// Process applies 2D Otsu thresholding to the input Mat
func (core *TwoDOtsuCore) Process(src gocv.Mat) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	core.debugManager.LogAlgorithmStart("2D Otsu", core.params)
	startTime := core.debugManager.StartTiming("2d_otsu_full_process")
	defer core.debugManager.EndTiming("2d_otsu_full_process", startTime)

	// Ensure we have a working copy that won't be modified
	working := src.Clone()
	defer working.Close()

	// Convert to grayscale if needed
	gray, err := core.prepareGrayscaleImage(&working)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	// Apply preprocessing if requested
	preprocessed, err := core.applyPreprocessing(&gray)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer preprocessed.Close()

	// Calculate neighborhood features
	neighborhood, err := core.calculateNeighborhoodFeatures(&preprocessed)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer neighborhood.Close()

	// Build and process 2D histogram
	err = core.build2DHistogram(&preprocessed, &neighborhood)
	if err != nil {
		return gocv.NewMat(), err
	}

	// Find threshold using 2D Otsu criterion
	threshold := core.findOptimalThreshold()
	core.debugManager.LogThresholdCalculation("2D Otsu", threshold, "variance_maximization")

	// Apply threshold to create binary result
	result := core.applyBinaryThreshold(&preprocessed, &neighborhood, threshold)

	core.debugManager.LogAlgorithmComplete("2D Otsu", core.debugManager.StartTiming("2d_otsu_full_process"), 
		fmt.Sprintf("%dx%d", result.Cols(), result.Rows()))

	return result, nil
}

// prepareGrayscaleImage converts input to grayscale using latest GoCV APIs
func (core *TwoDOtsuCore) prepareGrayscaleImage(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("2d_otsu_grayscale_conversion")
	defer core.debugManager.EndTiming("2d_otsu_grayscale_conversion", stepTime)

	gray := gocv.NewMat()

	if src.Channels() == 3 {
		// Use latest BGR to GRAY conversion
		gocv.CvtColor(*src, &gray, gocv.ColorBGRToGray)
	} else if src.Channels() == 4 {
		// Handle BGRA to GRAY conversion
		gocv.CvtColor(*src, &gray, gocv.ColorBGRAToGray)
	} else {
		// Already grayscale
		src.CopyTo(&gray)
	}

	if gray.Empty() {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed")
	}

	return gray, nil
}

// applyPreprocessing applies CLAHE and noise reduction if enabled
func (core *TwoDOtsuCore) applyPreprocessing(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("2d_otsu_preprocessing")
	defer core.debugManager.EndTiming("2d_otsu_preprocessing", stepTime)

	result := gocv.NewMat()

	if core.getBoolParam("apply_contrast_enhancement") {
		// Apply CLAHE for contrast enhancement using latest API
		clahe := gocv.NewCLAHEWithParams(2.0, image.Point{X: 8, Y: 8})
		defer clahe.Close()

		clahe.Apply(*src, &result)
		
		// Optional denoising
		if core.getStringParam("quality") == "Best" {
			denoised := gocv.NewMat()
			defer denoised.Close()
			
			// Use latest denoising API with automatic parameter selection
			gocv.FastNlMeansDenoising(result, &denoised)
			denoised.CopyTo(&result)
		}
	} else {
		src.CopyTo(&result)
	}

	return result, nil
}

// calculateNeighborhoodFeatures computes neighborhood-based features
func (core *TwoDOtsuCore) calculateNeighborhoodFeatures(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("2d_otsu_neighborhood_calculation")
	defer core.debugManager.EndTiming("2d_otsu_neighborhood_calculation", stepTime)

	windowSize := core.getIntParam("window_size")
	if windowSize%2 == 0 {
		windowSize++ // Ensure odd window size
	}

	metric := core.getStringParam("neighbourhood_metric")
	result := gocv.NewMat()

	switch metric {
	case "mean":
		// Use latest blur API for mean calculation
		gocv.BoxFilter(*src, &result, -1, image.Point{X: windowSize, Y: windowSize}, 
			image.Point{X: -1, Y: -1}, true, gocv.BorderReflect101)

	case "median":
		// Median filtering with proper bounds checking
		if windowSize > 31 {
			windowSize = 31 // OpenCV limitation for median
		}
		gocv.MedianBlur(*src, &result, windowSize)

	case "gaussian":
		// Gaussian blur with adaptive sigma
		sigma := float64(windowSize) / 6.0 // More conservative sigma
		gocv.GaussianBlur(*src, &result, image.Point{X: windowSize, Y: windowSize},
			sigma, sigma, gocv.BorderReflect101)

	default:
		// Default to mean
		gocv.BoxFilter(*src, &result, -1, image.Point{X: windowSize, Y: windowSize}, 
			image.Point{X: -1, Y: -1}, true, gocv.BorderReflect101)
	}

	return result, nil
}

// build2DHistogram constructs the 2D histogram using vectorized operations
func (core *TwoDOtsuCore) build2DHistogram(src, neighborhood *gocv.Mat) error {
	stepTime := core.debugManager.StartTiming("2d_otsu_histogram_building")
	defer core.debugManager.EndTiming("2d_otsu_histogram_building", stepTime)

	histBins := core.getIntParam("histogram_bins")
	pixelWeight := core.getFloatParam("pixel_weight_factor")

	// Initialize histogram data
	core.histogramData = &TwoDHistogramData{
		bins:        histBins,
		histogram:   make([][]float64, histBins),
		pixelCounts: make([][]int, histBins),
		totalPixels: src.Rows() * src.Cols(),
	}

	// Initialize 2D arrays
	for i := range core.histogramData.histogram {
		core.histogramData.histogram[i] = make([]float64, histBins)
		core.histogramData.pixelCounts[i] = make([]int, histBins)
	}

	// Build histogram using efficient pixel access
	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelVal := float64(src.GetUCharAt(y, x))
			neighVal := float64(neighborhood.GetUCharAt(y, x))

			// Blend pixel intensity and neighborhood feature
			blendedFeature := pixelWeight*pixelVal + (1.0-pixelWeight)*neighVal

			// Map to histogram bins with proper bounds checking
			pixelBin := int(pixelVal * float64(histBins-1) / 255.0)
			featureBin := int(blendedFeature * float64(histBins-1) / 255.0)

			// Clamp to valid ranges
			pixelBin = clampToRange(pixelBin, 0, histBins-1)
			featureBin = clampToRange(featureBin, 0, histBins-1)

			core.histogramData.pixelCounts[pixelBin][featureBin]++
			core.histogramData.histogram[pixelBin][featureBin] += 1.0
		}
	}

	// Apply post-processing to histogram
	core.processHistogram()

	return nil
}

// processHistogram applies smoothing, log scaling, and normalization
func (core *TwoDOtsuCore) processHistogram() {
	// Apply smoothing if requested
	smoothingSigma := core.getFloatParam("smoothing_sigma")
	if smoothingSigma > 0.0 {
		core.applyGaussianSmoothing(smoothingSigma)
		core.histogramData.smoothed = true
	}

	// Apply log scaling if requested
	if core.getBoolParam("use_log_histogram") {
		core.applyLogScaling()
		core.histogramData.logScaled = true
	}

	// Apply normalization if requested
	if core.getBoolParam("normalize_histogram") {
		core.normalizeHistogram()
		core.histogramData.normalized = true
	}
}

// applyGaussianSmoothing applies 2D Gaussian smoothing to the histogram
func (core *TwoDOtsuCore) applyGaussianSmoothing(sigma float64) {
	stepTime := core.debugManager.StartTiming("2d_otsu_histogram_smoothing")
	defer core.debugManager.EndTiming("2d_otsu_histogram_smoothing", stepTime)

	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}

	// Create 2D Gaussian kernel
	kernel := make([][]float64, kernelSize)
	center := kernelSize / 2
	sum := 0.0

	for i := 0; i < kernelSize; i++ {
		kernel[i] = make([]float64, kernelSize)
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
	bins := core.histogramData.bins
	smoothed := make([][]float64, bins)
	for i := range smoothed {
		smoothed[i] = make([]float64, bins)
	}

	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			value := 0.0

			for ki := 0; ki < kernelSize; ki++ {
				for kj := 0; kj < kernelSize; kj++ {
					hi := i + ki - center
					hj := j + kj - center

					if hi >= 0 && hi < bins && hj >= 0 && hj < bins {
						value += core.histogramData.histogram[hi][hj] * kernel[ki][kj]
					}
				}
			}
			smoothed[i][j] = value
		}
	}

	core.histogramData.histogram = smoothed
}

// applyLogScaling applies logarithmic scaling to histogram values
func (core *TwoDOtsuCore) applyLogScaling() {
	bins := core.histogramData.bins
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			if core.histogramData.histogram[i][j] > 0 {
				core.histogramData.histogram[i][j] = math.Log1p(core.histogramData.histogram[i][j])
			}
		}
	}
}

// normalizeHistogram converts histogram to probability distribution
func (core *TwoDOtsuCore) normalizeHistogram() {
	bins := core.histogramData.bins
	total := 0.0

	// Calculate total
	for i := 0; i < bins; i++ {
		for j := 0; j < bins; j++ {
			total += core.histogramData.histogram[i][j]
		}
	}

	// Normalize
	if total > 0 {
		for i := 0; i < bins; i++ {
			for j := 0; j < bins; j++ {
				core.histogramData.histogram[i][j] /= total
			}
		}
	}
}

// Utility functions for parameter access
func (core *TwoDOtsuCore) getIntParam(name string) int {
	if value, ok := core.params[name].(int); ok {
		return value
	}
	// Return sensible defaults
	defaults := map[string]int{
		"window_size":     7,
		"histogram_bins":  64,
	}
	if def, exists := defaults[name]; exists {
		return def
	}
	return 0
}

func (core *TwoDOtsuCore) getFloatParam(name string) float64 {
	if value, ok := core.params[name].(float64); ok {
		return value
	}
	// Return sensible defaults
	defaults := map[string]float64{
		"pixel_weight_factor": 0.5,
		"smoothing_sigma":     1.0,
	}
	if def, exists := defaults[name]; exists {
		return def
	}
	return 0.0
}

func (core *TwoDOtsuCore) getBoolParam(name string) bool {
	if value, ok := core.params[name].(bool); ok {
		return value
	}
	return false
}

func (core *TwoDOtsuCore) getStringParam(name string) string {
	if value, ok := core.params[name].(string); ok {
		return value
	}
	// Return sensible defaults
	defaults := map[string]string{
		"neighbourhood_metric": "mean",
		"quality":             "Fast",
	}
	if def, exists := defaults[name]; exists {
		return def
	}
	return ""
}

// clampToRange ensures value is within [min, max]
func clampToRange(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
