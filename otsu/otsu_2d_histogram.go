package otsu

import (
	"fmt"
	"math"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// TwoDHistogramBuilder handles 2D histogram construction for Otsu thresholding
type TwoDHistogramBuilder struct {
	params       map[string]interface{}
	debugManager *debug.Manager
}

// NewTwoDHistogramBuilder creates a new histogram builder
func NewTwoDHistogramBuilder(params map[string]interface{}) *TwoDHistogramBuilder {
	return &TwoDHistogramBuilder{
		params:       params,
		debugManager: debug.NewManager(),
	}
}

// BuildHistogram constructs 2D histogram from source and neighborhood images
func (builder *TwoDHistogramBuilder) BuildHistogram(src, neighborhood *gocv.Mat) (*TwoDHistogramData, error) {
	stepTime := time.Now()
	defer builder.debugManager.LogAlgorithmStep("2D Otsu", "histogram_building", time.Since(stepTime))

	if src.Empty() || neighborhood.Empty() {
		return nil, fmt.Errorf("source or neighborhood Mat is empty")
	}

	histBins := builder.getIntParam("histogram_bins", 64)
	pixelWeight := builder.getFloatParam("pixel_weight_factor", 0.5)

	// Initialize histogram data structure
	histData := &TwoDHistogramData{
		bins:        histBins,
		histogram:   make([][]float64, histBins),
		pixelCounts: make([][]int, histBins),
		totalPixels: src.Rows() * src.Cols(),
		smoothed:    false,
		normalized:  false,
		logScaled:   false,
	}

	// Initialize 2D arrays
	for i := range histData.histogram {
		histData.histogram[i] = make([]float64, histBins)
		histData.pixelCounts[i] = make([]int, histBins)
	}

	// Build histogram using vectorized operations where possible
	err := builder.populateHistogram(src, neighborhood, histData, pixelWeight)
	if err != nil {
		return nil, err
	}

	// Apply post-processing transformations
	builder.applyHistogramTransformations(histData)

	builder.debugManager.LogHistogramStatistics("2D Otsu", histBins, histData.totalPixels,
		histData.smoothed, histData.normalized)

	return histData, nil
}

// populateHistogram fills the 2D histogram with pixel data
func (builder *TwoDHistogramBuilder) populateHistogram(src, neighborhood *gocv.Mat, histData *TwoDHistogramData, pixelWeight float64) error {
	rows := src.Rows()
	cols := src.Cols()

	if rows != neighborhood.Rows() || cols != neighborhood.Cols() {
		return fmt.Errorf("dimension mismatch: src %dx%d vs neighborhood %dx%d", 
			cols, rows, neighborhood.Cols(), neighborhood.Rows())
	}

	// Use efficient pixel access pattern
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Get pixel and neighborhood values
			pixelVal := float64(src.GetUCharAt(y, x))
			neighVal := float64(neighborhood.GetUCharAt(y, x))

			// Calculate blended feature using weighted combination
			blendedFeature := pixelWeight*pixelVal + (1.0-pixelWeight)*neighVal

			// Map values to histogram bins with proper scaling
			pixelBin := builder.mapToHistogramBin(pixelVal, histData.bins)
			featureBin := builder.mapToHistogramBin(blendedFeature, histData.bins)

			// Accumulate counts
			histData.pixelCounts[pixelBin][featureBin]++
			histData.histogram[pixelBin][featureBin] += 1.0
		}
	}

	return nil
}

// mapToHistogramBin maps intensity value to histogram bin index
func (builder *TwoDHistogramBuilder) mapToHistogramBin(value float64, bins int) int {
	// Map [0,255] intensity range to [0,bins-1] bin range
	bin := int(value * float64(bins-1) / 255.0)
	
	// Ensure bin is within valid range
	if bin < 0 {
		return 0
	}
	if bin >= bins {
		return bins - 1
	}
	
	return bin
}

// applyHistogramTransformations applies smoothing, log scaling, and normalization
func (builder *TwoDHistogramBuilder) applyHistogramTransformations(histData *TwoDHistogramData) {
	// Apply Gaussian smoothing if requested
	smoothingSigma := builder.getFloatParam("smoothing_sigma", 0.0)
	if smoothingSigma > 0.0 {
		builder.applyGaussianSmoothing(histData, smoothingSigma)
		histData.smoothed = true
	}

	// Apply logarithmic scaling if requested
	if builder.getBoolParam("use_log_histogram") {
		builder.applyLogScaling(histData)
		histData.logScaled = true
	}

	// Apply normalization if requested
	if builder.getBoolParam("normalize_histogram") {
		builder.normalizeHistogram(histData)
		histData.normalized = true
	}
}

// applyGaussianSmoothing applies 2D Gaussian smoothing to histogram
func (builder *TwoDHistogramBuilder) applyGaussianSmoothing(histData *TwoDHistogramData, sigma float64) {
	stepTime := time.Now()
	defer builder.debugManager.LogAlgorithmStep("2D Otsu", "histogram_smoothing", time.Since(stepTime))

	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}

	// Create 2D Gaussian kernel
	kernel := builder.createGaussianKernel(kernelSize, sigma)

	// Apply convolution to smooth histogram
	smoothed := make([][]float64, histData.bins)
	for i := range smoothed {
		smoothed[i] = make([]float64, histData.bins)
	}

	// Perform 2D convolution
	center := kernelSize / 2
	for i := 0; i < histData.bins; i++ {
		for j := 0; j < histData.bins; j++ {
			value := 0.0

			for ki := 0; ki < kernelSize; ki++ {
				for kj := 0; kj < kernelSize; kj++ {
					hi := i + ki - center
					hj := j + kj - center

					// Handle boundary conditions using zero-padding
					if hi >= 0 && hi < histData.bins && hj >= 0 && hj < histData.bins {
						value += histData.histogram[hi][hj] * kernel[ki][kj]
					}
				}
			}
			smoothed[i][j] = value
		}
	}

	histData.histogram = smoothed
}

// createGaussianKernel generates a 2D Gaussian kernel
func (builder *TwoDHistogramBuilder) createGaussianKernel(size int, sigma float64) [][]float64 {
	kernel := make([][]float64, size)
	center := size / 2
	sum := 0.0

	// Generate kernel values
	for i := 0; i < size; i++ {
		kernel[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			x := float64(i - center)
			y := float64(j - center)
			
			// 2D Gaussian formula
			value := math.Exp(-(x*x + y*y) / (2.0 * sigma * sigma))
			kernel[i][j] = value
			sum += value
		}
	}

	// Normalize kernel to sum to 1
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			kernel[i][j] /= sum
		}
	}

	return kernel
}

// applyLogScaling applies logarithmic scaling to histogram values
func (builder *TwoDHistogramBuilder) applyLogScaling(histData *TwoDHistogramData) {
	for i := 0; i < histData.bins; i++ {
		for j := 0; j < histData.bins; j++ {
			if histData.histogram[i][j] > 0 {
				// Use log1p for numerical stability with small values
				histData.histogram[i][j] = math.Log1p(histData.histogram[i][j])
			}
		}
	}
}

// normalizeHistogram converts histogram to probability distribution
func (builder *TwoDHistogramBuilder) normalizeHistogram(histData *TwoDHistogramData) {
	total := 0.0

	// Calculate total sum
	for i := 0; i < histData.bins; i++ {
		for j := 0; j < histData.bins; j++ {
			total += histData.histogram[i][j]
		}
	}

	// Normalize to probability distribution
	if total > 0 {
		for i := 0; i < histData.bins; i++ {
			for j := 0; j < histData.bins; j++ {
				histData.histogram[i][j] /= total
			}
		}
	}
}

// GetHistogramStatistics returns statistical information about the histogram
func (builder *TwoDHistogramBuilder) GetHistogramStatistics(histData *TwoDHistogramData) map[string]interface{} {
	stats := make(map[string]interface{})

	stats["bins"] = histData.bins
	stats["total_pixels"] = histData.totalPixels
	stats["smoothed"] = histData.smoothed
	stats["normalized"] = histData.normalized
	stats["log_scaled"] = histData.logScaled

	// Calculate histogram statistics
	minVal := math.Inf(1)
	maxVal := math.Inf(-1)
	nonZeroBins := 0

	for i := 0; i < histData.bins; i++ {
		for j := 0; j < histData.bins; j++ {
			val := histData.histogram[i][j]
			if val > 0 {
				nonZeroBins++
				if val < minVal {
					minVal = val
				}
				if val > maxVal {
					maxVal = val
				}
			}
		}
	}

	stats["non_zero_bins"] = nonZeroBins
	stats["sparsity"] = 1.0 - float64(nonZeroBins)/float64(histData.bins*histData.bins)
	
	if nonZeroBins > 0 {
		stats["min_value"] = minVal
		stats["max_value"] = maxVal
		stats["dynamic_range"] = maxVal - minVal
	} else {
		stats["min_value"] = 0.0
		stats["max_value"] = 0.0
		stats["dynamic_range"] = 0.0
	}

	return stats
}

// Parameter access utilities
func (builder *TwoDHistogramBuilder) getIntParam(name string, defaultValue int) int {
	if value, ok := builder.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (builder *TwoDHistogramBuilder) getFloatParam(name string, defaultValue float64) float64 {
	if value, ok := builder.params[name].(float64); ok {
		return value
	}
	return defaultValue
}

func (builder *TwoDHistogramBuilder) getBoolParam(name string) bool {
	if value, ok := builder.params[name].(bool); ok {
		return value
	}
	return false
}
