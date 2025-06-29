package filters

import (
	"context"
	"fmt"
	"image"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

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

	// Use NewCLAHEWithParams - the correct GoCV v0.41.0 API
	clahe := gocv.NewCLAHEWithParams(clipLimit, image.Point{X: tileSize, Y: tileSize})
	defer clahe.Close()

	srcMat := input.GetMat()
	dstMat := dst.GetMat()

	clahe.Apply(srcMat, &dstMat)

	return dst, nil
}
