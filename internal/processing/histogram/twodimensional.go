package histogram

import (
	"math"

	"otsu-obliterator/internal/opencv/safe"
)

type TwoDimensionalBuilder struct{}

func NewTwoDimensionalBuilder() *TwoDimensionalBuilder {
	return &TwoDimensionalBuilder{}
}

func (t *TwoDimensionalBuilder) Build(src, neighborhood *safe.Mat, params map[string]interface{}) ([][]float64, error) {
	histBins := t.getHistogramBins(src, params)
	return t.build2DHistogramStable(src, neighborhood, histBins), nil
}

func (t *TwoDimensionalBuilder) getHistogramBins(src *safe.Mat, params map[string]interface{}) int {
	if histBins, ok := params["histogram_bins"].(int); ok && histBins > 0 {
		return histBins
	}
	return t.calculateAdaptiveBinCount(src)
}

func (t *TwoDimensionalBuilder) calculateAdaptiveBinCount(src *safe.Mat) int {
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
	noiseLevel := t.estimateNoiseLevel(src)

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

func (t *TwoDimensionalBuilder) build2DHistogramStable(src, neighborhood *safe.Mat, histBins int) [][]float64 {
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

func (t *TwoDimensionalBuilder) estimateNoiseLevel(src *safe.Mat) float64 {
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

func (t *TwoDimensionalBuilder) SmoothHistogram(histogram [][]float64, sigma float64) {
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
