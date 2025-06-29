package triclass

import (
	"context"
	"fmt"
	"image"
	"math"
	"runtime"
	"sync"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

type Processor struct {
	name       string
	workerPool chan struct{}
	mu         sync.RWMutex
}

func NewProcessor() *Processor {
	// Create worker pool for parallel processing
	workers := make(chan struct{}, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		workers <- struct{}{}
	}

	return &Processor{
		name:       "Iterative Triclass",
		workerPool: workers,
	}
}

func (p *Processor) GetName() string {
	return p.name
}

func (p *Processor) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"initial_threshold_method": "otsu",
		"histogram_bins":           0, // Auto-calculate
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
		validMethods := map[string]bool{"otsu": true, "mean": true, "median": true, "triangle": true}
		if !validMethods[method] {
			return fmt.Errorf("initial_threshold_method must be one of: otsu, mean, median, triangle, got: %s", method)
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

	// Acquire worker from pool
	select {
	case <-p.workerPool:
		defer func() { p.workerPool <- struct{}{} }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return p.processIterativeTriclass(ctx, input, params)
}

func (p *Processor) processIterativeTriclass(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	// Step 1: Apply preprocessing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	working, err := p.applyPreprocessing(input, params)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}
	defer working.Close()

	// Step 2: Perform iterative triclass segmentation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	result, err := p.performIterativeSegmentation(ctx, working, params)
	if err != nil {
		return nil, fmt.Errorf("iterative segmentation failed: %w", err)
	}

	// Step 3: Apply cleanup if enabled
	if shouldCleanup, ok := params["result_cleanup"].(bool); ok && shouldCleanup {
		select {
		case <-ctx.Done():
			result.Close()
			return nil, ctx.Err()
		default:
		}

		cleaned, err := p.applyPostprocessing(result, params)
		if err != nil {
			result.Close()
			return nil, fmt.Errorf("postprocessing failed: %w", err)
		}
		result.Close()
		result = cleaned
	}

	return result, nil
}

func (p *Processor) applyPreprocessing(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	current := input
	needsCleanup := false

	// Convert to grayscale
	if input.Channels() > 1 {
		grayscale, err := p.convertToGrayscale(input)
		if err != nil {
			return nil, err
		}
		current = grayscale
		needsCleanup = true
	}

	// Apply noise reduction if enabled
	if useNoise, ok := params["noise_robustness"].(bool); ok && useNoise {
		denoised, err := p.applyNonLocalMeansDenoising(current)
		if err != nil {
			if needsCleanup {
				current.Close()
			}
			return nil, err
		}

		if needsCleanup {
			current.Close()
		}
		current = denoised
		needsCleanup = true
	}

	// Apply guided filtering if enabled
	if useGuided, ok := params["guided_filtering"].(bool); ok && useGuided {
		filtered, err := p.applyGuidedFiltering(current, params)
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

	if !needsCleanup {
		return input.Clone()
	}

	return current, nil
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
		return nil, fmt.Errorf("unsupported channel count: %d", src.Channels())
	}

	return dst, nil
}

func (p *Processor) applyNonLocalMeansDenoising(src *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()

	// Apply non-local means denoising with moderate parameters
	gocv.FastNlMeansDenoisingWithParams(srcMat, &resultMat, 10.0, 7, 21)

	return result, nil
}

func (p *Processor) applyGuidedFiltering(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	radius := 6
	if val, ok := params["guided_radius"].(int); ok {
		radius = val
	}

	epsilon := 0.15
	if val, ok := params["guided_epsilon"].(float64); ok {
		epsilon = val
	}

	return p.performGuidedFilter(src, radius, epsilon)
}

func (p *Processor) performGuidedFilter(src *safe.Mat, radius int, epsilon float64) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()

	// Simple box filter approximation of guided filter
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-radius)
			x1 := max(0, x-radius)
			y2 := min(rows-1, y+radius)
			x2 := min(cols-1, x+radius)

			// Calculate local mean
			sum := 0.0
			count := 0.0
			for yy := y1; yy <= y2; yy++ {
				for xx := x1; xx <= x2; xx++ {
					val, _ := src.GetUCharAt(yy, xx)
					sum += float64(val)
					count++
				}
			}

			mean := sum / count
			centerVal, _ := src.GetUCharAt(y, x)

			// Apply guided filter formula (simplified)
			filtered := (1.0-epsilon)*mean + epsilon*float64(centerVal)
			result.SetUCharAt(y, x, uint8(math.Max(0, math.Min(255, filtered))))
		}
	}

	return result, nil
}

func (p *Processor) performIterativeSegmentation(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	maxIterations := p.getIntParam(params, "max_iterations", 8)
	convergencePrecision := p.getFloatParam(params, "convergence_precision", 1.0)
	minTBDFraction := p.getFloatParam(params, "minimum_tbd_fraction", 0.01)

	result, err := safe.NewMat(input.Rows(), input.Cols(), input.Type())
	if err != nil {
		return nil, err
	}

	currentRegion, err := input.Clone()
	if err != nil {
		result.Close()
		return nil, err
	}
	defer currentRegion.Close()

	previousThreshold := -1.0
	totalPixels := float64(currentRegion.Rows() * currentRegion.Cols())

	for iteration := 0; iteration < maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			result.Close()
			return nil, ctx.Err()
		default:
		}

		// Check if region has pixels to process
		nonZeroPixels := p.countNonZeroPixels(currentRegion)
		if nonZeroPixels == 0 {
			break
		}

		// Calculate threshold for current region
		threshold := p.calculateThreshold(currentRegion, params)

		// Check convergence
		convergence := math.Abs(threshold - previousThreshold)
		if convergence < convergencePrecision && iteration > 0 {
			break
		}
		previousThreshold = threshold

		// Segment current region into foreground, background, and TBD
		foreground, background, tbd, err := p.segmentRegion(currentRegion, threshold, params)
		if err != nil {
			result.Close()
			return nil, err
		}

		// Update result with foreground pixels
		p.updateResult(result, foreground)

		foreground.Close()
		background.Close()

		// Check TBD fraction
		tbdCount := p.countNonZeroPixels(tbd)
		tbdFraction := float64(tbdCount) / totalPixels

		if tbdFraction < minTBDFraction {
			tbd.Close()
			break
		}

		// Extract TBD region for next iteration
		newRegion, err := p.extractTBDRegion(input, tbd)
		tbd.Close()
		if err != nil {
			result.Close()
			return nil, err
		}

		currentRegion.Close()
		currentRegion = newRegion
	}

	return result, nil
}

func (p *Processor) calculateThreshold(region *safe.Mat, params map[string]interface{}) float64 {
	method := p.getStringParam(params, "initial_threshold_method", "otsu")
	histogram := p.buildHistogram(region)

	switch method {
	case "mean":
		return p.calculateMeanThreshold(histogram)
	case "median":
		return p.calculateMedianThreshold(histogram)
	case "triangle":
		return p.calculateTriangleThreshold(histogram)
	default:
		return p.calculateOtsuThreshold(histogram)
	}
}

func (p *Processor) buildHistogram(src *safe.Mat) []int {
	histogram := make([]int, 256)
	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val, err := src.GetUCharAt(y, x)
			if err == nil && val > 0 {
				histogram[val]++
			}
		}
	}

	return histogram
}

func (p *Processor) calculateOtsuThreshold(histogram []int) float64 {
	total := 0
	for _, count := range histogram {
		total += count
	}

	if total == 0 {
		return 127.5
	}

	sum := 0.0
	for i, count := range histogram {
		sum += float64(i) * float64(count)
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5

	for i := 0; i < 256; i++ {
		wB += histogram[i]
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += float64(i) * float64(histogram[i])
		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)

		varBetween := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = float64(i)
		}
	}

	return bestThreshold
}

func (p *Processor) calculateMeanThreshold(histogram []int) float64 {
	totalPixels := 0
	weightedSum := 0.0

	for i, count := range histogram {
		totalPixels += count
		weightedSum += float64(i) * float64(count)
	}

	if totalPixels == 0 {
		return 127.5
	}

	return weightedSum / float64(totalPixels)
}

func (p *Processor) calculateMedianThreshold(histogram []int) float64 {
	totalPixels := 0
	for _, count := range histogram {
		totalPixels += count
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := totalPixels / 2
	cumSum := 0

	for i, count := range histogram {
		cumSum += count
		if cumSum >= halfPixels {
			return float64(i)
		}
	}

	return 127.5
}

func (p *Processor) calculateTriangleThreshold(histogram []int) float64 {
	// Find peak
	maxCount := 0
	peakIndex := 0
	for i, count := range histogram {
		if count > maxCount {
			maxCount = count
			peakIndex = i
		}
	}

	// Find endpoints
	leftEnd := 0
	rightEnd := 255

	for i := 0; i < 256; i++ {
		if histogram[i] > 0 {
			leftEnd = i
			break
		}
	}

	for i := 255; i >= 0; i-- {
		if histogram[i] > 0 {
			rightEnd = i
			break
		}
	}

	// Use the farther endpoint for triangle method
	farEnd := rightEnd
	if peakIndex-leftEnd > rightEnd-peakIndex {
		farEnd = leftEnd
	}

	// Find point with maximum distance to line
	maxDistance := 0.0
	bestThreshold := float64(peakIndex)

	x1, y1 := float64(peakIndex), float64(maxCount)
	x2, y2 := float64(farEnd), float64(histogram[farEnd])

	if x1 != x2 {
		for i := min(peakIndex, farEnd); i <= max(peakIndex, farEnd); i++ {
			distance := math.Abs((y2-y1)*float64(i)-(x2-x1)*float64(histogram[i])+x2*y1-y2*x1) /
				math.Sqrt((y2-y1)*(y2-y1)+(x2-x1)*(x2-x1))

			if distance > maxDistance {
				maxDistance = distance
				bestThreshold = float64(i)
			}
		}
	}

	return bestThreshold
}

func (p *Processor) segmentRegion(region *safe.Mat, threshold float64, params map[string]interface{}) (*safe.Mat, *safe.Mat, *safe.Mat, error) {
	rows := region.Rows()
	cols := region.Cols()

	foreground, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		return nil, nil, nil, err
	}

	background, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		return nil, nil, nil, err
	}

	tbd, err := safe.NewMat(rows, cols, region.Type())
	if err != nil {
		foreground.Close()
		background.Close()
		return nil, nil, nil, err
	}

	classSeparation := p.getFloatParam(params, "class_separation", 0.5)
	lowerThreshold := threshold * (1.0 - classSeparation)
	upperThreshold := threshold * (1.0 + classSeparation)

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

func (p *Processor) updateResult(result, foregroundMask *safe.Mat) {
	rows := result.Rows()
	cols := result.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := foregroundMask.GetUCharAt(y, x); err == nil && value > 0 {
				result.SetUCharAt(y, x, 255)
			}
		}
	}
}

func (p *Processor) extractTBDRegion(original, tbdMask *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(original.Rows(), original.Cols(), original.Type())
	if err != nil {
		return nil, err
	}

	rows := original.Rows()
	cols := original.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdValue, err := tbdMask.GetUCharAt(y, x); err == nil && tbdValue > 0 {
				if origValue, err := original.GetUCharAt(y, x); err == nil {
					result.SetUCharAt(y, x, origValue)
				}
			}
		}
	}

	return result, nil
}

func (p *Processor) countNonZeroPixels(mat *safe.Mat) int {
	rows := mat.Rows()
	cols := mat.Cols()
	count := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := mat.GetUCharAt(y, x); err == nil && value > 0 {
				count++
			}
		}
	}

	return count
}

func (p *Processor) applyPostprocessing(src *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	// Apply morphological opening
	opened, err := p.applyMorphologicalOperation(src, gocv.MorphOpen, 3)
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	// Apply morphological closing
	result, err := p.applyMorphologicalOperation(opened, gocv.MorphClose, 5)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p *Processor) applyMorphologicalOperation(src *safe.Mat, op gocv.MorphType, kernelSize int) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: kernelSize, Y: kernelSize})
	defer kernel.Close()

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	gocv.MorphologyEx(srcMat, &resultMat, op, kernel)

	return result, nil
}

// Helper functions
func (p *Processor) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (p *Processor) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (p *Processor) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
