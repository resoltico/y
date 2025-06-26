package pipeline

import (
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

func (p *imageProcessor) ProcessImage(inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	p.logger.Debug("ImageProcessor", "processing started", map[string]interface{}{
		"algorithm": algorithm.GetName(),
		"width":     inputData.Width,
		"height":    inputData.Height,
	})

	if err := safe.ValidateMatForOperation(inputData.Mat, "ProcessImage"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	resultMat, err := algorithm.Process(inputData.Mat, params)
	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	resultImage, err := bridge.MatToImage(resultMat)
	if err != nil {
		p.memoryManager.ReleaseMat(resultMat, "processing_result")
		return nil, fmt.Errorf("Mat to image conversion failed: %w", err)
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
	})

	return processedData, nil
}
