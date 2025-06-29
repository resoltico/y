package filters

import (
	"context"
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"
	"otsu-obliterator/internal/opencv/safe"
)

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

// applyMAOTSUPreprocessing implements Median-Average Otsu preprocessing
// Combines median filtering (impulse noise removal) with average filtering (spatial correlation)
func (m *MAOTSUFilter) applyMAOTSUPreprocessing(src *safe.Mat) (*safe.Mat, error) {
	median, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create median Mat: %w", err)
	}
	defer median.Close()

	// Apply median filter for impulse noise removal
	srcMat := src.GetMat()
	medianMat := median.GetMat()
	gocv.MedianBlur(srcMat, &medianMat, 3)

	// Apply Gaussian filter for spatial correlation
	gaussian, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create gaussian Mat: %w", err)
	}
	defer gaussian.Close()

	gaussianMat := gaussian.GetMat()
	gocv.GaussianBlur(medianMat, &gaussianMat, image.Point{X: 3, Y: 3}, 0.8, 0.8, gocv.BorderDefault)

	// Weighted combination: 60% median (noise reduction) + 40% gaussian (smoothing)
	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	m.combineMatricesWeighted(median, gaussian, result, 0.6, 0.4)

	return result, nil
}

func (m *MAOTSUFilter) combineMatricesWeighted(mat1, mat2, result *safe.Mat, weight1, weight2 float64) {
	rows := mat1.Rows()
	cols := mat1.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val1, _ := mat1.GetUCharAt(y, x)
			val2, _ := mat2.GetUCharAt(y, x)

			combined := weight1*float64(val1) + weight2*float64(val2)
			result.SetUCharAt(y, x, uint8(math.Max(0, math.Min(255, combined))))
		}
	}
}