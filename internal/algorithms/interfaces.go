package algorithms

import (
	"otsu-obliterator/internal/opencv/safe"
)

// Algorithm defines the interface for image processing algorithms
type Algorithm interface {
	Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
	ValidateParameters(params map[string]interface{}) error
	GetDefaultParameters() map[string]interface{}
	GetName() string
}

// ProcessingContext provides context for algorithm execution
type ProcessingContext struct {
	Input            *safe.Mat
	Parameters       map[string]interface{}
	ProgressCallback func(float64)
}

// ProcessingResult contains the results of algorithm processing
type ProcessingResult struct {
	Output  *safe.Mat
	Metrics map[string]float64
	Error   error
}

// AlgorithmManager defines the interface for managing algorithms
type AlgorithmManager interface {
	SetCurrentAlgorithm(algorithm string) error
	GetCurrentAlgorithm() string
	GetParameters(algorithm string) map[string]interface{}
	GetAllParameters(algorithm string) map[string]interface{}
	SetParameter(algorithm, name string, value interface{}) error
	GetAlgorithm(name string) (Algorithm, error)
	GetAvailableAlgorithms() []string
}
