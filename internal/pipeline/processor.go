package pipeline

import (
	"fmt"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/opencv/bridge"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"
)

type imageProcessor struct {
	memoryManager    *memory.Manager
	logger           Logger
	timingTracker    TimingTracker
	algorithmManager *algorithms.Manager
}

func (p *imageProcessor) ProcessImage(inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	ctx := p.timingTracker.StartTiming("process_image")
	defer p.timingTracker.EndTiming(ctx)

	p.logger.Debug("ImageProcessor", "processing started", map[string]interface{}{
		"algorithm": algorithm.GetName(),
		"width":     inputData.Width,
		"height":    inputData.Height,
	})

	if err := safe.ValidateMatForOperation(inputData.Mat, "ProcessImage"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	algorithmCtx := p.timingTracker.StartTiming("algorithm_execution")
	resultMat, err := algorithm.Process(inputData.Mat, params)
	p.timingTracker.EndTiming(algorithmCtx)

	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	conversionCtx := p.timingTracker.StartTiming("mat_to_image_conversion")
	resultImage, err := bridge.MatToImage(resultMat)
	p.timingTracker.EndTiming(conversionCtx)

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

func (p *imageProcessor) Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	return nil, fmt.Errorf("not implemented for direct processor usage")
}

func (p *imageProcessor) ValidateParameters(params map[string]interface{}) error {
	return nil
}

func (p *imageProcessor) GetDefaultParameters() map[string]interface{} {
	return make(map[string]interface{})
}

func (p *imageProcessor) GetName() string {
	return "ImageProcessor"
}
