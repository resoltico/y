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