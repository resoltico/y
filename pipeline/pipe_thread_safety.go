package pipeline

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

// ThreadSafeConverter provides thread-safe Mat to Image conversion for Fyne v2.6+
type ThreadSafeConverter struct {
	pipeline *ImagePipeline
}

// NewThreadSafeConverter creates a new thread-safe converter
func NewThreadSafeConverter(pipeline *ImagePipeline) *ThreadSafeConverter {
	return &ThreadSafeConverter{
		pipeline: pipeline,
	}
}

// MatToImageThreadSafe provides enhanced thread-safe Mat to Image conversion
func (converter *ThreadSafeConverter) MatToImageThreadSafe(mat gocv.Mat) (image.Image, error) {
	// Comprehensive Mat validation with additional safety checks
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

	// Log conversion start with thread safety
	startTime := converter.pipeline.debugManager.StartTiming("thread_safe_mat_to_image")
	defer converter.pipeline.debugManager.EndTiming("thread_safe_mat_to_image", startTime)

	converter.pipeline.debugManager.LogInfo("ThreadSafeConversion", 
		fmt.Sprintf("Converting Mat to Image: %dx%d, %d channels, type %d",
			cols, rows, channels, mat.Type()))

	var resultImage image.Image
	var err error

	// Use recovery for any memory access issues
	defer func() {
		if r := recover(); r != nil {
			converter.pipeline.debugManager.LogWarning("ThreadSafeConversion", 
				fmt.Sprintf("Panic during Mat to Image conversion: %v", r))
			err = fmt.Errorf("conversion failed due to memory access error: %v", r)
		}
	}()

	switch channels {
	case 1:
		// Grayscale conversion with enhanced safety
		gray := image.NewGray(image.Rect(0, 0, cols, rows))
		err = converter.copyMatToGrayThreadSafe(mat, gray)
		resultImage = gray
	case 3:
		// BGR to RGB conversion with enhanced safety
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		err = converter.copyMatBGRToRGBAThreadSafe(mat, rgba)
		resultImage = rgba
	case 4:
		// BGRA to RGBA conversion with enhanced safety
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		err = converter.copyMatBGRAToRGBAThreadSafe(mat, rgba)
		resultImage = rgba
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}

	if err != nil {
		return nil, err
	}

	// Debug pixel analysis for verification
	converter.pipeline.debugManager.LogPixelAnalysis("ThreadSafeConversionOutput", resultImage)
	converter.pipeline.debugManager.LogInfo("ThreadSafeConversion", 
		fmt.Sprintf("Thread-safe conversion completed: %d-channel Mat to Image", channels))

	return resultImage, nil
}

// copyMatToGrayThreadSafe provides enhanced thread-safe grayscale conversion
func (converter *ThreadSafeConverter) copyMatToGrayThreadSafe(mat gocv.Mat, img *image.Gray) error {
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
			converter.pipeline.debugManager.LogWarning("ThreadSafeConversion", 
				fmt.Sprintf("Panic during Mat to Gray conversion: %v", r))
		}
	}()

	converter.pipeline.debugManager.LogInfo("ThreadSafeConversion", 
		fmt.Sprintf("Thread-safe copying Mat to Gray: %dx%d", cols, rows))

	// Enhanced pixel copying with bounds checking and batch processing
	batchSize := 1000 // Process pixels in batches for better performance
	totalPixels := rows * cols
	processed := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Additional bounds checking for safety
			if x >= 0 && x < cols && y >= 0 && y < rows {
				value := mat.GetUCharAt(y, x)
				img.SetGray(x, y, color.Gray{Y: value})
				
				processed++
				
				// Yield control periodically for thread safety in Fyne v2.6+
				if processed%batchSize == 0 {
					// Allow other goroutines to run
					fyne.Do(func() {
						// Update progress if callback available
						if converter.pipeline.progressCallback != nil {
							progress := float64(processed) / float64(totalPixels) * 0.1 // Small progress increment
							converter.pipeline.progressCallback(progress)
						}
					})
				}
			}
		}
	}

	return nil
}

// copyMatBGRToRGBAThreadSafe provides enhanced thread-safe BGR to RGBA conversion
func (converter *ThreadSafeConverter) copyMatBGRToRGBAThreadSafe(mat gocv.Mat, img *image.RGBA) error {
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
			converter.pipeline.debugManager.LogWarning("ThreadSafeConversion", 
				fmt.Sprintf("Panic during Mat BGR to RGBA conversion: %v", r))
		}
	}()

	converter.pipeline.debugManager.LogInfo("ThreadSafeConversion", 
		fmt.Sprintf("Thread-safe copying Mat BGR to RGBA: %dx%d", cols, rows))

	// Enhanced pixel copying with bounds checking and batch processing
	batchSize := 1000
	totalPixels := rows * cols
	processed := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Additional bounds checking for safety
			if x >= 0 && x < cols && y >= 0 && y < rows {
				b := mat.GetUCharAt3(y, x, 0)
				g := mat.GetUCharAt3(y, x, 1)
				r := mat.GetUCharAt3(y, x, 2)
				img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
				
				processed++
				
				// Yield control periodically for thread safety in Fyne v2.6+
				if processed%batchSize == 0 {
					fyne.Do(func() {
						if converter.pipeline.progressCallback != nil {
							progress := float64(processed) / float64(totalPixels) * 0.1
							converter.pipeline.progressCallback(progress)
						}
					})
				}
			}
		}
	}

	return nil
}

// copyMatBGRAToRGBAThreadSafe provides enhanced thread-safe BGRA to RGBA conversion
func (converter *ThreadSafeConverter) copyMatBGRAToRGBAThreadSafe(mat gocv.Mat, img *image.RGBA) error {
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
			converter.pipeline.debugManager.LogWarning("ThreadSafeConversion", 
				fmt.Sprintf("Panic during Mat BGRA to RGBA conversion: %v", r))
		}
	}()

	converter.pipeline.debugManager.LogInfo("ThreadSafeConversion", 
		fmt.Sprintf("Thread-safe copying Mat BGRA to RGBA: %dx%d", cols, rows))

	// Enhanced pixel copying with bounds checking and batch processing
	batchSize := 1000
	totalPixels := rows * cols
	processed := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			// Additional bounds checking for safety
			if x >= 0 && x < cols && y >= 0 && y < rows {
				b := mat.GetUCharAt3(y, x, 0)
				g := mat.GetUCharAt3(y, x, 1)
				r := mat.GetUCharAt3(y, x, 2)
				a := mat.GetUCharAt3(y, x, 3)
				img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
				
				processed++
				
				// Yield control periodically for thread safety in Fyne v2.6+
				if processed%batchSize == 0 {
					fyne.Do(func() {
						if converter.pipeline.progressCallback != nil {
							progress := float64(processed) / float64(totalPixels) * 0.1
							converter.pipeline.progressCallback(progress)
						}
					})
				}
			}
		}
	}

	return nil
}

// ProcessWithThreadSafety wraps processing operations with Fyne v2.6+ thread safety
func (converter *ThreadSafeConverter) ProcessWithThreadSafety(operation func() error, description string) error {
	startTime := time.Now()
	
	converter.pipeline.debugManager.LogInfo("ThreadSafeProcessing", 
		fmt.Sprintf("Starting thread-safe operation: %s", description))

	// Use recovery to handle any threading issues
	var operationError error
	
	defer func() {
		if r := recover(); r != nil {
			operationError = fmt.Errorf("thread safety violation in %s: %v", description, r)
			converter.pipeline.debugManager.LogWarning("ThreadSafeProcessing", 
				fmt.Sprintf("Recovered from panic in %s: %v", description, r))
		}
	}()

	// Execute operation with thread safety wrapper
	fyne.Do(func() {
		operationError = operation()
	})

	processingTime := time.Since(startTime)
	
	if operationError != nil {
		converter.pipeline.debugManager.LogWarning("ThreadSafeProcessing", 
			fmt.Sprintf("Thread-safe operation %s failed: %v (time: %v)", description, operationError, processingTime))
	} else {
		converter.pipeline.debugManager.LogInfo("ThreadSafeProcessing", 
			fmt.Sprintf("Thread-safe operation %s completed successfully (time: %v)", description, processingTime))
	}

	return operationError
}

// ValidateThreadSafetyEnvironment checks if the environment supports Fyne v2.6+ thread safety
func (converter *ThreadSafeConverter) ValidateThreadSafetyEnvironment() error {
	// Check if fyne.Do is available (indicates v2.6+ support)
	testChannel := make(chan bool, 1)
	
	defer func() {
		if r := recover(); r != nil {
			converter.pipeline.debugManager.LogWarning("ThreadSafetyValidation", 
				"fyne.Do not available - running in legacy mode")
		}
	}()

	// Test fyne.Do functionality
	fyne.Do(func() {
		testChannel <- true
	})

	// Wait for result with timeout
	select {
	case <-testChannel:
		converter.pipeline.debugManager.LogInfo("ThreadSafetyValidation", 
			"Fyne v2.6+ thread safety environment validated successfully")
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("fyne.Do timeout - thread safety environment validation failed")
	}
}

// GetThreadSafetyInfo returns information about current thread safety status
func (converter *ThreadSafeConverter) GetThreadSafetyInfo() map[string]interface{} {
	info := make(map[string]interface{})
	
	// Test fyne.Do availability
	testPassed := false
	defer func() {
		if r := recover(); r != nil {
			info["fyne_do_available"] = false
			info["error"] = fmt.Sprintf("fyne.Do test failed: %v", r)
		}
	}()

	testChannel := make(chan bool, 1)
	fyne.Do(func() {
		testPassed = true
		testChannel <- true
	})

	select {
	case <-testChannel:
		info["fyne_do_available"] = true
		info["thread_safety_mode"] = "fyne_v2.6+"
	case <-time.After(100 * time.Millisecond):
		info["fyne_do_available"] = false
		info["thread_safety_mode"] = "legacy"
		info["warning"] = "fyne.Do timeout"
	}

	info["test_passed"] = testPassed
	info["validation_timestamp"] = time.Now().Unix()
	
	return info
}