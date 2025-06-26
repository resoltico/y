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

	// Check for cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	gray, err := conversion.ConvertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to grayscale: %w", err)
	}
	defer gray.Close()

	// Apply Gaussian preprocessing if enabled
	working := gray
	if p.getBoolParam(params, "gaussian_preprocessing") {
		blurred, err := p.applyGaussianBlur(gray)
		if err != nil {
			return nil, fmt.Errorf("gaussian preprocessing failed: %w", err)
		}
		working = blurred
		defer blurred.Close()
	}

	// Check for cancellation after preprocessing
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if p.getBoolParam(params, "apply_contrast_enhancement") {
		enhanced, err := p.applyCLAHE(working)
		if err != nil {
			return nil, fmt.Errorf("contrast enhancement failed: %w", err)
		}
		if working != gray {
			working.Close()
		}
		working = enhanced
		defer enhanced.Close()
	}

	neighborhood, err := p.calculateNeighborhoodMean(working, params)
	if err != nil {
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	// Check for cancellation before histogram processing
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	histogram := p.build2DHistogram(working, neighborhood, params)

	smoothingSigma := p.getFloatParam(params, "smoothing_sigma")
	if smoothingSigma > 0.0 {
		p.smoothHistogram(histogram, smoothingSigma)
	}

	if p.getBoolParam(params, "use_log_histogram") {
		p.applyLogScaling(histogram)
	}

	if p.getBoolParam(params, "normalize_histogram") {
		p.normalizeHistogram(histogram)
	}

	threshold := p.find2DOtsuThreshold(histogram)

	result, err := safe.NewMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	if err := p.applyThreshold(working, neighborhood, result, threshold, params); err != nil {
		result.Close()
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}

	return result, nil
}

func (p *Processor) applyGaussianBlur(src *safe.Mat) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	// Apply Gaussian blur with kernel size 5x5 for noise reduction
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
