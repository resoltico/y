package otsu

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// applyPostprocessing applies cleanup operations to the final result
func (core *IterativeTriclassCore) applyPostprocessing() (gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("triclass_postprocessing")
	defer core.debugManager.EndTiming("triclass_postprocessing", stepTime)

	if core.iterationData.FinalResult.Empty() {
		return gocv.NewMat(), fmt.Errorf("final result is empty")
	}

	result := core.iterationData.FinalResult.Clone()

	// Apply cleanup if requested
	if core.getBoolParam("apply_cleanup") {
		cleaned, err := core.applyMorphologicalCleanup(&result)
		if err != nil {
			result.Close()
			return gocv.NewMat(), err
		}
		result.Close()
		result = cleaned
	}

	// Apply border preservation if requested
	if core.getBoolParam("preserve_borders") {
		preserved, err := core.preserveBorders(&result)
		if err != nil {
			result.Close()
			return gocv.NewMat(), err
		}
		result.Close()
		result = preserved
	}

	return result, nil
}

// applyMorphologicalCleanup performs morphological operations to clean the result
func (core *IterativeTriclassCore) applyMorphologicalCleanup(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("triclass_morphological_cleanup")
	defer core.debugManager.EndTiming("triclass_morphological_cleanup", stepTime)

	// Create morphological kernels using latest GoCV API
	smallKernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer smallKernel.Close()

	mediumKernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 5, Y: 5})
	defer mediumKernel.Close()

	// Step 1: Remove small noise with opening operation
	opened := gocv.NewMat()
	defer opened.Close()
	gocv.MorphologyEx(*src, &opened, gocv.MorphOpen, smallKernel)

	// Step 2: Fill small holes with closing operation
	closed := gocv.NewMat()
	defer closed.Close()
	gocv.MorphologyEx(opened, &closed, gocv.MorphClose, mediumKernel)

	// Step 3: Apply median filter to smooth boundaries
	smoothed := gocv.NewMat()
	gocv.MedianBlur(closed, &smoothed, 3)

	// Step 4: Optional additional cleanup for "Best" quality
	if core.getStringParam("quality") == "Best" {
		// Apply gradient morphology to preserve edges
		gradient := gocv.NewMat()
		defer gradient.Close()
		gocv.MorphologyEx(smoothed, &gradient, gocv.MorphGradient, smallKernel)

		// Combine with original for edge preservation
		enhanced := gocv.NewMat()
		defer enhanced.Close()
		gocv.BitwiseOr(smoothed, gradient, &enhanced)

		smoothed.Close()
		smoothed = enhanced.Clone()
	}

	return smoothed, nil
}

// preserveBorders ensures border pixels are handled appropriately
func (core *IterativeTriclassCore) preserveBorders(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := core.debugManager.StartTiming("triclass_border_preservation")
	defer core.debugManager.EndTiming("triclass_border_preservation", stepTime)

	result := src.Clone()
	rows := result.Rows()
	cols := result.Cols()

	// Create border mask (1-pixel wide border)
	borderMask := gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV8UC1)
	defer borderMask.Close()
	borderMask.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Set border pixels to 255 in mask
	for x := 0; x < cols; x++ {
		borderMask.SetUCharAt(0, x, 255)         // Top border
		borderMask.SetUCharAt(rows-1, x, 255)    // Bottom border
	}
	for y := 0; y < rows; y++ {
		borderMask.SetUCharAt(y, 0, 255)         // Left border
		borderMask.SetUCharAt(y, cols-1, 255)    // Right border
	}

	// Apply erosion to slightly shrink foreground regions near borders
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	eroded := gocv.NewMat()
	defer eroded.Close()
	gocv.Erode(result, &eroded)

	// Restore border pixels from original
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if borderMask.GetUCharAt(y, x) > 0 {
				// Keep original border pixel values
				originalValue := src.GetUCharAt(y, x)
				result.SetUCharAt(y, x, originalValue)
			} else {
				// Use eroded value for interior pixels
				erodedValue := eroded.GetUCharAt(y, x)
				result.SetUCharAt(y, x, erodedValue)
			}
		}
	}

	return result, nil
}

// logFinalResults logs comprehensive results and statistics
func (core *IterativeTriclassCore) logFinalResults() {
	if core.iterationData == nil {
		return
	}

	stepTime := core.debugManager.StartTiming("triclass_final_logging")
	defer core.debugManager.EndTiming("triclass_final_logging", stepTime)

	// Calculate final statistics
	totalPixels := core.iterationData.TotalPixels
	foregroundPixels := gocv.CountNonZero(core.iterationData.FinalResult)
	backgroundPixels := totalPixels - foregroundPixels

	// Create debug info structure
	debugInfo := &debug.TriclassDebugInfo{
		InputMatDimensions:   fmt.Sprintf("%dx%d", core.iterationData.FinalResult.Cols(), core.iterationData.FinalResult.Rows()),
		InputMatChannels:     1, // Always grayscale for processing
		InputMatType:         gocv.MatTypeCV8UC1,
		OutputMatDimensions:  fmt.Sprintf("%dx%d", core.iterationData.FinalResult.Cols(), core.iterationData.FinalResult.Rows()),
		OutputMatChannels:    1,
		OutputMatType:        gocv.MatTypeCV8UC1,
		IterationCount:       core.iterationData.IterationCount,
		TotalPixels:          totalPixels,
		ForegroundPixels:     foregroundPixels,
		BackgroundPixels:     backgroundPixels,
		TBDPixels:            0, // No TBD pixels in final result
		ProcessingSteps:      core.getProcessingSteps(),
		IterationThresholds:  core.getIterationThresholds(),
		IterationConvergence: core.getIterationConvergences(),
	}

	// Set final threshold from last iteration
	if len(core.convergenceLog) > 0 {
		debugInfo.FinalThreshold = core.convergenceLog[len(core.convergenceLog)-1].Threshold
	}

	core.debugManager.LogTriclassResult(debugInfo)

	// Log convergence analysis
	core.logConvergenceAnalysis()
}

// getProcessingSteps returns a list of processing steps performed
func (core *IterativeTriclassCore) getProcessingSteps() []string {
	steps := []string{"grayscale_conversion"}

	if core.getBoolParam("apply_preprocessing") {
		steps = append(steps, "contrast_enhancement")
		if core.getStringParam("quality") == "Best" {
			steps = append(steps, "denoising")
		}
	}

	steps = append(steps, "iterative_triclass_segmentation")

	if core.getBoolParam("apply_cleanup") {
		steps = append(steps, "morphological_cleanup")
	}

	if core.getBoolParam("preserve_borders") {
		steps = append(steps, "border_preservation")
	}

	return steps
}

// getIterationThresholds extracts thresholds from convergence log
func (core *IterativeTriclassCore) getIterationThresholds() []float64 {
	thresholds := make([]float64, len(core.convergenceLog))
	for i, conv := range core.convergenceLog {
		thresholds[i] = conv.Threshold
	}
	return thresholds
}

// getIterationConvergences extracts convergence values from convergence log
func (core *IterativeTriclassCore) getIterationConvergences() []float64 {
	convergences := make([]float64, len(core.convergenceLog))
	for i, conv := range core.convergenceLog {
		convergences[i] = conv.ConvergenceValue
	}
	return convergences
}

// logConvergenceAnalysis provides detailed convergence analysis
func (core *IterativeTriclassCore) logConvergenceAnalysis() {
	if len(core.convergenceLog) == 0 {
		return
	}

	// Calculate convergence metrics
	totalIterations := len(core.convergenceLog)
	finalConvergence := core.convergenceLog[totalIterations-1].ConvergenceValue
	averageConvergence := 0.0
	
	for _, conv := range core.convergenceLog {
		averageConvergence += conv.ConvergenceValue
	}
	averageConvergence /= float64(totalIterations)

	// Calculate processing efficiency
	finalTBDFraction := core.convergenceLog[totalIterations-1].TBDFraction
	convergenceEpsilon := core.getFloatParam("convergence_epsilon")
	converged := finalConvergence < convergenceEpsilon

	analysisReport := fmt.Sprintf(`Iterative Triclass Convergence Analysis:
- Total Iterations: %d
- Convergence Achieved: %t
- Final Convergence Value: %.6f (threshold: %.6f)
- Average Convergence Rate: %.6f
- Final TBD Fraction: %.6f
- Algorithm Efficiency: %.2f%%`,
		totalIterations,
		converged,
		finalConvergence,
		convergenceEpsilon,
		averageConvergence,
		finalTBDFraction,
		(1.0-finalTBDFraction)*100.0)

	core.debugManager.LogInfo("TriclassAnalysis", analysisReport)

	// Log per-iteration details if debugging is enabled
	if debug.EnableTriclassDebug {
		for i, conv := range core.convergenceLog {
			iterDetail := fmt.Sprintf("Iteration %d: threshold=%.2f, convergence=%.6f, TBD_fraction=%.6f, FG=%d, BG=%d, TBD=%d",
				i, conv.Threshold, conv.ConvergenceValue, conv.TBDFraction,
				conv.ForegroundCount, conv.BackgroundCount, conv.TBDCount)
			core.debugManager.LogInfo("TriclassIteration", iterDetail)
		}
	}
}