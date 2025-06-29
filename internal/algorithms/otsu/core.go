package otsu

import (
	"context"
	"fmt"
	"log"

	"otsu-obliterator/internal/opencv/safe"
	"otsu-obliterator/internal/processing/chain"
	"otsu-obliterator/internal/processing/filters"
	"otsu-obliterator/internal/processing/histogram"
	"otsu-obliterator/internal/processing/threshold"
)

type Processor struct {
	name          string
	preprocessor  *chain.ProcessingChain
	histogramCalc *histogram.TwoDimensionalBuilder
	thresholdCalc *threshold.Otsu2DCalculator
	postprocessor *chain.ProcessingChain
}

func NewProcessor() *Processor {
	return &Processor{
		name:          "2D Otsu",
		preprocessor:  createPreprocessingChain(),
		histogramCalc: histogram.NewTwoDimensionalBuilder(),
		thresholdCalc: threshold.NewOtsu2DCalculator(),
		postprocessor: createPostprocessingChain(),
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"window_size":            7,
		"histogram_bins":         0,
		"smoothing_strength":     1.0,
		"noise_robustness":       true,
		"gaussian_preprocessing": true,
		"use_clahe":              false,
		"clahe_clip_limit":       3.0,
		"clahe_tile_size":        8,
		"guided_filtering":       false,
		"guided_radius":          4,
		"guided_epsilon":         0.05,
		"parallel_processing":    true,
	}
}

func (p *Processor) ValidateParameters(params map[string]interface{}) error {
	if windowSize, ok := params["window_size"].(int); ok {
		if windowSize < 3 || windowSize > 21 || windowSize%2 == 0 {
			return fmt.Errorf("window_size must be odd number between 3 and 21, got: %d", windowSize)
		}
	}

	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins != 0 && (histBins < 8 || histBins > 256) {
			return fmt.Errorf("histogram_bins must be 0 (auto) or between 8 and 256, got: %d", histBins)
		}
	}

	if smoothing, ok := params["smoothing_strength"].(float64); ok {
		if smoothing < 0.0 || smoothing > 5.0 {
			return fmt.Errorf("smoothing_strength must be between 0.0 and 5.0, got: %f", smoothing)
		}
	}

	if clipLimit, ok := params["clahe_clip_limit"].(float64); ok {
		if clipLimit < 1.0 || clipLimit > 8.0 {
			return fmt.Errorf("clahe_clip_limit must be between 1.0 and 8.0, got: %f", clipLimit)
		}
	}

	if tileSize, ok := params["clahe_tile_size"].(int); ok {
		if tileSize < 4 || tileSize > 16 {
			return fmt.Errorf("clahe_tile_size must be between 4 and 16, got: %d", tileSize)
		}
	}

	if radius, ok := params["guided_radius"].(int); ok {
		if radius < 1 || radius > 8 {
			return fmt.Errorf("guided_radius must be between 1 and 8, got: %d", radius)
		}
	}

	if epsilon, ok := params["guided_epsilon"].(float64); ok {
		if epsilon < 0.001 || epsilon > 0.5 {
			return fmt.Errorf("guided_epsilon must be between 0.001 and 0.5, got: %f", epsilon)
		}
	}

	return nil
}

func (p *Processor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return p.ProcessWithContext(context.Background(), input, params)
}

func (p *Processor) ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	log.Printf("DEBUG: Starting ProcessWithContext")

	if err := safe.ValidateMatForOperation(input, "2D Otsu processing"); err != nil {
		log.Printf("DEBUG: Input validation failed: %v", err)
		return nil, err
	}
	log.Printf("DEBUG: Input Mat valid: %dx%d, channels=%d", input.Cols(), input.Rows(), input.Channels())

	if err := p.ValidateParameters(params); err != nil {
		log.Printf("DEBUG: Parameter validation failed: %v", err)
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	log.Printf("DEBUG: Parameters validated")

	// Apply preprocessing chain
	log.Printf("DEBUG: Starting preprocessing")
	processed, err := p.preprocessor.Execute(ctx, input, params)
	if err != nil {
		log.Printf("DEBUG: Preprocessing failed: %v", err)
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}
	if processed == nil {
		log.Printf("DEBUG: Preprocessing returned nil Mat")
		return nil, fmt.Errorf("preprocessing returned nil Mat")
	}
	if err := safe.ValidateMatForOperation(processed, "after preprocessing"); err != nil {
		log.Printf("DEBUG: Processed Mat invalid after preprocessing: %v", err)
		processed.Close()
		return nil, fmt.Errorf("preprocessing produced invalid Mat: %w", err)
	}
	log.Printf("DEBUG: Preprocessing completed: %dx%d, channels=%d", processed.Cols(), processed.Rows(), processed.Channels())
	defer processed.Close()

	// Calculate neighborhood means
	log.Printf("DEBUG: Calculating neighborhood means")
	neighborhood, err := p.calculateNeighborhoodMeans(processed, params)
	if err != nil {
		log.Printf("DEBUG: Neighborhood calculation failed: %v", err)
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	if neighborhood == nil {
		log.Printf("DEBUG: Neighborhood calculation returned nil")
		return nil, fmt.Errorf("neighborhood calculation returned nil")
	}
	if err := safe.ValidateMatForOperation(neighborhood, "neighborhood calculation"); err != nil {
		log.Printf("DEBUG: Neighborhood Mat invalid: %v", err)
		neighborhood.Close()
		return nil, fmt.Errorf("neighborhood calculation produced invalid Mat: %w", err)
	}
	log.Printf("DEBUG: Neighborhood calculation completed: %dx%d", neighborhood.Cols(), neighborhood.Rows())
	defer neighborhood.Close()

	// Build 2D histogram
	log.Printf("DEBUG: Building 2D histogram")
	hist, err := p.histogramCalc.Build(processed, neighborhood, params)
	if err != nil {
		log.Printf("DEBUG: Histogram calculation failed: %v", err)
		return nil, fmt.Errorf("histogram calculation failed: %w", err)
	}
	if hist == nil || len(hist) == 0 {
		log.Printf("DEBUG: Histogram calculation returned empty histogram")
		return nil, fmt.Errorf("histogram calculation returned empty histogram")
	}
	log.Printf("DEBUG: Histogram built: %dx%d bins", len(hist), len(hist[0]))

	// Calculate threshold
	log.Printf("DEBUG: Calculating threshold")
	threshold, err := p.thresholdCalc.Calculate(hist)
	if err != nil {
		log.Printf("DEBUG: Threshold calculation failed: %v", err)
		return nil, fmt.Errorf("threshold calculation failed: %w", err)
	}
	log.Printf("DEBUG: Threshold calculated: [%.3f, %.3f]", threshold[0], threshold[1])

	// Apply threshold
	log.Printf("DEBUG: Applying threshold")
	result, err := p.applyThreshold(processed, neighborhood, threshold, params)
	if err != nil {
		log.Printf("DEBUG: Threshold application failed: %v", err)
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}
	if result == nil {
		log.Printf("DEBUG: Threshold application returned nil")
		return nil, fmt.Errorf("threshold application returned nil")
	}
	if err := safe.ValidateMatForOperation(result, "after threshold"); err != nil {
		log.Printf("DEBUG: Result Mat invalid after threshold: %v", err)
		result.Close()
		return nil, fmt.Errorf("threshold application produced invalid Mat: %w", err)
	}
	log.Printf("DEBUG: Threshold applied: %dx%d", result.Cols(), result.Rows())

	// Apply postprocessing
	log.Printf("DEBUG: Starting postprocessing")
	final, err := p.postprocessor.Execute(ctx, result, params)
	if err != nil {
		log.Printf("DEBUG: Postprocessing failed: %v", err)
		result.Close()
		return nil, fmt.Errorf("postprocessing failed: %w", err)
	}
	result.Close()

	if final == nil {
		log.Printf("DEBUG: Postprocessing returned nil")
		return nil, fmt.Errorf("postprocessing returned nil")
	}
	if err := safe.ValidateMatForOperation(final, "final result"); err != nil {
		log.Printf("DEBUG: Final Mat invalid: %v", err)
		final.Close()
		return nil, fmt.Errorf("postprocessing produced invalid Mat: %w", err)
	}
	log.Printf("DEBUG: Processing completed successfully: %dx%d", final.Cols(), final.Rows())

	return final, nil
}

func createPreprocessingChain() *chain.ProcessingChain {
	return chain.NewProcessingChain([]chain.ProcessingStep{
		filters.NewGrayscaleConverter(),
		filters.NewCLAHEFilter(),
		filters.NewGuidedFilter(),
		filters.NewGaussianFilter(),
		filters.NewMAOTSUFilter(),
	})
}

func createPostprocessingChain() *chain.ProcessingChain {
	return chain.NewProcessingChain([]chain.ProcessingStep{
		filters.NewMorphologyFilter(),
		filters.NewMedianFilter(),
	})
}

func (p *Processor) calculateNeighborhoodMeans(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	windowSize, ok := params["window_size"].(int)
	if !ok {
		windowSize = 7
	}

	calc := filters.NewNeighborhoodCalculator(windowSize)
	return calc.Calculate(src)
}

func (p *Processor) applyThreshold(src, neighborhood *safe.Mat, thresholds [2]float64, params map[string]interface{}) (*safe.Mat, error) {
	applier := threshold.NewBilinearApplier()
	return applier.Apply(src, neighborhood, thresholds)
}
