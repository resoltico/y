package filters

import (
	"context"
	"fmt"
	"image"

	"gocv.io/x/gocv"
	"otsu-obliterator/internal/opencv/safe"
)

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
	// Adaptive kernel size based on image dimensions
	rows := src.Rows()
	cols := src.Cols()

	smallKernelSize := 3
	largeKernelSize := 5

	// Adjust kernel sizes for large images
	if rows*cols > 1000000 {
		smallKernelSize = 5
		largeKernelSize = 7
	}

	kernel3 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: smallKernelSize, Y: smallKernelSize})
	defer kernel3.Close()

	kernel5 := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: largeKernelSize, Y: largeKernelSize})
	defer kernel5.Close()

	// Opening operation to remove small noise
	opened, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create opened Mat: %w", err)
	}
	defer opened.Close()

	srcMat := src.GetMat()
	openedMat := opened.GetMat()
	gocv.MorphologyEx(srcMat, &openedMat, gocv.MorphOpen, kernel3)

	// Closing operation to fill small gaps
	result, err := safe.NewMat(opened.Rows(), opened.Cols(), opened.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	resultMat := result.GetMat()
	gocv.MorphologyEx(openedMat, &resultMat, gocv.MorphClose, kernel5)

	return result, nil
}

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
	// Adaptive kernel size based on image dimensions
	rows := src.Rows()
	cols := src.Cols()

	kernelSize := 3
	if rows*cols > 1000000 {
		kernelSize = 5
	}

	result, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	srcMat := src.GetMat()
	resultMat := result.GetMat()
	gocv.MedianBlur(srcMat, &resultMat, kernelSize)

	return result, nil
}