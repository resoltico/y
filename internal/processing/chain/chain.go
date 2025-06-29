package chain

import (
	"context"
	"fmt"

	"otsu-obliterator/internal/opencv/safe"
)

type ProcessingStep interface {
	Apply(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
	Name() string
	ShouldExecute(params map[string]interface{}) bool
}

type ProcessingChain struct {
	steps []ProcessingStep
}

func NewProcessingChain(steps []ProcessingStep) *ProcessingChain {
	return &ProcessingChain{
		steps: steps,
	}
}

func (pc *ProcessingChain) Execute(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	current := input
	needsCleanup := false

	for _, step := range pc.steps {
		select {
		case <-ctx.Done():
			if needsCleanup && current != input {
				current.Close()
			}
			return nil, ctx.Err()
		default:
		}

		if !step.ShouldExecute(params) {
			continue
		}

		result, err := step.Apply(ctx, current, params)
		if err != nil {
			if needsCleanup && current != input {
				current.Close()
			}
			return nil, fmt.Errorf("step %s failed: %w", step.Name(), err)
		}

		if needsCleanup && current != input {
			current.Close()
		}

		current = result
		needsCleanup = true
	}

	return current, nil
}

func (pc *ProcessingChain) AddStep(step ProcessingStep) {
	pc.steps = append(pc.steps, step)
}

func (pc *ProcessingChain) InsertStep(index int, step ProcessingStep) error {
	if index < 0 || index > len(pc.steps) {
		return fmt.Errorf("index out of range: %d", index)
	}

	pc.steps = append(pc.steps[:index], append([]ProcessingStep{step}, pc.steps[index:]...)...)
	return nil
}

func (pc *ProcessingChain) RemoveStep(index int) error {
	if index < 0 || index >= len(pc.steps) {
		return fmt.Errorf("index out of range: %d", index)
	}

	pc.steps = append(pc.steps[:index], pc.steps[index+1:]...)
	return nil
}

func (pc *ProcessingChain) StepCount() int {
	return len(pc.steps)
}

func (pc *ProcessingChain) GetStepNames() []string {
	names := make([]string, len(pc.steps))
	for i, step := range pc.steps {
		names[i] = step.Name()
	}
	return names
}