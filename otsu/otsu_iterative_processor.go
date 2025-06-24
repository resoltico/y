package otsu

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

type IterativeTriclassProcessor struct {
	params        map[string]interface{}
	debugManager  *debug.Manager
	memoryManager MemoryManagerInterface
	segmenter     *TriclassSegmenter
	thresholder   *TriclassThresholder
}

type MemoryManagerInterface interface {
	GetMat(rows, cols int, matType gocv.MatType) gocv.Mat
	ReleaseMat(mat gocv.Mat)
}

func NewIterativeTriclassProcessor(params map[string]interface{}, memoryManager MemoryManagerInterface) *IterativeTriclassProcessor {
	return &IterativeTriclassProcessor{
		params:        params,
		debugManager:  debug.NewManager(),
		memoryManager: memoryManager,
		segmenter:     NewTriclassSegmenter(params, memoryManager),
		thresholder:   NewTriclassThresholder(params),
	}
}

func (processor *IterativeTriclassProcessor) Process(src gocv.Mat) (gocv.Mat, error) {
	// Validate input Mat thoroughly
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	if src.Rows() <= 0 || src.Cols() <= 0 {
		return gocv.NewMat(), fmt.Errorf("input Mat has invalid dimensions: %dx%d", src.Cols(), src.Rows())
	}

	// Create a safe working copy immediately to avoid memory issues
	safeCopy := src.Clone()
	defer processor.memoryManager.ReleaseMat(safeCopy)

	if safeCopy.Empty() {
		return gocv.NewMat(), fmt.Errorf("failed to create safe copy of input Mat")
	}

	// Debug initial input using safe copy
	processor.debugManager.LogTriclassStart(safeCopy, processor.params)
	processor.debugManager.LogMatPixelAnalysis("TriclassInput", safeCopy)

	// Convert to grayscale if needed
	gray := processor.memoryManager.GetMat(safeCopy.Rows(), safeCopy.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(gray)

	if safeCopy.Channels() == 3 {
		gocv.CvtColor(safeCopy, &gray, gocv.ColorBGRToGray)
	} else {
		safeCopy.CopyTo(&gray)
	}

	if gray.Empty() {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed")
	}

	processor.debugManager.LogMatPixelAnalysis("TriclassGrayscale", gray)

	// Apply preprocessing if requested
	working := processor.memoryManager.GetMat(gray.Rows(), gray.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(working)

	if processor.getBoolParam("apply_preprocessing") {
		processor.applyPreprocessing(&gray, &working)
		if working.Empty() {
			return gocv.NewMat(), fmt.Errorf("preprocessing failed")
		}
		processor.debugManager.LogMatPixelAnalysis("TriclassPreprocessed", working)
	} else {
		gray.CopyTo(&working)
	}

	// Iterative triclass processing
	result, err := processor.performIterativeTriclass(&working)
	if err != nil {
		return gocv.NewMat(), err
	}

	// Apply cleanup if requested
	if processor.getBoolParam("apply_cleanup") {
		cleaned := processor.memoryManager.GetMat(result.Rows(), result.Cols(), gocv.MatTypeCV8UC1)
		processor.memoryManager.ReleaseMat(result)
		processor.applyCleanup(&result, &cleaned)
		if cleaned.Empty() {
			return gocv.NewMat(), fmt.Errorf("cleanup failed")
		}
		processor.debugManager.LogMatPixelAnalysis("TriclassCleanedResult", cleaned)
		return cleaned, nil
	}

	processor.debugManager.LogMatPixelAnalysis("TriclassFinalResult", result)
	return result, nil
}

func (processor *IterativeTriclassProcessor) performIterativeTriclass(working *gocv.Mat) (gocv.Mat, error) {
	maxIterations := processor.getIntParam("max_iterations")
	convergenceEpsilon := processor.getFloatParam("convergence_epsilon")
	minTBDFraction := processor.getFloatParam("minimum_tbd_fraction")

	// Initialize final result
	result := processor.memoryManager.GetMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	result.SetTo(gocv.NewScalar(0, 0, 0, 0)) // Start with all background

	// Current working region
	currentRegion := processor.memoryManager.GetMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(currentRegion)

	if working.Empty() || currentRegion.Empty() {
		return gocv.NewMat(), fmt.Errorf("invalid Mat for CopyTo operation")
	}
	working.CopyTo(&currentRegion)

	var previousThreshold float64 = -1
	var iterationThresholds []float64
	var iterationConvergence []float64

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Check if current region has any pixels to process
		nonZeroPixels := gocv.CountNonZero(currentRegion)
		if nonZeroPixels == 0 {
			processor.debugManager.LogTriclassIteration(iteration, previousThreshold, 0, 0, 0, 0)
			break
		}

		// Calculate threshold for current region
		threshold := processor.thresholder.CalculateThresholdForRegion(&currentRegion)
		iterationThresholds = append(iterationThresholds, threshold)

		// Check convergence
		convergence := abs(threshold - previousThreshold)
		iterationConvergence = append(iterationConvergence, convergence)

		if previousThreshold >= 0 && convergence < convergenceEpsilon {
			processor.debugManager.LogTriclassIteration(iteration, threshold, convergence, 0, 0, 0)
			break
		}
		previousThreshold = threshold

		// Segment current region into three classes
		foregroundMask, backgroundMask, tbdMask := processor.segmenter.SegmentRegion(&currentRegion, threshold)

		// Count pixels in each class
		foregroundCount := gocv.CountNonZero(foregroundMask)
		backgroundCount := gocv.CountNonZero(backgroundMask)
		tbdCount := gocv.CountNonZero(tbdMask)

		processor.debugManager.LogTriclassIteration(iteration, threshold, convergence,
			foregroundCount, backgroundCount, tbdCount)

		// Update final result with current classifications
		processor.updateResult(&result, &foregroundMask, &backgroundMask)

		// Check if TBD region is too small
		totalPixels := currentRegion.Rows() * currentRegion.Cols()
		tbdFraction := float64(tbdCount) / float64(totalPixels)

		if tbdFraction < minTBDFraction {
			processor.memoryManager.ReleaseMat(foregroundMask)
			processor.memoryManager.ReleaseMat(backgroundMask)
			processor.memoryManager.ReleaseMat(tbdMask)
			break
		}

		// Update current region to only include TBD pixels
		newRegion := processor.memoryManager.GetMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
		processor.extractTBDRegion(working, &tbdMask, &newRegion)
		processor.memoryManager.ReleaseMat(currentRegion)
		currentRegion = newRegion

		processor.memoryManager.ReleaseMat(foregroundMask)
		processor.memoryManager.ReleaseMat(backgroundMask)
		processor.memoryManager.ReleaseMat(tbdMask)
	}

	// Log debug information
	totalPixels := result.Rows() * result.Cols()
	foregroundPixels := gocv.CountNonZero(result)
	backgroundPixels := totalPixels - foregroundPixels

	debugInfo := &debug.TriclassDebugInfo{
		InputMatDimensions:   fmt.Sprintf("%dx%d", working.Cols(), working.Rows()),
		InputMatChannels:     working.Channels(),
		InputMatType:         working.Type(),
		OutputMatDimensions:  fmt.Sprintf("%dx%d", result.Cols(), result.Rows()),
		OutputMatChannels:    result.Channels(),
		OutputMatType:        result.Type(),
		IterationCount:       len(iterationThresholds),
		FinalThreshold:       previousThreshold,
		TotalPixels:          totalPixels,
		ForegroundPixels:     foregroundPixels,
		BackgroundPixels:     backgroundPixels,
		TBDPixels:            0,
		ProcessingSteps:      []string{"grayscale", "preprocessing", "iterative_segmentation", "final_result"},
		IterationThresholds:  iterationThresholds,
		IterationConvergence: iterationConvergence,
	}

	processor.debugManager.LogTriclassResult(debugInfo)

	return result, nil
}

func (processor *IterativeTriclassProcessor) updateResult(result, foregroundMask, backgroundMask *gocv.Mat) {
	rows := result.Rows()
	cols := result.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if foregroundMask.GetUCharAt(y, x) > 0 {
				result.SetUCharAt(y, x, 255)
			}
		}
	}
}

func (processor *IterativeTriclassProcessor) extractTBDRegion(original, tbdMask, result *gocv.Mat) {
	if original.Empty() || tbdMask.Empty() || result.Empty() {
		return
	}

	result.SetTo(gocv.NewScalar(0, 0, 0, 0))

	rows := original.Rows()
	cols := original.Cols()

	if rows != tbdMask.Rows() || cols != tbdMask.Cols() {
		return
	}

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdMask.GetUCharAt(y, x) > 0 {
				result.SetUCharAt(y, x, original.GetUCharAt(y, x))
			}
		}
	}
}

func (processor *IterativeTriclassProcessor) applyPreprocessing(src, dst *gocv.Mat) {
	// Apply CLAHE for contrast enhancement
	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	clahe.Apply(*src, dst)

	// Apply denoising
	denoised := processor.memoryManager.GetMat(dst.Rows(), dst.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(denoised)

	gocv.FastNlMeansDenoising(*dst, &denoised)
	denoised.CopyTo(dst)
}

func (processor *IterativeTriclassProcessor) applyCleanup(src, dst *gocv.Mat) {
	// Apply morphological operations to clean up the result
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	// Opening to remove small noise
	opened := processor.memoryManager.GetMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(opened)
	gocv.MorphologyEx(*src, &opened, gocv.MorphOpen, kernel)

	// Closing to fill small holes
	gocv.MorphologyEx(opened, dst, gocv.MorphClose, kernel)

	// Apply median filter to smooth boundaries
	medianFiltered := processor.memoryManager.GetMat(dst.Rows(), dst.Cols(), gocv.MatTypeCV8UC1)
	defer processor.memoryManager.ReleaseMat(medianFiltered)
	gocv.MedianBlur(*dst, &medianFiltered, 3)
	medianFiltered.CopyTo(dst)
}

func (processor *IterativeTriclassProcessor) getIntParam(name string) int {
	if value, ok := processor.params[name].(int); ok {
		return value
	}
	return 0
}

func (processor *IterativeTriclassProcessor) getFloatParam(name string) float64 {
	if value, ok := processor.params[name].(float64); ok {
		return value
	}
	return 0.0
}

func (processor *IterativeTriclassProcessor) getBoolParam(name string) bool {
	if value, ok := processor.params[name].(bool); ok {
		return value
	}
	return false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
