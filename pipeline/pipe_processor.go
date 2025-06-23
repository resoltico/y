package pipeline

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"

	"otsu-obliterator/otsu"
)

// Process2DOtsu processes image using the modularized 2D Otsu algorithm
func (pipeline *ImagePipeline) Process2DOtsu(params map[string]interface{}) (*ImageData, error) {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()

	if pipeline.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	startTime := pipeline.debugManager.StartTiming("2d_otsu_pipeline_process")
	defer pipeline.debugManager.EndTiming("2d_otsu_pipeline_process", startTime)

	processStartTime := time.Now()

	// Use fyne.Do for thread safety in v2.6+
	fyne.Do(func() {
		pipeline.updateProgress(0.1)
		pipeline.updateStatus("Initializing 2D Otsu processor...")
	})

	// Create algorithm manager
	algorithmManager := otsu.NewAlgorithmManager()
	defer algorithmManager.Cleanup()

	fyne.Do(func() {
		pipeline.updateProgress(0.3)
		pipeline.updateStatus("Processing image with 2D Otsu...")
	})

	// Validate parameters before processing
	err := algorithmManager.ValidateParameters("2D Otsu", params)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Process image using modularized 2D Otsu
	srcMat := pipeline.originalImage.Mat.Clone()
	result, err := algorithmManager.Process2DOtsu(srcMat, params)
	srcMat.Close() // Clean up source clone

	if err != nil {
		return nil, fmt.Errorf("2D Otsu processing failed: %w", err)
	}

	fyne.Do(func() {
		pipeline.updateProgress(0.8)
		pipeline.updateStatus("Converting result to image format...")
	})

	// Convert result back to image using thread-safe conversion
	resultImage, err := pipeline.matToImageThreadSafe(result.Result)
	if err != nil {
		result.Result.Close()
		return nil, fmt.Errorf("failed to convert result to image: %w", err)
	}

	fyne.Do(func() {
		pipeline.updateProgress(0.9)
		pipeline.updateStatus("Finalizing processed image...")
	})

	// Clean up previous processed image
	if pipeline.processedImage != nil {
		pipeline.processedImage.Mat.Close()
	}

	// Store processed image with comprehensive metadata
	bounds := resultImage.Bounds()
	pipeline.processedImage = &ImageData{
		Image:       resultImage,
		Mat:         result.Result,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    result.Result.Channels(),
		Format:      pipeline.originalImage.Format,
		OriginalURI: pipeline.originalImage.OriginalURI,
	}

	fyne.Do(func() {
		pipeline.updateProgress(1.0)
		pipeline.updateStatus("2D Otsu processing completed successfully")
	})

	// Log comprehensive processing information
	processingTime := time.Since(processStartTime)
	pipeline.debugManager.LogImageProcessing("2D Otsu", params, processingTime)
	pipeline.logProcessingStatistics("2D Otsu", result.Statistics, processingTime)

	return pipeline.processedImage, nil
}

// ProcessIterativeTriclass processes image using the modularized iterative triclass algorithm
func (pipeline *ImagePipeline) ProcessIterativeTriclass(params map[string]interface{}) (*ImageData, error) {
	pipeline.mu.Lock()
	defer pipeline.mu.Unlock()

	if pipeline.originalImage == nil {
		return nil, fmt.Errorf("no image loaded")
	}

	startTime := pipeline.debugManager.StartTiming("iterative_triclass_pipeline_process")
	defer pipeline.debugManager.EndTiming("iterative_triclass_pipeline_process", startTime)

	processStartTime := time.Now()

	// Use fyne.Do for thread safety in v2.6+
	fyne.Do(func() {
		pipeline.updateProgress(0.1)
		pipeline.updateStatus("Initializing Iterative Triclass processor...")
	})

	// Create algorithm manager
	algorithmManager := otsu.NewAlgorithmManager()
	defer algorithmManager.Cleanup()

	fyne.Do(func() {
		pipeline.updateProgress(0.3)
		pipeline.updateStatus("Processing image with Iterative Triclass...")
	})

	// Validate parameters before processing
	err := algorithmManager.ValidateParameters("Iterative Triclass", params)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Process image using modularized iterative triclass
	srcMat := pipeline.originalImage.Mat.Clone()
	result, err := algorithmManager.ProcessIterativeTriclass(srcMat, params)
	srcMat.Close() // Clean up source clone

	if err != nil {
		return nil, fmt.Errorf("Iterative Triclass processing failed: %w", err)
	}

	fyne.Do(func() {
		pipeline.updateProgress(0.8)
		pipeline.updateStatus("Converting result to image format...")
	})

	// Convert result back to image using thread-safe conversion
	resultImage, err := pipeline.matToImageThreadSafe(result.Result)
	if err != nil {
		result.Result.Close()
		return nil, fmt.Errorf("failed to convert result to image: %w", err)
	}

	fyne.Do(func() {
		pipeline.updateProgress(0.9)
		pipeline.updateStatus("Finalizing processed image...")
	})

	// Clean up previous processed image
	if pipeline.processedImage != nil {
		pipeline.processedImage.Mat.Close()
	}

	// Store processed image with comprehensive metadata
	bounds := resultImage.Bounds()
	pipeline.processedImage = &ImageData{
		Image:       resultImage,
		Mat:         result.Result,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    result.Result.Channels(),
		Format:      pipeline.originalImage.Format,
		OriginalURI: pipeline.originalImage.OriginalURI,
	}

	fyne.Do(func() {
		pipeline.updateProgress(1.0)
		pipeline.updateStatus("Iterative Triclass processing completed successfully")
	})

	// Log comprehensive processing information
	processingTime := time.Since(processStartTime)
	pipeline.debugManager.LogImageProcessing("Iterative Triclass", params, processingTime)
	pipeline.logProcessingStatistics("Iterative Triclass", result.Statistics, processingTime)

	return pipeline.processedImage, nil
}

// matToImageThreadSafe provides thread-safe Mat to Image conversion for Fyne v2.6+
func (pipeline *ImagePipeline) matToImageThreadSafe(mat gocv.Mat) (image.Image, error) {
	// Validate Mat before conversion with additional safety checks
	if mat.Empty() {
		return nil, fmt.Errorf("Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Type() < 0 {
		return nil, fmt.Errorf("Mat has invalid type: %d", mat.Type())
	}

	// Log conversion start
	startTime := pipeline.debugManager.StartTiming("thread_safe_mat_to_image")
	defer pipeline.debugManager.EndTiming("thread_safe_mat_to_image", startTime)

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Converting Mat to Image: %dx%d, %d channels, type %d",
		cols, rows, channels, mat.Type()))

	var resultImage image.Image
	var err error

	// Use recovery for any memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat to Image conversion: %v", r))
			err = fmt.Errorf("conversion failed due to memory access error: %v", r)
		}
	}()

	switch channels {
	case 1:
		// Grayscale conversion with enhanced safety
		gray := image.NewGray(image.Rect(0, 0, cols, rows))
		err = pipeline.copyMatToGrayThreadSafe(mat, gray)
		resultImage = gray
	case 3:
		// BGR to RGB conversion with enhanced safety
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		err = pipeline.copyMatBGRToRGBAThreadSafe(mat, rgba)
		resultImage = rgba
	case 4:
		// BGRA to RGBA conversion with enhanced safety
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		err = pipeline.copyMatBGRAToRGBAThreadSafe(mat, rgba)
		resultImage = rgba
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}

	if err != nil {
		return nil, err
	}

	// Debug pixel analysis for verification
	pipeline.debugManager.LogPixelAnalysis("ThreadSafeConversionOutput", resultImage)
	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Thread-safe conversion completed: %d-channel Mat to Image", channels))

	return resultImage, nil
}

// copyMatToGrayThreadSafe provides enhanced thread-safe grayscale conversion
func (pipeline *ImagePipeline) copyMatToGrayThreadSafe(mat gocv.Mat, img *image.Gray) error {
	// Enhanced validation
	if mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Channels() != 1 {
		return fmt.Errorf("expected 1-channel Mat, got %d channels", mat.Channels())
	}

	bounds := img.Bounds()
	if bounds.Dx() != cols || bounds.Dy() != rows {
		return fmt.Errorf("image size mismatch: Mat=%dx%d, Image=%dx%d", cols, rows, bounds.Dx(), bounds.Dy())
	}

	// Use recovery for any remaining memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat to Gray conversion: %v", r))
		}
	}()

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Thread-safe copying Mat to Gray: %dx%d", cols, rows))

	// Enhanced pixel copying with bounds checking
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Additional bounds checking for safety
			if x >= 0 && x < cols && y >= 0 && y < rows {
				value := mat.GetUCharAt(y, x)
				img.SetGray(x, y, color.Gray{Y: value})
			}
		}
	}

	return nil
}

// copyMatBGRToRGBAThreadSafe provides enhanced thread-safe BGR to RGBA conversion
func (pipeline *ImagePipeline) copyMatBGRToRGBAThreadSafe(mat gocv.Mat, img *image.RGBA) error {
	// Enhanced validation
	if mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Channels() != 3 {
		return fmt.Errorf("expected 3-channel Mat, got %d channels", mat.Channels())
	}

	bounds := img.Bounds()
	if bounds.Dx() != cols || bounds.Dy() != rows {
		return fmt.Errorf("image size mismatch: Mat=%dx%d, Image=%dx%d", cols, rows, bounds.Dx(), bounds.Dy())
	}

	// Use recovery for any remaining memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat BGR to RGBA conversion: %v", r))
		}
	}()

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Thread-safe copying Mat BGR to RGBA: %dx%d", cols, rows))

	// Enhanced pixel copying with bounds checking
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Additional bounds checking for safety
			if x >= 0 && x < cols && y >= 0 && y < rows {
				b := mat.GetUCharAt3(y, x, 0)
				g := mat.GetUCharAt3(y, x, 1)
				r := mat.GetUCharAt3(y, x, 2)
				img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	return nil
}

// copyMatBGRAToRGBAThreadSafe provides enhanced thread-safe BGRA to RGBA conversion
func (pipeline *ImagePipeline) copyMatBGRAToRGBAThreadSafe(mat gocv.Mat, img *image.RGBA) error {
	// Enhanced validation
	if mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Channels() != 4 {
		return fmt.Errorf("expected 4-channel Mat, got %d channels", mat.Channels())
	}

	bounds := img.Bounds()
	if bounds.Dx() != cols || bounds.Dy() != rows {
		return fmt.Errorf("image size mismatch: Mat=%dx%d, Image=%dx%d", cols, rows, bounds.Dx(), bounds.Dy())
	}

	// Use recovery for any remaining memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat BGRA to RGBA conversion: %v", r))
		}
	}()

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Thread-safe copying Mat BGRA to RGBA: %dx%d", cols, rows))

	// Enhanced pixel copying with bounds checking
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Additional bounds checking for safety
			if x >= 0 && x < cols && y >= 0 && y < rows {
				b := mat.GetUCharAt3(y, x, 0)
				g := mat.GetUCharAt3(y, x, 1)
				r := mat.GetUCharAt3(y, x, 2)
				a := mat.GetUCharAt3(y, x, 3)
				img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
			}
		}
	}

	return nil
}

// logProcessingStatistics logs comprehensive processing statistics
func (pipeline *ImagePipeline) logProcessingStatistics(algorithm string, statistics map[string]interface{}, processingTime time.Duration) {
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
	}

	pipeline.debugManager.LogInfo("ProcessingStatistics", report)
}
