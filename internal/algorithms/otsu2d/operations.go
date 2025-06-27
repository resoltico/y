package otsu2d

import (
	"fmt"
	"image"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

func (p *Processor) calculateNeighborhoodMean(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	windowSize := p.getIntParam(params, "window_size")
	metric := p.getStringParam(params, "neighbourhood_metric")

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	switch metric {
	case "mean":
		gocv.Blur(srcMat, &dstMat, image.Point{X: windowSize, Y: windowSize})
	case "median":
		if windowSize%2 == 0 {
			windowSize++
		}
		gocv.MedianBlur(srcMat, &dstMat, windowSize)
	case "gaussian":
		sigma := float64(windowSize) / 3.0
		gocv.GaussianBlur(srcMat, &dstMat, image.Point{X: windowSize, Y: windowSize},
			sigma, sigma, gocv.BorderDefault)
	default:
		gocv.Blur(srcMat, &dstMat, image.Point{X: windowSize, Y: windowSize})
	}

	return dst, nil
}

func (p *Processor) applyCLAHE(src *safe.Mat) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	clahe.Apply(srcMat, &dstMat)

	return dst, nil
}

func (p *Processor) find2DOtsuThreshold(histogram [][]float64) [2]int {
	histBins := len(histogram)
	bestThreshold := [2]int{histBins / 2, histBins / 2}
	maxVariance := 0.0

	totalSum := 0.0
	totalCount := 0.0

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			weight := histogram[i][j]
			totalSum += float64(i*histBins+j) * weight
			totalCount += weight
		}
	}

	if totalCount == 0 {
		return bestThreshold
	}

	for t1 := 1; t1 < histBins-1; t1++ {
		for t2 := 1; t2 < histBins-1; t2++ {
			var w0, w1, sum0, sum1 float64

			for i := 0; i <= t1; i++ {
				for j := 0; j <= t2; j++ {
					weight := histogram[i][j]
					w0 += weight
					sum0 += float64(i*histBins+j) * weight
				}
			}

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
				meanDiff := mean0 - mean1

				variance := w0 * w1 * meanDiff * meanDiff

				if variance > maxVariance {
					maxVariance = variance
					bestThreshold = [2]int{t1, t2}
				}
			}
		}
	}

	return bestThreshold
}

func (p *Processor) applyThreshold(src, neighborhood, dst *safe.Mat, threshold [2]int, params map[string]interface{}) error {
	histBins := p.getIntParam(params, "histogram_bins")
	pixelWeightFactor := p.getFloatParam(params, "pixel_weight_factor")

	rows := src.Rows()
	cols := src.Cols()
	binScale := float64(histBins-1) / 255.0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := src.GetUCharAt(y, x)
			if err != nil {
				return fmt.Errorf("failed to get pixel at (%d,%d): %w", x, y, err)
			}

			neighValue, err := neighborhood.GetUCharAt(y, x)
			if err != nil {
				return fmt.Errorf("failed to get neighborhood pixel at (%d,%d): %w", x, y, err)
			}

			feature := pixelWeightFactor*float64(pixelValue) +
				(1.0-pixelWeightFactor)*float64(neighValue)

			pixelBin := int(float64(pixelValue) * binScale)
			neighBin := int(feature * binScale)

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

			var value uint8
			if pixelBin > threshold[0] && neighBin > threshold[1] {
				value = 255
			} else {
				value = 0
			}

			if err := dst.SetUCharAt(y, x, value); err != nil {
				return fmt.Errorf("failed to set pixel at (%d,%d): %w", x, y, err)
			}
		}
	}

	return nil
}
