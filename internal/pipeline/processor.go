package pipeline

import (
	"context"
	"fmt"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/bridge"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"
)

type imageProcessor struct {
	memoryManager    *memory.Manager
	logger           logger.Logger
	algorithmManager *algorithms.Manager
}

// ContextualAlgorithm defines algorithms that support context cancellation
type ContextualAlgorithm interface {
	ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
}

func (p *imageProcessor) ProcessImage(inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	return p.ProcessImageWithContext(context.Background(), inputData, algorithm, params)
}

func (p *imageProcessor) ProcessImageWithContext(ctx context.Context, inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	p.logger.Debug("ImageProcessor", "processing started", map[string]interface{}{
		"algorithm": algorithm.GetName(),
		"width":     inputData.Width,
		"height":    inputData.Height,
	})

	if err := safe.ValidateMatForOperation(inputData.Mat, "ProcessImage"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	// Check for cancellation before processing
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var resultMat *safe.Mat
	var err error

	// Use context-aware processing if algorithm supports it
	if contextualAlg, ok := algorithm.(ContextualAlgorithm); ok {
		resultMat, err = contextualAlg.ProcessWithContext(ctx, inputData.Mat, params)
	} else {
		// Fallback to regular processing for algorithms that don't support context
		resultMat, err = algorithm.Process(inputData.Mat, params)
	}

	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	if resultMat == nil {
		return nil, fmt.Errorf("algorithm returned nil result")
	}

	// Log result Mat properties for debugging
	p.logger.Debug("ImageProcessor", "result Mat created", map[string]interface{}{
		"rows":     resultMat.Rows(),
		"cols":     resultMat.Cols(),
		"channels": resultMat.Channels(),
		"empty":    resultMat.Empty(),
		"valid":    resultMat.IsValid(),
	})

	// Check for cancellation after processing
	if ctx.Err() != nil {
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, ctx.Err()
	}

	resultImage, err := bridge.MatToImage(resultMat)
	if err != nil {
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion failed: %w", err)
	}

	if resultImage == nil {
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion returned nil")
	}

	bounds := resultImage.Bounds()
	processedData := &ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      inputData.Format,
		OriginalURI: inputData.OriginalURI,
	}

	p.logger.Info("ImageProcessor", "processing completed", map[string]interface{}{
		"algorithm":   algorithm.GetName(),
		"input_size":  fmt.Sprintf("%dx%d", inputData.Width, inputData.Height),
		"output_size": fmt.Sprintf("%dx%d", processedData.Width, processedData.Height),
		"image_type":  fmt.Sprintf("%T", resultImage),
	})

	return processedData, nil
}
