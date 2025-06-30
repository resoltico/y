package filters

import (
	"context"
	"fmt"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

// GrayscaleConverter converts images to grayscale using modern GoCV v0.41.0 patterns
type GrayscaleConverter struct{}

// NewGrayscaleConverter creates a new grayscale converter
func NewGrayscaleConverter() *GrayscaleConverter {
	return &GrayscaleConverter{}
}

// Name returns the filter name
func (g *GrayscaleConverter) Name() string {
	return "grayscale_converter"
}

// ShouldExecute determines if the filter should run
func (g *GrayscaleConverter) ShouldExecute(params map[string]interface{}) bool {
	return true // Always convert to grayscale for processing
}

// Apply performs the grayscale conversion with GoCV v0.41.0 API
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

// convertToGrayscale performs the actual conversion using modern GoCV patterns
func (g *GrayscaleConverter) convertToGrayscale(src *safe.Mat) (*safe.Mat, error) {
	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("destination Mat creation failed: %w", err)
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	switch src.Channels() {
	case 3:
		gocv.CvtColor(srcMat, &dstMat, gocv.ColorBGRToGray)
	case 4:
		// Handle BGRA by first converting to BGR, then to grayscale
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

// ConvertToGrayscale provides a standalone conversion function for direct use
func ConvertToGrayscale(src *safe.Mat) (*safe.Mat, error) {
	converter := NewGrayscaleConverter()
	return converter.convertToGrayscale(src)
}