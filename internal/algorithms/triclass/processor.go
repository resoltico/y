package triclass

import (
	"context"
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
		"quality":                  "Fast",
		"initial_threshold_method": "otsu",
		"histogram_bins":           64,
		"convergence_epsilon":      1.0,
		"max_iterations":           10,
		"minimum_tbd_fraction":     0.01,
		"lower_upper_gap_factor":   0.5,
		"apply_preprocessing":      false,
		"apply_cleanup":            true,
		"preserve_borders":         false,
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
		working = preprocessed
		defer preprocessed.Close()
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	quality := p.getStringParam(params, "quality")
	var result *safe.Mat

	if quality == "Best" {
		result, err = p.performIterativeTriclassFloat(ctx, working, params)
	} else {
		result, err = p.performIterativeTriclass(ctx, working, params)
	}

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

func (p *Processor) performIterativeTriclassFloat(ctx context.Context, working *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	maxIterations := p.getIntParam(params, "max_iterations")
	convergenceEpsilon := p.getFloatParam(params, "convergence_epsilon")
	minTBDFraction := p.getFloatParam(params, "minimum_tbd_fraction")

	result, err := safe.NewMat(working.Rows(), working.Cols(), working.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	currentRegion, err := working.Clone()
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("failed to clone working Mat: %w", err)
	}
	defer currentRegion.Close()

	previousThreshold := -1.0
	totalPixels := float64(currentRegion.Rows() * currentRegion.Cols())

	for iteration := 0; iteration < maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		nonZeroPixels := p.countNonZeroPixels(currentRegion)
		if nonZeroPixels == 0 {
			break
		}

		threshold := p.calculateThresholdForRegionFloat(currentRegion, params)

		convergence := abs(threshold - previousThreshold)
		if previousThreshold >= 0 && convergence < convergenceEpsilon {
			break
		}
		previousThreshold = threshold

		foregroundMask, backgroundMask, tbdMask, err := p.segmentRegionFloat(currentRegion, threshold, params)
		if err != nil {
			return nil, fmt.Errorf("segmentation failed at iteration %d: %w", iteration, err)
		}

		tbdCount := p.countNonZeroPixels(tbdMask)
		tbdFraction := float64(tbdCount) / totalPixels

		p.updateResult(result, foregroundMask)

		foregroundMask.Close()
		backgroundMask.Close()

		if tbdFraction < minTBDFraction {
			tbdMask.Close()
			break
		}

		newRegion, err := p.extractTBDRegion(working, tbdMask)
		tbdMask.Close()
		if err != nil {
			return nil, fmt.Errorf("TBD region extraction failed: %w", err)
		}

		currentRegion.Close()
		currentRegion = newRegion
	}

	return result, nil
}

func (p *Processor) calculateThresholdForRegionFloat(region *safe.Mat, params map[string]interface{}) float64 {
	method := p.getStringParam(params, "initial_threshold_method")
	histBins := p.getIntParam(params, "histogram_bins")

	histogram := p.calculateHistogram(region, histBins)

	switch method {
	case "mean":
		return p.calculateMeanThresholdFloat(histogram, histBins)
	case "median":
		return p.calculateMedianThresholdFloat(histogram, histBins)
	default:
		return p.calculateOtsuThresholdFloat(histogram, histBins)
	}
}

func (p *Processor) calculateOtsuThresholdFloat(histogram []int, histBins int) float64 {
	total := 0
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return 127.5
	}

	sum := 0.0
	for i := 0; i < histBins; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5
	invTotal := 1.0 / float64(total)
	binToValue := 255.0 / float64(histBins-1)

	subPixelStep := 0.1
	for t := 0.0; t < float64(histBins); t += subPixelStep {
		tInt := int(t)
		if tInt >= histBins {
			break
		}

		wB += histogram[tInt]
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += float64(tInt) * float64(histogram[tInt])

		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)
		meanDiff := mB - mF

		varBetween := float64(wB) * float64(wF) * invTotal * meanDiff * meanDiff

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = t * binToValue
		}
	}

	return bestThreshold
}

func (p *Processor) calculateMeanThresholdFloat(histogram []int, histBins int) float64 {
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

func (p *Processor) calculateMedianThresholdFloat(histogram []int, histBins int) float64 {
	totalPixels := 0
	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := float64(totalPixels) / 2.0
	cumSum := 0.0

	for i := 0; i < histBins; i++ {
		cumSum += float64(histogram[i])
		if cumSum >= halfPixels {
			interpolationFactor := (cumSum - halfPixels) / float64(histogram[i])
			return (float64(i) - interpolationFactor) * 255.0 / float64(histBins-1)
		}
	}

	return 127.5
}

func (p *Processor) segmentRegionFloat(region *safe.Mat, threshold float64, params map[string]interface{}) (*safe.Mat, *safe.Mat, *safe.Mat, error) {
	rows := region.Rows()
	cols := region.Cols()

	foreground, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create foreground Mat: %w", err)
	}

	background, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		return nil, nil, nil, fmt.Errorf("failed to create background Mat: %w", err)
	}

	tbd, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		background.Close()
		return nil, nil, nil, fmt.Errorf("failed to create TBD Mat: %w", err)
	}

	gapFactor := p.getFloatParam(params, "lower_upper_gap_factor")
	lowerThreshold := threshold * (1.0 - gapFactor)
	upperThreshold := threshold * (1.0 + gapFactor)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := region.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			if pixelValue > 0 {
				pixelFloat := float64(pixelValue)
				if pixelFloat > upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if pixelFloat < lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd, nil
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
