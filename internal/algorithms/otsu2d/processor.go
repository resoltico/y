package otsu2d

import (
	"fmt"

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
	if err := safe.ValidateMatForOperation(input, "2D Otsu processing"); err != nil {
		return nil, err
	}

	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	gray, err := conversion.ConvertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to grayscale: %w", err)
	}
	defer gray.Close()

	if p.getBoolParam(params, "apply_contrast_enhancement") {
		enhanced, err := p.applyCLAHE(gray)
		if err != nil {
			return nil, fmt.Errorf("contrast enhancement failed: %w", err)
		}
		gray.Close()
		gray = enhanced
	}

	neighborhood, err := p.calculateNeighborhoodMean(gray, params)
	if err != nil {
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	histogram := p.build2DHistogram(gray, neighborhood, params)

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

	result, err := safe.NewMat(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	if err := p.applyThreshold(gray, neighborhood, result, threshold, params); err != nil {
		result.Close()
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}

	return result, nil
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
