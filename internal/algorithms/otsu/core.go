package otsu

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"sync"

	"otsu-obliterator/internal/opencv/safe"
	"otsu-obliterator/internal/processing/filters"
	"otsu-obliterator/internal/processing/histogram"
	"otsu-obliterator/internal/processing/threshold"

	"gocv.io/x/gocv"
)

type Processor struct {
	name       string
	workerPool chan struct{}
	matPool    sync.Pool
	mu         sync.RWMutex
}

func NewProcessor() *Processor {
	// Create worker pool sized for CPU count
	workers := make(chan struct{}, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		workers <- struct{}{}
	}

	return &Processor{
		name:       "2D Otsu",
		workerPool: workers,
		matPool: sync.Pool{
			New: func() interface{} {
				return &matPoolItem{}
			},
		},
	}
}

type matPoolItem struct {
	grayscale    *safe.Mat
	neighborhood *safe.Mat
	temporary    *safe.Mat
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"window_size":            7,
		"histogram_bins":         0, // Auto-calculate
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

	// Acquire worker from pool
	select {
	case <-p.workerPool:
		defer func() { p.workerPool <- struct{}{} }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Get pooled Mats for intermediate operations
	poolItem := p.matPool.Get().(*matPoolItem)
	defer p.matPool.Put(poolItem)

	return p.processInternal(ctx, input, params, poolItem)
}

func (p *Processor) processInternal(ctx context.Context, input *safe.Mat, params map[string]interface{}, poolItem *matPoolItem) (*safe.Mat, error) {
	// Step 1: Convert to grayscale with context check
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	grayscale, err := p.convertToGrayscale(input)
	if err != nil {
		return nil, fmt.Errorf("grayscale conversion failed: %w", err)
	}
	defer grayscale.Close()

	// Step 2: Apply preprocessing pipeline
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	preprocessed, err := p.applyPreprocessing(ctx, grayscale, params)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}
	defer preprocessed.Close()

	// Step 3: Calculate neighborhood means
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	neighborhood, err := p.calculateNeighborhoodMeans(preprocessed, params)
	if err != nil {
		return nil, fmt.Errorf("neighborhood calculation failed: %w", err)
	}
	defer neighborhood.Close()

	// Step 4: Build 2D histogram
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	histogramBuilder := histogram.NewTwoDimensionalBuilder()
	hist, err := histogramBuilder.Build(preprocessed, neighborhood, params)
	if err != nil {
		return nil, fmt.Errorf("histogram calculation failed: %w", err)
	}

	// Step 5: Calculate threshold
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	thresholdCalc := threshold.NewOtsu2DCalculator()
	thresholds, err := thresholdCalc.Calculate(hist)
	if err != nil {
		return nil, fmt.Errorf("threshold calculation failed: %w", err)
	}

	// Step 6: Apply threshold
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := p.applyThreshold(preprocessed, neighborhood, thresholds, params)
	if err != nil {
		return nil, fmt.Errorf("threshold application failed: %w", err)
	}

	// Step 7: Apply postprocessing
	select {
	case <-ctx.Done():
		result.Close()
		return nil, ctx.Err()
	default:
	}

	final, err := p.applyPostprocessing(ctx, result, params)
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("postprocessing failed: %w", err)
	}
	result.Close()

	return final, nil
}

func (p *Processor) convertToGrayscale(src *safe.Mat) (*safe.Mat, error) {
	if src.Channels() == 1 {
		return src.Clone()
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	switch src.Channels() {
	case 3:
		gocv.CvtColor(srcMat, &dstMat, gocv.ColorBGRToGray)
	case 4:
		tempBGR := gocv.NewMat()
		defer tempBGR.Close()
		gocv.CvtColor(srcMat, &tempBGR, gocv.ColorBGRAToBGR)
		gocv.CvtColor(tempBGR, &dstMat, gocv.ColorBGRToGray)
	default:
		dst.Close()
		return nil, fmt.Errorf("unsupported channel count for grayscale conversion: %d", src.Channels())
	}

	return dst, nil
}

func (p *Processor) applyPreprocessing(ctx context.Context, src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	current := src
	needsCleanup := false

	// Apply noise reduction if enabled
	if useNoise, ok := params["noise_robustness"].(bool); ok && useNoise {
		select {
		case <-ctx.Done():
			if needsCleanup {
				current.Close()
			}
			return nil, ctx.Err()
		default:
		}

		filtered, err := p.applyMAOTSUFilter(current)
		if err != nil {
			if needsCleanup {
				current.Close()
			}
			return nil, err
		}

		if needsCleanup {
			current.Close()
		}
		current = filtered
		needsCleanup = true
	}

	// Apply CLAHE if enabled
	if useCLAHE, ok := params["use_clahe"].(bool); ok && useCLAHE {
		select {
		case <-ctx.Done():
			if needsCleanup {
				current.Close()
			}
			return nil, ctx.Err()
		default:
		}

		enhanced, err := p.applyCLAHE(current, params)
		if err != nil {
			if needsCleanup {
				current.Close()
			}
			return nil, err
		}

		if needsCleanup {
			current.Close()
		}
		current = enhanced
		needsCleanup = true
	}

	// Apply Gaussian smoothing if enabled
	if useGaussian, ok := params["gaussian_preprocessing"].(bool); ok && useGaussian {
		select {
		case <-ctx.Done():
			if needsCleanup {
				current.Close()
			}
			return nil, ctx.Err()
		default:
		}

		smoothed, err := p.applyGaussianSmoothing(current, params)
		if err != nil {
			if needsCleanup {
				current.Close()
			}
			return nil, err
		}

		if needsCleanup {
			current.Close()
		}
		current = smoothed
		needsCleanup = true
	}

	if !needsCleanup {
		return src.Clone()
	}

	return current, nil
}

func (p *Processor) applyMAOTSUFilter(src *safe.Mat) (*safe.Mat, error) {
	// Apply median filter for noise reduction
	median := gocv.NewMat()
	defer median.Close()

	srcMat := src.GetMat()
	gocv.MedianBlur(srcMat, &median, 3)

	// Apply Gaussian for smoothing
	gaussian := gocv.NewMat()
	defer gaussian.Close()

	gocv.GaussianBlur(median, &gaussian, image.Point{X: 3, Y: 3}, 0.8, 0.8, gocv.BorderDefault)

	// Create result Mat
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	// Weighted combination: 60% median + 40% gaussian
	rows := src.Rows()
	cols := src.Cols()
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			medVal := median.GetUCharAt(y, x)
			gausVal := gaussian.GetUCharAt(y, x)

			combined := 0.6*float64(medVal) + 0.4*float64(gausVal)
			result.SetUCharAt(y, x, uint8(combined))
		}
	}

	return result, nil
}

func (p *Processor) applyCLAHE(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	clipLimit := 3.0
	if val, ok := params["clahe_clip_limit"].(float64); ok {
		clipLimit = val
	}

	tileSize := 8
	if val, ok := params["clahe_tile_size"].(int); ok {
		tileSize = val
	}

	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	clahe := gocv.NewCLAHEWithParams(clipLimit, image.Point{X: tileSize, Y: tileSize})
	defer clahe.Close()

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	clahe.Apply(srcMat, &resultMat)

	return result, nil
}

func (p *Processor) applyGaussianSmoothing(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	sigma := 1.0
	if val, ok := params["smoothing_strength"].(float64); ok {
		sigma = val
	}

	if sigma <= 0.0 {
		return src.Clone()
	}

	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}
	if kernelSize < 3 {
		kernelSize = 3
	}
	if kernelSize > 15 {
		kernelSize = 15
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	gocv.GaussianBlur(srcMat, &resultMat, image.Point{X: kernelSize, Y: kernelSize}, sigma, sigma, gocv.BorderDefault)

	return result, nil
}

func (p *Processor) calculateNeighborhoodMeans(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	windowSize := 7
	if val, ok := params["window_size"].(int); ok {
		windowSize = val
	}

	calc := filters.NewNeighborhoodCalculator(windowSize)
	return calc.Calculate(src)
}

func (p *Processor) applyThreshold(src, neighborhood *safe.Mat, thresholds [2]float64, params map[string]interface{}) (*safe.Mat, error) {
	applier := threshold.NewBilinearApplier()
	return applier.Apply(src, neighborhood, thresholds)
}

func (p *Processor) applyPostprocessing(ctx context.Context, src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	// Apply morphological operations for cleanup
	opened, err := p.applyMorphologicalOpening(src)
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Apply closing to fill gaps
	result, err := p.applyMorphologicalClosing(opened)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p *Processor) applyMorphologicalOpening(src *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	gocv.MorphologyEx(srcMat, &resultMat, gocv.MorphOpen, kernel)

	return result, nil
}

func (p *Processor) applyMorphologicalClosing(src *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 5, Y: 5})
	defer kernel.Close()

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	gocv.MorphologyEx(srcMat, &resultMat, gocv.MorphClose, kernel)

	return result, nil
}
