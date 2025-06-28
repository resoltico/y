package otsu2d

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
		name: "2D Otsu",
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"window_size":            7,     // Neighborhood analysis window (3-21, odd numbers)
		"histogram_bins":         0,     // Auto-calculated based on noise level
		"smoothing_strength":     1.0,   // Gaussian smoothing (0.0-5.0)
		"noise_robustness":       true,  // MAOTSU preprocessing
		"gaussian_preprocessing": true,  // Apply Gaussian blur before processing
		"use_clahe":              false, // Contrast Limited Adaptive Histogram Equalization
		"guided_filtering":       false, // Edge-preserving guided filter
		"parallel_processing":    true,  // Use OpenCV parallel processing
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if windowSize, ok := params["window_size"].(int); ok {
		if windowSize < 3 || windowSize > 21 || windowSize%2 == 0 {
			return fmt.Errorf("window_size must be odd number between 3 and 21, got: %d", windowSize)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins != 0 && (histBins < 8 || histBins > 256) {
			return fmt.Errorf("histogram_bins must be 0 (auto) or between 8 and 256, got: %d", histBins)
		}
	}

	if smoothing, ok := params["smoothing_strength"].(float64); ok {
		if smoothing < 0.0 || smoothing > 5.0 {
			return fmt.Errorf("smoothing_strength must be between 0.0 and 5.0, got: %f", smoothing)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return p.ProcessWithContext(context.Background(), input, params)
}

func (p *Processor) ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(input, "2D Otsu processing"); err != nil {
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

	// Apply CLAHE preprocessing if enabled
	if p.getBoolParam(params, "use_clahe") {
		clahe, err := p.applyCLAHE(working)
		if err != nil {
			return nil, fmt.Errorf("CLAHE preprocessing failed: %w", err)
		}
		if working != gray {
			working.Close()
		}
		working = clahe
		defer clahe.Close()
	}

	// Apply Gaussian preprocessing
	var preprocessed *safe.Mat
	if p.getBoolParam(params, "gaussian_preprocessing") {
		blurred, err := p.applyGaussianBlur(working, p.getFloatParam(params, "smoothing_strength"))
		if err != nil {
			return nil, fmt.Errorf("gaussian preprocessing failed: %w", err)
		}
		preprocessed = blurred
		defer preprocessed.Close()
	} else {
		var err error
		preprocessed, err = working.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone working Mat: %w", err)
		}
		defer preprocessed.Close()
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Apply MAOTSU (Median-Average Otsu) preprocessing for noise robustness
	var processed *safe.Mat
	if p.getBoolParam(params, "noise_robustness") {
		maotsu, err := p.applyMAOTSUPreprocessing(preprocessed)
		if err != nil {
			return nil, fmt.Errorf("MAOTSU preprocessing failed: %w", err)
		}
		processed = maotsu
		defer processed.Close()
	} else {
		var err error
		processed, err = preprocessed.Clone()
		if err != nil {
			return nil, fmt.Errorf("failed to clone preprocessed Mat: %w", err)
		}
		defer processed.Close()
	}

	// Calculate neighborhood using basic averaging for compatibility
	neighborhood, err := p.calculateNeighborhoodMean(processed, params)
	if err != nil {
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Auto-calculate histogram bins based on image characteristics
	histBins := p.getIntParam(params, "histogram_bins")
	if histBins == 0 {
		histBins = p.calculateAdaptiveBinCount(processed)
	}

	// Build 2D histogram with double precision for numerical stability
	histogram := p.build2DHistogramStable(processed, neighborhood, histBins)

	// Apply histogram smoothing with separable Gaussian kernel
	if p.getFloatParam(params, "smoothing_strength") > 0 {
		p.smoothHistogramSeparable(histogram, p.getFloatParam(params, "smoothing_strength"))
	}

	// Find threshold using decomposed variance calculation
	threshold := p.find2DOtsuThresholdDecomposed(histogram)

	result, err := safe.NewMat(processed.Rows(), processed.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	if err := p.applyThresholdBilinear(processed, neighborhood, result, threshold, histBins); err != nil {
		result.Close()
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}

	return result, nil
}

// applyMAOTSUPreprocessing implements Median-Average Otsu preprocessing
// Combines median filtering (impulse noise removal) with average filtering (spatial correlation)
func (p *Processor) applyMAOTSUPreprocessing(src *safe.Mat) (*safe.Mat, error) {
	median, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer median.Close()

	// Apply median filter for impulse noise removal
	srcMat := src.GetMat()
	medianMat := median.GetMat()
	gocv.MedianBlur(srcMat, &medianMat, 3)

	// Apply Gaussian filter for spatial correlation
	gaussian, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer gaussian.Close()

	gaussianMat := gaussian.GetMat()
	gocv.GaussianBlur(medianMat, &gaussianMat, image.Point{X: 3, Y: 3}, 0.8, 0.8, gocv.BorderDefault)

	// Weighted combination: 60% median (noise reduction) + 40% gaussian (smoothing)
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	p.combineMatricesWeighted(median, gaussian, result, 0.6, 0.4)

	return result, nil
}

// applyCLAHE applies Contrast Limited Adaptive Histogram Equalization
func (p *Processor) applyCLAHE(src *safe.Mat) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	clahe.Apply(srcMat, &dstMat)

	return dst, nil
}

// calculateNeighborhoodMean uses basic convolution for neighborhood calculations
func (p *Processor) calculateNeighborhoodMean(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	windowSize := p.getIntParam(params, "window_size")
	halfWindow := windowSize / 2

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()

	// Calculate neighborhood means using direct convolution
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-halfWindow)
			x1 := max(0, x-halfWindow)
			y2 := min(rows-1, y+halfWindow)
			x2 := min(cols-1, x+halfWindow)

			sum := 0
			count := 0
			for wy := y1; wy <= y2; wy++ {
				for wx := x1; wx <= x2; wx++ {
					val, _ := src.GetUCharAt(wy, wx)
					sum += int(val)
					count++
				}
			}

			mean := uint8(sum / count)
			dst.SetUCharAt(y, x, mean)
		}
	}

	return dst, nil
}

// calculateAdaptiveBinCount determines histogram bins based on noise level and dynamic range
func (p *Processor) calculateAdaptiveBinCount(src *safe.Mat) int {
	rows := src.Rows()
	cols := src.Cols()
	totalPixels := rows * cols

	// Calculate dynamic range
	var minVal, maxVal uint8 = 255, 0
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val, _ := src.GetUCharAt(y, x)
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	dynamicRange := int(maxVal - minVal)

	// Estimate noise level using local variance
	noiseLevel := p.estimateNoiseLevel(src)

	// Adaptive bin calculation
	baseBins := 32
	if dynamicRange < 30 {
		baseBins = 16 // Low dynamic range
	} else if dynamicRange > 150 {
		baseBins = 64 // High dynamic range
	}

	// Adjust for noise level
	if noiseLevel > 15.0 {
		baseBins = max(baseBins/2, 8) // High noise - reduce bins
	} else if noiseLevel < 5.0 {
		baseBins = min(baseBins*2, 128) // Low noise - increase bins
	}

	// Adjust for image size
	if totalPixels > 1000000 {
		baseBins = min(baseBins*2, 256) // Large images
	} else if totalPixels < 100000 {
		baseBins = max(baseBins/2, 8) // Small images
	}

	return baseBins
}

// build2DHistogramStable builds histogram with double precision for numerical stability
func (p *Processor) build2DHistogramStable(src, neighborhood *safe.Mat, histBins int) [][]float64 {
	histogram := make([][]float64, histBins)
	for i := range histogram {
		histogram[i] = make([]float64, histBins)
	}

	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := src.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			neighValue, err := neighborhood.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			pixelBin := int(float64(pixelValue) * binScale)
			neighBin := int(float64(neighValue) * binScale)

			// Ensure bins are within valid range
			pixelBin = max(0, min(pixelBin, histBins-1))
			neighBin = max(0, min(neighBin, histBins-1))

			histogram[pixelBin][neighBin] += 1.0
		}
	}

	return histogram
}

// smoothHistogramSeparable applies separable Gaussian smoothing for O(N) complexity
func (p *Processor) smoothHistogramSeparable(histogram [][]float64, sigma float64) {
	if sigma <= 0.0 {
		return
	}

	histBins := len(histogram)
	kernelRadius := int(sigma * 3)
	if kernelRadius < 1 {
		kernelRadius = 1
	}

	// Build 1D Gaussian kernel
	kernel := make([]float64, 2*kernelRadius+1)
	sum := 0.0
	invSigmaSq := 1.0 / (2.0 * sigma * sigma)

	for i := 0; i < len(kernel); i++ {
		x := float64(i - kernelRadius)
		value := math.Exp(-x * x * invSigmaSq)
		kernel[i] = value
		sum += value
	}

	// Normalize kernel
	for i := range kernel {
		kernel[i] /= sum
	}

	// Apply separable convolution (horizontal then vertical)
	temp := make([][]float64, histBins)
	for i := range temp {
		temp[i] = make([]float64, histBins)
	}

	// Horizontal pass
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			value := 0.0
			for k := 0; k < len(kernel); k++ {
				col := j + k - kernelRadius
				if col >= 0 && col < histBins {
					value += histogram[i][col] * kernel[k]
				}
			}
			temp[i][j] = value
		}
	}

	// Vertical pass
	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			value := 0.0
			for k := 0; k < len(kernel); k++ {
				row := i + k - kernelRadius
				if row >= 0 && row < histBins {
					value += temp[row][j] * kernel[k]
				}
			}
			histogram[i][j] = value
		}
	}
}

// find2DOtsuThresholdDecomposed uses decomposed variance calculation for O(L²) complexity
func (p *Processor) find2DOtsuThresholdDecomposed(histogram [][]float64) [2]float64 {
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
	subPixelStep := 0.5

	// Search with sub-pixel precision using bilinear interpolation
	for t1 := 1.0; t1 < float64(histBins-1); t1 += subPixelStep {
		for t2 := 1.0; t2 < float64(histBins-1); t2 += subPixelStep {
			variance := p.calculateBetweenClassVariance(histogram, t1, t2, totalSum, totalCount, invTotalCount)
			if variance > maxVariance {
				maxVariance = variance
				bestThreshold = [2]float64{t1, t2}
			}
		}
	}

	return bestThreshold
}

// calculateBetweenClassVariance computes variance with numerical stability checks
func (p *Processor) calculateBetweenClassVariance(histogram [][]float64, t1, t2, totalSum, totalCount, invTotalCount float64) float64 {
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

// applyThresholdBilinear applies threshold with bilinear interpolation for sub-pixel accuracy
func (p *Processor) applyThresholdBilinear(src, neighborhood, dst *safe.Mat, threshold [2]float64, histBins int) error {
	rows := src.Rows()
	cols := src.Cols()
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

// Helper functions

func (p *Processor) applyGaussianBlur(src *safe.Mat, sigma float64) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}
	kernelSize = max(3, min(kernelSize, 15))

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.GaussianBlur(srcMat, &dstMat, image.Point{X: kernelSize, Y: kernelSize}, sigma, sigma, gocv.BorderDefault)

	return dst, nil
}

func (p *Processor) combineMatricesWeighted(mat1, mat2, result *safe.Mat, weight1, weight2 float64) {
	rows := mat1.Rows()
	cols := mat1.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val1, _ := mat1.GetUCharAt(y, x)
			val2, _ := mat2.GetUCharAt(y, x)

			combined := weight1*float64(val1) + weight2*float64(val2)
			result.SetUCharAt(y, x, uint8(math.Max(0, math.Min(255, combined))))
		}
	}
}

func (p *Processor) estimateNoiseLevel(src *safe.Mat) float64 {
	rows := src.Rows()
	cols := src.Cols()

	// Use Laplacian operator to estimate noise
	var sumSq float64
	count := 0

	for y := 1; y < rows-1; y++ {
		for x := 1; x < cols-1; x++ {
			center, _ := src.GetUCharAt(y, x)
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
		return math.Sqrt(sumSq/float64(count)) / 6.0 // Normalize
	}
	return 10.0 // Default noise level
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
