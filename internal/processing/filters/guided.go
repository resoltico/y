package filters

import (
	"context"
	"fmt"
	"math"

	"gocv.io/x/gocv"
	"otsu-obliterator/internal/opencv/safe"
)

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

	// Build integral images for efficient box filtering
	integralI, integralI2, integralP, integralIP, err := g.buildGuidedFilterIntegrals(src)
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("failed to build integral images: %w", err)
	}
	defer integralI.Close()
	defer integralI2.Close()
	defer integralP.Close()
	defer integralIP.Close()

	// Apply guided filter with box filter approximation
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			y1 := max(0, y-radius)
			x1 := max(0, x-radius)
			y2 := min(rows-1, y+radius)
			x2 := min(cols-1, x+radius)

			area := float64((y2 - y1 + 1) * (x2 - x1 + 1))

			// Calculate local statistics using integral images
			meanI, _ := g.getIntegralSum(integralI, y1, x1, y2, x2)
			meanI /= area
			
			meanI2, _ := g.getIntegralSum(integralI2, y1, x1, y2, x2)
			meanI2 /= area
			
			meanP, _ := g.getIntegralSum(integralP, y1, x1, y2, x2)
			meanP /= area
			
			meanIP, _ := g.getIntegralSum(integralIP, y1, x1, y2, x2)
			meanIP /= area

			varI := meanI2 - meanI*meanI
			covIP := meanIP - meanI*meanP

			a := covIP / (varI + epsilon)
			b := meanP - a*meanI

			pixelVal, _ := src.GetUCharAt(y, x)
			filteredVal := a*float64(pixelVal) + b

			result.SetUCharAt(y, x, uint8(math.Max(0, math.Min(255, filteredVal))))
		}
	}

	return result, nil
}

func (g *GuidedFilter) buildGuidedFilterIntegrals(src *safe.Mat) (*safe.Mat, *safe.Mat, *safe.Mat, *safe.Mat, error) {
	rows := src.Rows()
	cols := src.Cols()

	integralI, err := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	integralI2, err := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	if err != nil {
		integralI.Close()
		return nil, nil, nil, nil, err
	}

	integralP, err := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	if err != nil {
		integralI.Close()
		integralI2.Close()
		return nil, nil, nil, nil, err
	}

	integralIP, err := safe.NewMat(rows+1, cols+1, gocv.MatTypeCV64FC1)
	if err != nil {
		integralI.Close()
		integralI2.Close()
		integralP.Close()
		return nil, nil, nil, nil, err
	}

	// Initialize borders to zero
	for i := 0; i <= rows; i++ {
		integralI.SetDoubleAt(i, 0, 0.0)
		integralI2.SetDoubleAt(i, 0, 0.0)
		integralP.SetDoubleAt(i, 0, 0.0)
		integralIP.SetDoubleAt(i, 0, 0.0)
	}
	for j := 0; j <= cols; j++ {
		integralI.SetDoubleAt(0, j, 0.0)
		integralI2.SetDoubleAt(0, j, 0.0)
		integralP.SetDoubleAt(0, j, 0.0)
		integralIP.SetDoubleAt(0, j, 0.0)
	}

	// Build integral images
	for y := 1; y <= rows; y++ {
		for x := 1; x <= cols; x++ {
			pixelVal, _ := src.GetUCharAt(y-1, x-1)
			I := float64(pixelVal)
			P := I // For guided filter, guide image equals input image

			// Calculate integral sums using modern GetDoubleAt/SetDoubleAt
			prevRowI, _ := integralI.GetDoubleAt(y-1, x)
			prevColI, _ := integralI.GetDoubleAt(y, x-1)
			prevDiagI, _ := integralI.GetDoubleAt(y-1, x-1)
			integralI.SetDoubleAt(y, x, I+prevRowI+prevColI-prevDiagI)

			prevRowI2, _ := integralI2.GetDoubleAt(y-1, x)
			prevColI2, _ := integralI2.GetDoubleAt(y, x-1)
			prevDiagI2, _ := integralI2.GetDoubleAt(y-1, x-1)
			integralI2.SetDoubleAt(y, x, I*I+prevRowI2+prevColI2-prevDiagI2)

			prevRowP, _ := integralP.GetDoubleAt(y-1, x)
			prevColP, _ := integralP.GetDoubleAt(y, x-1)
			prevDiagP, _ := integralP.GetDoubleAt(y-1, x-1)
			integralP.SetDoubleAt(y, x, P+prevRowP+prevColP-prevDiagP)

			prevRowIP, _ := integralIP.GetDoubleAt(y-1, x)
			prevColIP, _ := integralIP.GetDoubleAt(y, x-1)
			prevDiagIP, _ := integralIP.GetDoubleAt(y-1, x-1)
			integralIP.SetDoubleAt(y, x, I*P+prevRowIP+prevColIP-prevDiagIP)
		}
	}

	return integralI, integralI2, integralP, integralIP, nil
}

func (g *GuidedFilter) getIntegralSum(integral *safe.Mat, y1, x1, y2, x2 int) (float64, error) {
	sum, err := integral.GetDoubleAt(y2+1, x2+1)
	if err != nil {
		return 0, err
	}
	
	val1, _ := integral.GetDoubleAt(y1, x2+1)
	val2, _ := integral.GetDoubleAt(y2+1, x1)
	val3, _ := integral.GetDoubleAt(y1, x1)
	
	return sum - val1 - val2 + val3, nil
}