package triclass

import (
	"fmt"

	"otsu-obliterator/internal/opencv/conversion"
	"otsu-obliterator/internal/opencv/safe"
)

type Processor struct {
	name string
}

func NewProcessor() *Processor {
	return &Processor{
		name: "Iterative Triclass",
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"quality":                   "Fast",
		"initial_threshold_method":  "otsu",
		"histogram_bins":            64,
		"convergence_epsilon":       1.0,
		"max_iterations":            10,
		"minimum_tbd_fraction":      0.01,
		"lower_upper_gap_factor":    0.5,
		"apply_preprocessing":       false,
		"apply_cleanup":             true,
		"preserve_borders":          false,
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if quality, ok := params["quality"].(string); ok {
		if quality != "Fast" && quality != "Best" {
			return fmt.Errorf("quality must be 'Fast' or 'Best', got: %s", quality)
		}
	}

	if method, ok := params["initial_threshold_method"].(string); ok {
		if method != "otsu" && method != "mean" && method != "median" {
			return fmt.Errorf("initial_threshold_method must be 'otsu', 'mean', or 'median', got: %s", method)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins < 16 || histBins > 256 {
			return fmt.Errorf("histogram_bins must be between 16 and 256, got: %d", histBins)
		}
	}

	if epsilon, ok := params["convergence_epsilon"].(float64); ok {
		if epsilon < 0.1 || epsilon > 10.0 {
			return fmt.Errorf("convergence_epsilon must be between 0.1 and 10.0, got: %f", epsilon)
		}
	}

	if maxIter, ok := params["max_iterations"].(int); ok {
		if maxIter < 1 || maxIter > 20 {
			return fmt.Errorf("max_iterations must be between 1 and 20, got: %d", maxIter)
		}
	}

	if fraction, ok := params["minimum_tbd_fraction"].(float64); ok {
		if fraction < 0.001 || fraction > 0.2 {
			return fmt.Errorf("minimum_tbd_fraction must be between 0.001 and 0.2, got: %f", fraction)
		}
	}

	if gapFactor, ok := params["lower_upper_gap_factor"].(float64); ok {
		if gapFactor < 0.0 || gapFactor > 1.0 {
			return fmt.Errorf("lower_upper_gap_factor must be between 0.0 and 1.0, got: %f", gapFactor)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(input, "Iterative Triclass processing"); err != nil {
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

	working := gray
	if p.getBoolParam(params, "apply_preprocessing") {
		preprocessed, err := p.applyPreprocessing(gray)
		if err != nil {
			return nil, fmt.Errorf("preprocessing failed: %w", err)
		}
		if working != gray {
			working.Close()
		}
		working = preprocessed
	}
	defer func() {
		if working != gray {
			working.Close()
		}
	}()

	result, err := p.performIterativeTriclass(working, params)
	if err != nil {
		return nil, fmt.Errorf("iterative processing failed: %w", err)
	}

	if p.getBoolParam(params, "apply_cleanup") {
		cleaned, err := p.applyCleanup(result)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("cleanup failed: %w", err)
		}
		result.Close()
		result = cleaned
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