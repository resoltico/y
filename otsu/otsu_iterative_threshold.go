package otsu

import (
	"gocv.io/x/gocv"
)

type TriclassThresholder struct {
	params map[string]interface{}
}

func NewTriclassThresholder(params map[string]interface{}) *TriclassThresholder {
	return &TriclassThresholder{
		params: params,
	}
}

func (thresholder *TriclassThresholder) CalculateThresholdForRegion(region *gocv.Mat) float64 {
	// Build histogram for the region
	histogram := thresholder.calculateHistogram(region)

	method := thresholder.getStringParam("initial_threshold_method")
	switch method {
	case "mean":
		return thresholder.calculateMeanThreshold(histogram)
	case "median":
		return thresholder.calculateMedianThreshold(histogram)
	default: // "otsu"
		return thresholder.calculateOtsuThreshold(histogram)
	}
}

func (thresholder *TriclassThresholder) calculateHistogram(src *gocv.Mat) []int {
	histBins := thresholder.getIntParam("histogram_bins")
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

func (thresholder *TriclassThresholder) calculateOtsuThreshold(histogram []int) float64 {
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

func (thresholder *TriclassThresholder) calculateMeanThreshold(histogram []int) float64 {
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

func (thresholder *TriclassThresholder) calculateMedianThreshold(histogram []int) float64 {
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

func (thresholder *TriclassThresholder) getIntParam(name string) int {
	if value, ok := thresholder.params[name].(int); ok {
		return value
	}
	return 0
}

func (thresholder *TriclassThresholder) getStringParam(name string) string {
	if value, ok := thresholder.params[name].(string); ok {
		return value
	}
	return ""
}