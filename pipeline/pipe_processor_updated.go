package pipeline

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Updated ImagePipeline with modular components
// This file replaces the original pipe_processor.go with a cleaner, modular design

// Process2DOtsu processes image using the modularized 2D Otsu algorithm
func (pipeline *ImagePipeline) Process2DOtsu(params map[string]interface{}) (*ImageData, error) {
	// Create algorithm processor
	algorithmProcessor := NewAlgorithmProcessor(pipeline)
	
	// Validate environment before processing
	if err := algorithmProcessor.ValidateProcessingEnvironment(); err != nil {
		return nil, fmt.Errorf("processing environment validation failed: %w", err)
	}
	
	// Process using modular algorithm interface
	return algorithmProcessor.Process2DOtsu(params)
}

// ProcessIterativeTriclass processes image using the modularized iterative triclass algorithm  
func (pipeline *ImagePipeline) ProcessIterativeTriclass(params map[string]interface{}) (*ImageData, error) {
	// Create algorithm processor
	algorithmProcessor := NewAlgorithmProcessor(pipeline)
	
	// Validate environment before processing
	if err := algorithmProcessor.ValidateProcessingEnvironment(); err != nil {
		return nil, fmt.Errorf("processing environment validation failed: %w", err)
	}
	
	// Process using modular algorithm interface
	return algorithmProcessor.ProcessIterativeTriclass(params)
}

// matToImageThreadSafe provides thread-safe Mat to Image conversion using modular converter
func (pipeline *ImagePipeline) matToImageThreadSafe(mat gocv.Mat) (image.Image, error) {
	// Create thread-safe converter
	converter := NewThreadSafeConverter(pipeline)
	
	// Validate thread safety environment
	if err := converter.ValidateThreadSafetyEnvironment(); err != nil {
		pipeline.debugManager.LogWarning("Pipeline", 
			fmt.Sprintf("Thread safety validation warning: %v", err))
		// Continue with conversion but log the warning
	}
	
	// Perform thread-safe conversion
	return converter.MatToImageThreadSafe(mat)
}

// GetSupportedAlgorithms returns algorithms supported by this pipeline
func (pipeline *ImagePipeline) GetSupportedAlgorithms() []string {
	algorithmProcessor := NewAlgorithmProcessor(pipeline)
	return algorithmProcessor.GetSupportedAlgorithms()
}

// GetAlgorithmInfo returns detailed information about a specific algorithm
func (pipeline *ImagePipeline) GetAlgorithmInfo(algorithm string) map[string]interface{} {
	algorithmProcessor := NewAlgorithmProcessor(pipeline)
	return algorithmProcessor.GetAlgorithmInfo(algorithm)
}

// ValidateProcessingCapabilities checks if pipeline can handle processing requests
func (pipeline *ImagePipeline) ValidateProcessingCapabilities() error {
	pipeline.mu.RLock()
	defer pipeline.mu.RUnlock()
	
	if pipeline.originalImage == nil {
		return fmt.Errorf("no image loaded for processing")
	}
	
	// Create algorithm processor for validation
	algorithmProcessor := NewAlgorithmProcessor(pipeline)
	return algorithmProcessor.ValidateProcessingEnvironment()
}

// GetProcessingStatistics returns current processing capabilities and status
func (pipeline *ImagePipeline) GetProcessingStatistics() map[string]interface{} {
	pipeline.mu.RLock()
	defer pipeline.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Basic pipeline status
	stats["has_original_image"] = pipeline.originalImage != nil
	stats["has_processed_image"] = pipeline.processedImage != nil
	
	if pipeline.originalImage != nil {
		stats["original_dimensions"] = fmt.Sprintf("%dx%d", 
			pipeline.originalImage.Width, pipeline.originalImage.Height)
		stats["original_channels"] = pipeline.originalImage.Channels
		stats["original_format"] = pipeline.originalImage.Format
	}
	
	if pipeline.processedImage != nil {
		stats["processed_dimensions"] = fmt.Sprintf("%dx%d", 
			pipeline.processedImage.Width, pipeline.processedImage.Height)
		stats["processed_channels"] = pipeline.processedImage.Channels
	}
	
	// Thread safety information
	converter := NewThreadSafeConverter(pipeline)
	threadSafetyInfo := converter.GetThreadSafetyInfo()
	stats["thread_safety"] = threadSafetyInfo
	
	// Supported algorithms
	algorithmProcessor := NewAlgorithmProcessor(pipeline)
	stats["supported_algorithms"] = algorithmProcessor.GetSupportedAlgorithms()
	
	return stats
}