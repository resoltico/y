package otsu2d

import (
	"math"

	"otsu-obliterator/internal/opencv/safe"
)

func (p *Processor) build2DHistogram(src, neighborhood *safe.Mat, params map[string]interface{}) [][]float64 {
	histBins := p.getIntParam(params, "histogram_bins")
	pixelWeightFactor := p.getFloatParam(params, "pixel_weight_factor")

	histogram := make([][]float64, histBins)
	for i := range histogram {
		histogram[i] = make([]float64, histBins)
	}

	rows := src.Rows()
	cols := src.Cols()

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

			feature := pixelWeightFactor*float64(pixelValue) +
				(1.0-pixelWeightFactor)*float64(neighValue)

			pixelBin := int(float64(pixelValue) * float64(histBins-1) / 255.0)
			neighBin := int(feature * float64(histBins-1) / 255.0)

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

func (p *Processor) smoothHistogram(histogram [][]float64, sigma float64) {
	if sigma <= 0.0 {
		return
	}

	histBins := len(histogram)
	kernelSize := int(sigma*3)*2 + 1

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

	for i := 0; i < kernelSize; i++ {
		for j := 0; j < kernelSize; j++ {
			kernel[i][j] /= sum
		}
	}

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

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			histogram[i][j] = smoothed[i][j]
		}
	}
}

func (p *Processor) applyLogScaling(histogram [][]float64) {
	histBins := len(histogram)

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			if histogram[i][j] > 0 {
				histogram[i][j] = math.Log(1.0 + histogram[i][j])
			}
		}
	}
}

func (p *Processor) normalizeHistogram(histogram [][]float64) {
	histBins := len(histogram)
	total := 0.0

	for i := 0; i < histBins; i++ {
		for j := 0; j < histBins; j++ {
			total += histogram[i][j]
		}
	}

	if total > 0 {
		for i := 0; i < histBins; i++ {
			for j := 0; j < histBins; j++ {
				histogram[i][j] /= total
			}
		}
	}
}