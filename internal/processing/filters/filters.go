package filters

import (
	"context"
	"fmt"
	"image"
	"math"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

// GrayscaleConverter converts images to grayscale using modern GoCV patterns
type GrayscaleConverter struct{}

func NewGrayscaleConverter() *GrayscaleConverter {
	return &GrayscaleConverter{}
}

func (g *GrayscaleConverter) Name() string {
	return "grayscale_converter"
}

func (g *GrayscaleConverter) ShouldExecute(params map[string]interface{}) bool {
	return true // Always convert to grayscale for processing
}

func (g *GrayscaleConverter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if input.Channels() == 1 {
		return input.Clone()
	}

	return g.convertToGrayscale(input)
}

func (g *GrayscaleConverter) convertToGrayscale(src *safe.Mat) (*safe.Mat, error) {
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

// CLAHEFilter applies Contrast Limited Adaptive Histogram Equalization
type CLAHEFilter struct{}

func NewCLAHEFilter() *CLAHEFilter {
	return &CLAHEFilter{}
}

func (c *CLAHEFilter) Name() string {
	return "clahe_filter"
}

func (c *CLAHEFilter) ShouldExecute(params map[string]interface{}) bool {
	useClahe, ok := params["use_clahe"].(bool)
	return ok && useClahe
}

func (c *CLAHEFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	clipLimit := 3.0
	if val, ok := params["clahe_clip_limit"].(float64); ok {
		clipLimit = val
	}

	tileSize := 8
	if val, ok := params["clahe_tile_size"].(int); ok {
		tileSize = val
	}

	dst, err := safe.NewMat(input.Rows(), input.Cols(), input.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	clahe := gocv.NewCLAHEWithParams(clipLimit, image.Point{X: tileSize, Y: tileSize})
	defer clahe.Close()

	srcMat := input.GetMat()
	dstMat := dst.GetMat()
	clahe.Apply(srcMat, &dstMat)

	return dst, nil
}

// GuidedFilter applies guided filtering for edge-preserving smoothing
type GuidedFilter struct{}

func NewGuidedFilter() *GuidedFilter {
	return &GuidedFilter{}
}

func (g *GuidedFilter) Name() string {
	return "guided_filter"
}

func (g *GuidedFilter) ShouldExecute(params map[string]interface{}) bool {
	useGuided, ok := params["guided_filtering"].(bool)
	return ok && useGuided
}

func (g *GuidedFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	radius := 4
	if val, ok := params["guided_radius"].(int); ok {
		radius = val
	}

	epsilon := 0.05
	if val, ok := params["guided_epsilon"].(float64); ok {
		epsilon = val
	}

	return g.applyGuidedFilter(input, radius, epsilon)
}

func (g *GuidedFilter) applyGuidedFilter(src *safe.Mat, radius int, epsilon float64) (*safe.Mat, error) {
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
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

// GaussianFilter applies Gaussian smoothing
type GaussianFilter struct{}

func NewGaussianFilter() *GaussianFilter {
	return &GaussianFilter{}
}

func (g *GaussianFilter) Name() string {
	return "gaussian_filter"
}

func (g *GaussianFilter) ShouldExecute(params map[string]interface{}) bool {
	useGaussian, ok := params["gaussian_preprocessing"].(bool)
	return ok && useGaussian
}

func (g *GaussianFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	sigma := 1.0
	if val, ok := params["smoothing_strength"].(float64); ok {
		sigma = val
	}

	if sigma <= 0.0 {
		return input.Clone()
	}

	return g.applyGaussianBlur(input, sigma)
}

func (g *GaussianFilter) applyGaussianBlur(src *safe.Mat, sigma float64) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	kernelSize := int(sigma*6) + 1
	if kernelSize%2 == 0 {
		kernelSize++
	}
	kernelSize = max(3, min(kernelSize, 15))

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	
	gocv.GaussianBlur(srcMat, &dstMat, image.Point{X: kernelSize, Y: kernelSize}, sigma, sigma, gocv.BorderDefault)

	return dst, nil
}

// MAOTSUFilter applies MAOTSU noise reduction
type MAOTSUFilter struct{}

func NewMAOTSUFilter() *MAOTSUFilter {
	return &MAOTSUFilter{}
}

func (m *MAOTSUFilter) Name() string {
	return "maotsu_filter"
}

func (m *MAOTSUFilter) ShouldExecute(params map[string]interface{}) bool {
	useMAOTSU, ok := params["noise_robustness"].(bool)
	return ok && useMAOTSU
}

func (m *MAOTSUFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return m.applyMAOTSUPreprocessing(input)
}

func (m *MAOTSUFilter) applyMAOTSUPreprocessing(src *safe.Mat) (*safe.Mat, error) {
	// Apply median filter for impulse noise removal
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

// MorphologyFilter applies morphological operations for cleanup
type MorphologyFilter struct{}

func NewMorphologyFilter() *MorphologyFilter {
	return &MorphologyFilter{}
}

func (m *MorphologyFilter) Name() string {
	return "morphology_filter"
}

func (m *MorphologyFilter) ShouldExecute(params map[string]interface{}) bool {
	cleanup, ok := params["result_cleanup"].(bool)
	return ok && cleanup
}

func (m *MorphologyFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return m.applyMorphologicalCleanup(input)
}

func (m *MorphologyFilter) applyMorphologicalCleanup(src *safe.Mat) (*safe.Mat, error) {
	// Opening operation to remove small noise
	opened, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	kernel3 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel3.Close()

	srcMat := src.GetMat()
	openedMat := opened.GetMat()
	gocv.MorphologyEx(srcMat, &openedMat, gocv.MorphOpen, kernel3)

	// Closing operation to fill small gaps
	result, err := safe.NewMat(opened.Rows(), opened.Cols(), opened.Type())
	if err != nil {
		return nil, err
	}

	kernel5 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 5, Y: 5})
	defer kernel5.Close()

	resultMat := result.GetMat()
	gocv.MorphologyEx(openedMat, &resultMat, gocv.MorphClose, kernel5)

	return result, nil
}

// MedianFilter applies median filtering for noise reduction
type MedianFilter struct{}

func NewMedianFilter() *MedianFilter {
	return &MedianFilter{}
}

func (m *MedianFilter) Name() string {
	return "median_filter"
}

func (m *MedianFilter) ShouldExecute(params map[string]interface{}) bool {
	cleanup, ok := params["result_cleanup"].(bool)
	return ok && cleanup
}

func (m *MedianFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return m.applyMedianFiltering(input)
}

func (m *MedianFilter) applyMedianFiltering(src *safe.Mat) (*safe.Mat, error) {
	kernelSize := 3
	rows := src.Rows()
	cols := src.Cols()
	if rows*cols > 1000000 {
		kernelSize = 5
	}

	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	gocv.MedianBlur(srcMat, &resultMat, kernelSize)

	return result, nil
}

// NonLocalMeansFilter applies non-local means denoising
type NonLocalMeansFilter struct{}

func NewNonLocalMeansFilter() *NonLocalMeansFilter {
	return &NonLocalMeansFilter{}
}

func (n *NonLocalMeansFilter) Name() string {
	return "non_local_means_filter"
}

func (n *NonLocalMeansFilter) ShouldExecute(params map[string]interface{}) bool {
	useNLM, ok := params["noise_robustness"].(bool)
	return ok && useNLM
}

func (n *NonLocalMeansFilter) Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return n.applyNonLocalMeansDenoising(input)
}

func (n *NonLocalMeansFilter) applyNonLocalMeansDenoising(src *safe.Mat) (*safe.Mat, error) {
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

// NeighborhoodCalculator calculates neighborhood mean values
type NeighborhoodCalculator struct {
	windowSize int
}

func NewNeighborhoodCalculator(windowSize int) *NeighborhoodCalculator {
	return &NeighborhoodCalculator{
		windowSize: windowSize,
	}
}

func (n *NeighborhoodCalculator) Calculate(src *safe.Mat) (*safe.Mat, error) {
	halfWindow := n.windowSize / 2

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-halfWindow)
			x1 := max(0, x-halfWindow)
			y2 := min(rows-1, y+halfWindow)
			x2 := min(cols-1, x+halfWindow)

			sum := 0.0
			count := 0.0
			for yy := y1; yy <= y2; yy++ {
				for xx := x1; xx <= x2; xx++ {
					val, _ := src.GetUCharAt(yy, xx)
					sum += float64(val)
					count++
				}
			}

			mean := uint8(sum / count)
			dst.SetUCharAt(y, x, mean)
		}
	}

	return dst, nil
}

// Utility functions
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