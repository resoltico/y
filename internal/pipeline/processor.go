package pipeline

import (
	"fmt"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/opencv/bridge"
	"otsu-obliterator/internal/opencv/safe"
)

type imageProcessor struct {
	memoryManager    MemoryManager
	debugManager     DebugManager
	algorithmManager *algorithms.Manager
}

func (p *imageProcessor) ProcessImage(inputData *ImageData, algorithm algorithms.Algorithm, params map[string]interface{}) (*ImageData, error) {
	startTime := p.debugManager.StartTiming("ProcessImage")
	defer p.debugManager.EndTiming("ProcessImage", startTime)

	if err := safe.ValidateMatForOperation(inputData.Mat, "ProcessImage"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	resultMat, err := algorithm.Process(inputData.Mat, params)
	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	resultImage, err := bridge.MatToImage(resultMat)
	if err != nil {
		p.memoryManager.ReleaseMat(resultMat)
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

	p.debugManager.LogInfo("ImageProcessor",
		fmt.Sprintf("Processed image with %s", algorithm.GetName()))

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
