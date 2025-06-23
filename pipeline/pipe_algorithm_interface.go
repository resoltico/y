package pipeline

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"

	"otsu-obliterator/otsu"
)

// AlgorithmProcessor handles algorithm-specific processing operations
type AlgorithmProcessor struct {
	pipeline *ImagePipeline
}

// NewAlgorithmProcessor creates a new algorithm processor
func NewAlgorithmProcessor(pipeline *ImagePipeline) *AlgorithmProcessor {
	return &AlgorithmProcessor{
		pipeline: pipeline,
	}
}

// Process2DOtsu processes image using the modularized 2D Otsu algorithm
func (processor *AlgorithmProcessor) Process2DOtsu(params map[string]interface{}) (*ImageData, error) {
	processor.pipeline.mu.Lock()
	defer processor.pipeline.mu.Unlock()

	if processor.pipeline.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	startTime := processor.pipeline.debugManager.StartTiming("2d_otsu_pipeline_process")
	defer processor.pipeline.debugManager.EndTiming("2d_otsu_pipeline_process", startTime)

	processStartTime := time.Now()

	// Use fyne.Do for thread safety in v2.6+
	fyne.Do(func() {
		processor.pipeline.updateProgress(0.1)
		processor.pipeline.updateStatus("Initializing 2D Otsu processor...")
	})

	// Create algorithm manager with latest API
	algorithmManager := otsu.NewAlgorithmManager()
	defer algorithmManager.Cleanup()

	fyne.Do(func() {
		processor.pipeline.updateProgress(0.3)
		processor.pipeline.updateStatus("Processing image with 2D Otsu...")
	})

	// Validate parameters before processing
	err := algorithmManager.ValidateParameters("2D Otsu", params)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Process image using modularized 2D Otsu with enhanced safety
	srcMat := processor.pipeline.originalImage.Mat.Clone()
	result, err := algorithmManager.Process2DOtsu(srcMat, params)
	srcMat.Close() // Clean up source clone immediately

	if err != nil {
		return nil, fmt.Errorf("2D Otsu processing failed: %w", err)
	}

	fyne.Do(func() {
		processor.pipeline.updateProgress(0.8)
		processor.pipeline.updateStatus("Converting result to image format...")
	})

	// Convert result back to image using thread-safe conversion
	resultImage, err := processor.pipeline.matToImageThreadSafe(result.Result)
	if err != nil {
		result.Result.Close()
		return nil, fmt.Errorf("failed to convert result to image: %w", err)
	}

	fyne.Do(func() {
		processor.pipeline.updateProgress(0.9)
		processor.pipeline.updateStatus("Finalizing processed image...")
	})

	// Clean up previous processed image
	if processor.pipeline.processedImage != nil {
		processor.pipeline.processedImage.Mat.Close()
	}

	// Store processed image with comprehensive metadata
	bounds := resultImage.Bounds()
	processor.pipeline.processedImage = &ImageData{
		Image:       resultImage,
		Mat:         result.Result,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    result.Result.Channels(),
		Format:      processor.pipeline.originalImage.Format,
		OriginalURI: processor.pipeline.originalImage.OriginalURI,
	}

	fyne.Do(func() {
		processor.pipeline.updateProgress(1.0)
		processor.pipeline.updateStatus("2D Otsu processing completed successfully")
	})

	// Log comprehensive processing information
	processingTime := time.Since(processStartTime)
	processor.pipeline.debugManager.LogImageProcessing("2D Otsu", params, processingTime)
	processor.logProcessingStatistics("2D Otsu", result.Statistics, processingTime)

	return processor.pipeline.processedImage, nil
}

// ProcessIterativeTriclass processes image using the modularized iterative triclass algorithm
func (processor *AlgorithmProcessor) ProcessIterativeTriclass(params map[string]interface{}) (*ImageData, error) {
	processor.pipeline.mu.Lock()
	defer processor.pipeline.mu.Unlock()

	if processor.pipeline.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	startTime := processor.pipeline.debugManager.StartTiming("iterative_triclass_pipeline_process")
	defer processor.pipeline.debugManager.EndTiming("iterative_triclass_pipeline_process", startTime)

	processStartTime := time.Now()

	// Use fyne.Do for thread safety in v2.6+
	fyne.Do(func() {
		processor.pipeline.updateProgress(0.1)
		processor.pipeline.updateStatus("Initializing Iterative Triclass processor...")
	})

	// Create algorithm manager with latest API
	algorithmManager := otsu.NewAlgorithmManager()
	defer algorithmManager.Cleanup()

	fyne.Do(func() {
		processor.pipeline.updateProgress(0.3)
		processor.pipeline.updateStatus("Processing image with Iterative Triclass...")
	})

	// Validate parameters before processing
	err := algorithmManager.ValidateParameters("Iterative Triclass", params)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Process image using modularized iterative triclass with enhanced safety
	srcMat := processor.pipeline.originalImage.Mat.Clone()
	result, err := algorithmManager.ProcessIterativeTriclass(srcMat, params)
	srcMat.Close() // Clean up source clone immediately

	if err != nil {
		return nil, fmt.Errorf("Iterative Triclass processing failed: %w", err)
	}

	fyne.Do(func() {
		processor.pipeline.updateProgress(0.8)
		processor.pipeline.updateStatus("Converting result to image format...")
	})

	// Convert result back to image using thread-safe conversion
	resultImage, err := processor.pipeline.matToImageThreadSafe(result.Result)
	if err != nil {
		result.Result.Close()
		return nil, fmt.Errorf("failed to convert result to image: %w", err)
	}

	fyne.Do(func() {
		processor.pipeline.updateProgress(0.9)
		processor.pipeline.updateStatus("Finalizing processed image...")
	})

	// Clean up previous processed image
	if processor.pipeline.processedImage != nil {
		processor.pipeline.processedImage.Mat.Close()
	}

	// Store processed image with comprehensive metadata
	bounds := resultImage.Bounds()
	processor.pipeline.processedImage = &ImageData{
		Image:       resultImage,
		Mat:         result.Result,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    result.Result.Channels(),
		Format:      processor.pipeline.originalImage.Format,
		OriginalURI: processor.pipeline.originalImage.OriginalURI,
	}

	fyne.Do(func() {
		processor.pipeline.updateProgress(1.0)
		processor.pipeline.updateStatus("Iterative Triclass processing completed successfully")
	})

	// Log comprehensive processing information
	processingTime := time.Since(processStartTime)
	processor.pipeline.debugManager.LogImageProcessing("Iterative Triclass", params, processingTime)
	processor.logProcessingStatistics("Iterative Triclass", result.Statistics, processingTime)

	return processor.pipeline.processedImage, nil
}

// logProcessingStatistics logs comprehensive processing statistics
func (processor *AlgorithmProcessor) logProcessingStatistics(algorithm string, statistics map[string]interface{}, processingTime time.Duration) {
	// Create detailed processing report
	report := fmt.Sprintf(`%s Processing Statistics:
- Processing Time: %v
- Algorithm Type: %v
- Output Dimensions: %v
- Total Pixels: %v
- Foreground Pixels: %v
- Background Pixels: %v
- Foreground Ratio: %.4f`,
		algorithm,
		processingTime,
		statistics["algorithm_type"],
		statistics["output_dimensions"],
		statistics["total_pixels"],
		statistics["foreground_pixels"],
		statistics["background_pixels"],
		statistics["foreground_ratio"])

	// Add algorithm-specific statistics
	if algorithm == "2D Otsu" {
		if histBins, ok := statistics["histogram_bins"]; ok {
			report += fmt.Sprintf("\n- Histogram Bins: %v", histBins)
		}
		if smoothed, ok := statistics["histogram_smoothed"]; ok {
			report += fmt.Sprintf("\n- Histogram Smoothed: %v", smoothed)
		}
		if normalized, ok := statistics["histogram_normalized"]; ok {
			report += fmt.Sprintf("\n- Histogram Normalized: %v", normalized)
		}
		if processingMode, ok := statistics["processing_mode"]; ok {
			report += fmt.Sprintf("\n- Processing Mode: %v", processingMode)
		}
	} else if algorithm == "Iterative Triclass" {
		if iterCount, ok := statistics["iteration_count"]; ok {
			report += fmt.Sprintf("\n- Iteration Count: %v", iterCount)
		}
		if converged, ok := statistics["convergence_achieved"]; ok {
			report += fmt.Sprintf("\n- Convergence Achieved: %v", converged)
		}
		if finalConv, ok := statistics["final_convergence"]; ok {
			report += fmt.Sprintf("\n- Final Convergence: %.6f", finalConv)
		}
		if processingMode, ok := statistics["processing_mode"]; ok {
			report += fmt.Sprintf("\n- Processing Mode: %v", processingMode)
		}
		if convergenceLog, ok := statistics["convergence_log_length"]; ok {
			report += fmt.Sprintf("\n- Convergence Log Length: %v", convergenceLog)
		}
	}

	processor.pipeline.debugManager.LogInfo("ProcessingStatistics", report)
}

// ValidateProcessingEnvironment checks if environment is ready for processing
func (processor *AlgorithmProcessor) ValidateProcessingEnvironment() error {
	if processor.pipeline.originalImage == nil {
		return fmt.Errorf("no original image loaded")
	}

	if processor.pipeline.originalImage.Mat.Empty() {
		return fmt.Errorf("original image Mat is empty")
	}

	// Check Mat dimensions and type
	rows := processor.pipeline.originalImage.Mat.Rows()
	cols := processor.pipeline.originalImage.Mat.Cols()
	
	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("original image has invalid dimensions: %dx%d", cols, rows)
	}

	if rows < 10 || cols < 10 {
		return fmt.Errorf("image too small for processing: %dx%d (minimum 10x10)", cols, rows)
	}

	matType := processor.pipeline.originalImage.Mat.Type()
	if matType < 0 {
		return fmt.Errorf("original image Mat has invalid type: %d", matType)
	}

	channels := processor.pipeline.originalImage.Mat.Channels()
	if channels < 1 || channels > 4 {
		return fmt.Errorf("unsupported channel count: %d", channels)
	}

	return nil
}

// GetSupportedAlgorithms returns list of algorithms supported by the processor
func (processor *AlgorithmProcessor) GetSupportedAlgorithms() []string {
	return []string{"2D Otsu", "Iterative Triclass"}
}

// GetAlgorithmInfo returns information about a specific algorithm
func (processor *AlgorithmProcessor) GetAlgorithmInfo(algorithm string) map[string]interface{} {
	info := make(map[string]interface{})

	switch algorithm {
	case "2D Otsu":
		info["name"] = "2D Otsu Thresholding"
		info["description"] = "Advanced thresholding using pixel intensity and neighborhood context"
		info["type"] = "histogram_based"
		info["supports_quality_modes"] = true
		info["supports_preprocessing"] = true
		info["typical_processing_time"] = "fast_to_medium"
		info["memory_usage"] = "medium"
		info["best_for"] = []string{"noisy_images", "textured_backgrounds", "variable_illumination"}

	case "Iterative Triclass":
		info["name"] = "Iterative Triclass Thresholding"
		info["description"] = "Iterative segmentation into foreground, background, and undetermined regions"
		info["type"] = "iterative_refinement"
		info["supports_quality_modes"] = true
		info["supports_preprocessing"] = true
		info["typical_processing_time"] = "medium_to_slow"
		info["memory_usage"] = "low_to_medium"
		info["best_for"] = []string{"complex_scenes", "multi_modal_histograms", "gradual_transitions"}

	default:
		info["error"] = "unknown algorithm"
	}

	return info
}
