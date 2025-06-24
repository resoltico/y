package algorithms

import (
	"otsu-obliterator/internal/opencv/safe"
)

type Algorithm interface {
	Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
	ValidateParameters(params map[string]interface{}) error
	GetDefaultParameters() map[string]interface{}
	GetName() string
}

type ProcessingContext struct {
	Input      *safe.Mat
	Parameters map[string]interface{}
	ProgressCallback func(float64)
}

type ProcessingResult struct {
	Output *safe.Mat
	Metrics map[string]float64
	Error   error
}