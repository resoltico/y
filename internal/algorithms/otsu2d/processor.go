package otsu2d

import (
	"context"
	"fmt"
	"image"

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
		"quality":                    "Fast",
		"window_size":                7,
		"histogram_bins":             64,
		"neighbourhood_metric":       "mean",
		"pixel_weight_factor":        0.5,
		"smoothing_sigma":            1.0,
		"use_log_histogram":          false,
		"normalize_histogram":        true,
		"apply_contrast_enhancement": false,
		"gaussian_preprocessing":     true,
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if quality, ok := params["quality"].(string); ok {
		if quality != "Fast" && quality != "Best" {
			return fmt.Errorf("quality must be 'Fast' or 'Best', got: %s", quality)
		}
	}

	if windowSize, ok := params["window_size"].(int); ok {
		if windowSize < 3 || windowSize > 21 || windowSize%2 == 0 {
			return fmt.Errorf("window_size must be odd number between 3 and 21, got: %d", windowSize)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins < 16 || histBins > 256 {
			return fmt.Errorf("histogram_bins must be between 16 and 256, got: %d", histBins)
		}
	}

	if metric, ok := params["neighbourhood_metric"].(string); ok {
		if metric != "mean" && metric != "median" && metric != "gaussian" {
			return fmt.Errorf("neighbourhood_metric must be 'mean', 'median', or 'gaussian', got: %s", metric)
		}
	}

	if weight, ok := params["pixel_weight_factor"].(float64); ok {
		if weight < 0.0 || weight > 1.0 {
			return fmt.Errorf("pixel_weight_factor must be between 0.0 and 1.0, got: %f", weight)
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

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	gray, err := conversion.ConvertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to grayscale: %w", err)
	}
	defer gray.Close()

	working := gray
	var blurred *safe.Mat

	if p.getBoolParam(params, "gaussian_preprocessing") {
		blurred, err = p.applyGaussianBlur(gray)
		if err != nil {
			return nil, fmt.Errorf("gaussian preprocessing failed: %w", err)
		}
		working = blurred
		defer blurred.Close()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var enhanced *safe.Mat
	if p.getBoolParam(params, "apply_contrast_enhancement") {
		enhanced, err = p.applyCLAHE(working)
		if err != nil {
			return nil, fmt.Errorf("contrast enhancement failed: %w", err)
		}
		working = enhanced
		defer enhanced.Close()
	}

	neighborhood, err := p.calculateNeighborhoodMean(working, params)
	if err != nil {
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	quality := p.getStringParam(params, "quality")
	var threshold [2]float64

	if quality == "Best" {
		histogram := p.build2DHistogramFloat(working, neighborhood, params)
		threshold = p.find2DOtsuThresholdFloat(histogram, params)
	} else {
		histogram := p.build2DHistogram(working, neighborhood, params)
		thresholdInt := p.find2DOtsuThreshold(histogram)
		threshold = [2]float64{float64(thresholdInt[0]), float64(thresholdInt[1])}
	}

	result, err := safe.NewMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	if err := p.applyThresholdFloat(working, neighborhood, result, threshold, params); err != nil {
		result.Close()
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}

	return result, nil
}

func (p *Processor) build2DHistogramFloat(src, neighborhood *safe.Mat, params map[string]interface{}) [][]float64 {
	histBins := p.getIntParam(params, "histogram_bins")
	pixelWeightFactor := p.getFloatParam(params, "pixel_weight_factor")

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

			feature := pixelWeightFactor*float64(pixelValue) +
				(1.0-pixelWeightFactor)*float64(neighValue)

			pixelBinFloat := float64(pixelValue) * binScale
			neighBinFloat := feature * binScale

			pixelBin := int(pixelBinFloat)
			neighBin := int(neighBinFloat)

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

func (p *Processor) find2DOtsuThresholdFloat(histogram [][]float64, params map[string]interface{}) [2]float64 {
	histBins := len(histogram)
	bestThreshold := [2]float64{float64(histBins) / 2.0, float64(histBins) / 2.0}
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

	subPixelStep := 0.1
	for t1 := 1.0; t1 < float64(histBins-1); t1 += subPixelStep {
		for t2 := 1.0; t2 < float64(histBins-1); t2 += subPixelStep {
			var w0, w1, sum0, sum1 float64

			t1Int := int(t1)
			t2Int := int(t2)

			for i := 0; i <= t1Int; i++ {
				for j := 0; j <= t2Int; j++ {
					weight := histogram[i][j]

					if float64(i) <= t1 && float64(j) <= t2 {
						interpolationFactor := 1.0
						if i == t1Int {
							interpolationFactor *= (t1 - float64(t1Int))
						}
						if j == t2Int {
							interpolationFactor *= (t2 - float64(t2Int))
						}

						weightInterpolated := weight * interpolationFactor
						w0 += weightInterpolated
						sum0 += float64(i*histBins+j) * weightInterpolated
					}
				}
			}

			for i := t1Int + 1; i < histBins; i++ {
				for j := t2Int + 1; j < histBins; j++ {
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
					bestThreshold = [2]float64{t1, t2}
				}
			}
		}
	}

	return bestThreshold
}

func (p *Processor) applyThresholdFloat(src, neighborhood, dst *safe.Mat, threshold [2]float64, params map[string]interface{}) error {
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

			pixelBin := float64(pixelValue) * binScale
			neighBin := feature * binScale

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

func (p *Processor) applyGaussianBlur(src *safe.Mat) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	gocv.GaussianBlur(srcMat, &dstMat, image.Point{X: 5, Y: 5}, 1.0, 1.0, gocv.BorderDefault)

	return dst, nil
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

func (p *Processor) getStringParam(params map[string]interface{}, key string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return ""
}
