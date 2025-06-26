package algorithms

import (
	"context"

	"otsu-obliterator/internal/opencv/safe"
)

// Algorithm defines the interface for image processing algorithms
type Algorithm interface {
	Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
	ValidateParameters(params map[string]interface{}) error
	GetDefaultParameters() map[string]interface{}
	GetName() string
}

// ContextualAlgorithm extends Algorithm with context support for cancellation
type ContextualAlgorithm interface {
	Algorithm
	ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
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
