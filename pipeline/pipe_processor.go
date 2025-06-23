package pipeline

import (
	"fmt"
	"time"

	"otsu-obliterator/otsu"
)

func (pipeline *ImagePipeline) Process2DOtsu(params map[string]interface{}) (*ImageData, error) {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()

	if pipeline.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	startTime := pipeline.debugManager.StartTiming("2d_otsu_process")
	defer pipeline.debugManager.EndTiming("2d_otsu_process", startTime)

	processStartTime := time.Now()
	pipeline.updateProgress(0.1)

	// Create processor
	processor := otsu.NewTwoDOtsuProcessor(params)
	pipeline.updateProgress(0.3)

	// Process image
	srcMat := pipeline.originalImage.Mat.Clone()
	resultMat, err := processor.Process(srcMat)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	pipeline.updateProgress(0.8)

	// Convert result back to image
	resultImage, err := pipeline.matToImage(resultMat)
	if err != nil {
		resultMat.Close()
		return nil, fmt.Errorf("failed to convert result to image: %w", err)
	}
	pipeline.updateProgress(0.9)

	// Clean up previous processed image
	if pipeline.processedImage != nil {
		pipeline.processedImage.Mat.Close()
	}

	// Store processed image with original format info
	bounds := resultImage.Bounds()
	pipeline.processedImage = &ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      pipeline.originalImage.Format, // Inherit format
		OriginalURI: pipeline.originalImage.OriginalURI,
	}

	pipeline.updateProgress(1.0)

	// Log debug information
	processingTime := time.Since(processStartTime)
	pipeline.debugManager.LogImageProcessing("2D Otsu", params, processingTime)
	pipeline.debugManager.LogInfo("Pipeline", "2D Otsu processing completed")

	return pipeline.processedImage, nil
}

func (pipeline *ImagePipeline) ProcessIterativeTriclass(params map[string]interface{}) (*ImageData, error) {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()

	if pipeline.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	startTime := pipeline.debugManager.StartTiming("iterative_triclass_process")
	defer pipeline.debugManager.EndTiming("iterative_triclass_process", startTime)

	processStartTime := time.Now()
	pipeline.updateProgress(0.1)

	// Create processor
	processor := otsu.NewIterativeTriclassProcessor(params)
	pipeline.updateProgress(0.3)

	// Process image
	srcMat := pipeline.originalImage.Mat.Clone()
	resultMat, err := processor.Process(srcMat)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	pipeline.updateProgress(0.8)

	// Convert result back to image
	resultImage, err := pipeline.matToImage(resultMat)
	if err != nil {
		resultMat.Close()
		return nil, fmt.Errorf("failed to convert result to image: %w", err)
	}
	pipeline.updateProgress(0.9)

	// Clean up previous processed image
	if pipeline.processedImage != nil {
		pipeline.processedImage.Mat.Close()
	}

	// Store processed image with original format info
	bounds := resultImage.Bounds()
	pipeline.processedImage = &ImageData{
		Image:       resultImage,
		Mat:         resultMat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    resultMat.Channels(),
		Format:      pipeline.originalImage.Format, // Inherit format
		OriginalURI: pipeline.originalImage.OriginalURI,
	}

	pipeline.updateProgress(1.0)

	// Log debug information
	processingTime := time.Since(processStartTime)
	pipeline.debugManager.LogImageProcessing("Iterative Triclass", params, processingTime)
	pipeline.debugManager.LogInfo("Pipeline", "Iterative Triclass processing completed")

	return pipeline.processedImage, nil
}
