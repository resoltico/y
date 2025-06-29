package filters

import (
	"context"

	"otsu-obliterator/internal/opencv/conversion"
	"otsu-obliterator/internal/opencv/safe"
)

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

	return conversion.ConvertToGrayscale(input)
}