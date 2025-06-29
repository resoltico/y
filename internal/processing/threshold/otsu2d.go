package threshold

import (
	"fmt"

	"otsu-obliterator/internal/opencv/safe"
)

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
