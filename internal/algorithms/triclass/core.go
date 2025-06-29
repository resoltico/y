package triclass

import (
	"context"
	"fmt"

	"otsu-obliterator/internal/opencv/safe"
	"otsu-obliterator/internal/processing/chain"
	"otsu-obliterator/internal/processing/filters"
	"otsu-obliterator/internal/processing/threshold"
)

type Processor struct {
	name          string
	preprocessor  *chain.ProcessingChain
	thresholdCalc *threshold.TriclassCalculator
	postprocessor *chain.ProcessingChain
}

func NewProcessor() *Processor {
	return &Processor{
		name:          "Iterative Triclass",
		preprocessor:  createTriclassPreprocessingChain(),
		thresholdCalc: threshold.NewTriclassCalculator(),
		postprocessor: createTriclassPostprocessingChain(),
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"initial_threshold_method": "otsu",
		"histogram_bins":           0,
		"convergence_precision":    1.0,
		"max_iterations":           8,
		"minimum_tbd_fraction":     0.01,
		"class_separation":         0.5,
		"preprocessing":            true,
		"result_cleanup":           true,
		"preserve_borders":         false,
		"noise_robustness":         true,
		"guided_filtering":         true,
		"guided_radius":            6,
		"guided_epsilon":           0.15,
		"parallel_processing":      true,
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if method, ok := params["initial_threshold_method"].(string); ok {
		if method != "otsu" && method != "mean" && method != "median" && method != "triangle" {
			return fmt.Errorf("initial_threshold_method must be 'otsu', 'mean', 'median', or 'triangle', got: %s", method)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins != 0 && (histBins < 8 || histBins > 256) {
			return fmt.Errorf("histogram_bins must be 0 (auto) or between 8 and 256, got: %d", histBins)
		}
	}

	if precision, ok := params["convergence_precision"].(float64); ok {
		if precision < 0.5 || precision > 2.0 {
			return fmt.Errorf("convergence_precision must be between 0.5 and 2.0, got: %f", precision)
		}
	}

	if maxIter, ok := params["max_iterations"].(int); ok {
		if maxIter < 3 || maxIter > 15 {
			return fmt.Errorf("max_iterations must be between 3 and 15, got: %d", maxIter)
		}
	}

	if fraction, ok := params["minimum_tbd_fraction"].(float64); ok {
		if fraction < 0.001 || fraction > 0.2 {
			return fmt.Errorf("minimum_tbd_fraction must be between 0.001 and 0.2, got: %f", fraction)
		}
	}

	if separation, ok := params["class_separation"].(float64); ok {
		if separation < 0.1 || separation > 0.8 {
			return fmt.Errorf("class_separation must be between 0.1 and 0.8, got: %f", separation)
		}
	}

	if radius, ok := params["guided_radius"].(int); ok {
		if radius < 1 || radius > 8 {
			return fmt.Errorf("guided_radius must be between 1 and 8, got: %d", radius)
		}
	}

	if epsilon, ok := params["guided_epsilon"].(float64); ok {
		if epsilon < 0.01 || epsilon > 0.5 {
			return fmt.Errorf("guided_epsilon must be between 0.01 and 0.5, got: %f", epsilon)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return p.ProcessWithContext(context.Background(), input, params)
}

func (p *Processor) ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(input, "Iterative Triclass processing"); err != nil {
		return nil, err
	}

	if err := p.ValidateParameters(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Apply preprocessing chain
	working, err := p.preprocessor.Execute(ctx, input, params)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}
	defer working.Close()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Perform iterative triclass segmentation
	result, err := p.thresholdCalc.ProcessIterative(ctx, working, params)
	if err != nil {
		return nil, fmt.Errorf("iterative processing failed: %w", err)
	}

	// Apply postprocessing if enabled
	shouldCleanup, ok := params["result_cleanup"].(bool)
	if !ok || !shouldCleanup {
		return result, nil
	}

	final, err := p.postprocessor.Execute(ctx, result, params)
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("postprocessing failed: %w", err)
	}
	result.Close()

	return final, nil
}

func createTriclassPreprocessingChain() *chain.ProcessingChain {
	return chain.NewProcessingChain([]chain.ProcessingStep{
		filters.NewGrayscaleConverter(),
		filters.NewGuidedFilter(),
		filters.NewNonLocalMeansFilter(),
	})
}

func createTriclassPostprocessingChain() *chain.ProcessingChain {
	return chain.NewProcessingChain([]chain.ProcessingStep{
		filters.NewMorphologyFilter(),
		filters.NewMedianFilter(),
	})
}