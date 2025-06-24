package stages

import (
	"fmt"
	"image"
	"time"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/opencv/bridge"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"
	"otsu-obliterator/internal/pipeline"
)

type Processor struct {
	memoryManager    *memory.Manager
	debugManager     *debug.Manager
	algorithmManager *algorithms.Manager
}

func NewProcessor(memMgr *memory.Manager, debugMgr *debug.Manager, algMgr *algorithms.Manager) *Processor {
	return &Processor{
		memoryManager:    memMgr,
		debugManager:     debugMgr,
		algorithmManager: algMgr,
	}
}

func (p *Processor) ProcessImage(inputData *pipeline.ImageData, algorithmName string, params map[string]interface{}) (*pipeline.ImageData, error) {
	startTime := time.Now()

	algorithm, err := p.algorithmManager.GetAlgorithm(algorithmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get algorithm: %w", err)
	}

	inputMat, ok := inputData.Mat.(*safe.Mat)
	if !ok {
		return nil, fmt.Errorf("input Mat is not a safe Mat")
	}

	if err := safe.ValidateMatForOperation(inputMat, "ProcessImage"); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	resultMat, err := algorithm.Process(inputMat, params)
	if err != nil {
		return nil, fmt.Errorf("algorithm processing failed: %w", err)
	}

	resultImage, err := bridge.MatToImage(resultMat)
	if err != nil {
		resultMat.Close()
		return nil, fmt.Errorf("Mat to image conversion failed: %w", err)
	}

	bounds := resultImage.Bounds()
	processedData := &pipeline.ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      inputData.Format,
		OriginalURI: inputData.OriginalURI,
	}

	p.debugManager.LogInfo("PipelineProcessor", 
		fmt.Sprintf("Processed image with %s, time: %v", algorithmName, time.Since(startTime)))

	return processedData, nil
}