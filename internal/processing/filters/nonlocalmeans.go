package filters

import (
	"context"
	"fmt"

	"gocv.io/x/gocv"
	"otsu-obliterator/internal/opencv/safe"
)

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
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()

	// Apply non-local means denoising with adaptive parameters
	// h=10 for moderate denoising, templateWindowSize=7, searchWindowSize=21
	gocv.FastNlMeansDenoisingWithParams(srcMat, &resultMat, 10.0, 7, 21)

	return result, nil
}

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
		return nil, fmt.Errorf("failed to create destination Mat: %w", err)
	}

	rows := src.Rows()
	cols := src.Cols()

	// Build integral image
	integral, err := n.buildIntegralImage(src)
	if err != nil {
		dst.Close()
		return nil, fmt.Errorf("failed to build integral image: %w", err)
	}
	defer integral.Close()

	// Calculate neighborhood means using integral image
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-halfWindow)
			x1 := max(0, x-halfWindow)
			y2 := min(rows-1, y+halfWindow)
			x2 := min(cols-1, x+halfWindow)

			area := int64((y2 - y1 + 1) * (x2 - x1 + 1))
			sum, _ := n.getIntegralSum(integral, y1, x1, y2, x2)
			mean := uint8(sum / float64(area))

			dst.SetUCharAt(y, x, mean)
		}
	}

	return dst, nil
}

func (n *NeighborhoodCalculator) buildIntegralImage(src *safe.Mat) (*safe.Mat, error) {
	rows := src.Rows()
	cols := src.Cols()

	integral, err := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	if err != nil {
		return nil, err
	}

	// Initialize first row and column to zero
	for i := 0; i <= rows; i++ {
		integral.SetDoubleAt(i, 0, 0.0)
	}
	for j := 0; j <= cols; j++ {
		integral.SetDoubleAt(0, j, 0.0)
	}

	// Build integral image
	for y := 1; y <= rows; y++ {
		for x := 1; x <= cols; x++ {
			pixelVal, _ := src.GetUCharAt(y-1, x-1)
			val := float64(pixelVal)

			prevRow, _ := integral.GetDoubleAt(y-1, x)
			prevCol, _ := integral.GetDoubleAt(y, x-1)
			prevDiag, _ := integral.GetDoubleAt(y-1, x-1)

			integral.SetDoubleAt(y, x, val+prevRow+prevCol-prevDiag)
		}
	}

	return integral, nil
}

func (n *NeighborhoodCalculator) getIntegralSum(integral *safe.Mat, y1, x1, y2, x2 int) (float64, error) {
	sum, err := integral.GetDoubleAt(y2+1, x2+1)
	if err != nil {
		return 0, err
	}

	val1, _ := integral.GetDoubleAt(y1, x2+1)
	val2, _ := integral.GetDoubleAt(y2+1, x1)
	val3, _ := integral.GetDoubleAt(y1, x1)

	return sum - val1 - val2 + val3, nil
}