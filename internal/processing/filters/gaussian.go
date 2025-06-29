package filters

import (
	"context"
	"fmt"
	"image"

	"gocv.io/x/gocv"
	"otsu-obliterator/internal/opencv/safe"
)

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