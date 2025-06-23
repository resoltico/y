package otsu

import "gocv.io/x/gocv"

// TwoDHistogramData encapsulates 2D histogram construction and processing
type TwoDHistogramData struct {
	histogram   [][]float64
	pixelCounts [][]int
	bins        int
	totalPixels int
	smoothed    bool
	normalized  bool
	logScaled   bool
}

// TwoDThreshold represents a 2D threshold point
type TwoDThreshold struct {
	PixelThreshold   int
	FeatureThreshold int
	Variance         float64
}

// ProcessingResult encapsulates algorithm processing results with metadata (legacy compatibility)
type ProcessingResult struct {
	Result      gocv.Mat
	Algorithm   string
	Parameters  map[string]interface{}
	Statistics  map[string]interface{}
	ProcessTime float64 // in milliseconds
}

// GetThresholdInfo returns human-readable threshold information
func (threshold *TwoDThreshold) GetThresholdInfo() map[string]interface{} {
	return map[string]interface{}{
		"pixel_threshold":   threshold.PixelThreshold,
		"feature_threshold": threshold.FeatureThreshold,
		"variance":          threshold.Variance,
		"threshold_type":    "2D_Otsu",
	}
}

// IsValid checks if the threshold is mathematically valid
func (threshold *TwoDThreshold) IsValid(maxBins int) bool {
	return threshold.PixelThreshold >= 0 &&
		threshold.PixelThreshold < maxBins &&
		threshold.FeatureThreshold >= 0 &&
		threshold.FeatureThreshold < maxBins &&
		threshold.Variance >= 0.0
}
