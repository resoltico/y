package otsu

import (
	"fmt"
	"image"
	"math"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// IterativeTriclassCore handles the core iterative triclass thresholding algorithm
type IterativeTriclassCore struct {
	params         map[string]interface{}
	debugManager   *debug.Manager
	iterationData  *IterationData
	convergenceLog []ConvergenceInfo
}

// IterationData tracks the state during iterative processing
type IterationData struct {
	CurrentRegion   gocv.Mat
	FinalResult     gocv.Mat
	IterationCount  int
	TotalPixels     int
	ProcessedPixels int
	ActivePixels    int
}

// ConvergenceInfo stores information about each iteration
type ConvergenceInfo struct {
	Iteration        int
	Threshold        float64
	ConvergenceValue float64
	ForegroundCount  int
	BackgroundCount  int
	TBDCount         int
	TBDFraction      float64
	ProcessingTime   float64 // in milliseconds
}

// NewIterativeTriclassCore creates a new iterative triclass processor
func NewIterativeTriclassCore(params map[string]interface{}) *IterativeTriclassCore {
	return &IterativeTriclassCore{
		params:         params,
		debugManager:   debug.NewManager(),
		convergenceLog: make([]ConvergenceInfo, 0),
	}
}

// Process applies iterative triclass thresholding to the input Mat
func (core *IterativeTriclassCore) Process(src gocv.Mat) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("input Mat is empty")
	}

	if src.Rows() <= 0 || src.Cols() <= 0 {
		return gocv.NewMat(), fmt.Errorf("input Mat has invalid dimensions: %dx%d", src.Cols(), src.Rows())
	}

	core.debugManager.LogAlgorithmStart("Iterative Triclass", core.params)
	startTime := time.Now()

	// Create safe working copy
	working := src.Clone()
	defer working.Close()

	// Convert to grayscale if needed
	gray, err := core.prepareGrayscaleImage(&working)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer gray.Close()

	// Apply preprocessing if requested
	preprocessed, err := core.applyPreprocessing(&gray)
	if err != nil {
		return gocv.NewMat(), err
	}
	defer preprocessed.Close()

	// Initialize iteration data
	core.initializeIterationData(&preprocessed)
	defer core.cleanupIterationData()

	// Perform iterative triclass segmentation
	err = core.performIterativeSegmentation(&preprocessed)
	if err != nil {
		return gocv.NewMat(), err
	}

	// Apply post-processing if requested
	result, err := core.applyPostprocessing()
	if err != nil {
		return gocv.NewMat(), err
	}

	// Log final results
	core.logFinalResults()

	core.debugManager.LogAlgorithmComplete("Iterative Triclass", time.Since(startTime),
		fmt.Sprintf("%dx%d", result.Cols(), result.Rows()))

	return result, nil
}

// prepareGrayscaleImage converts input to grayscale using latest GoCV APIs
func (core *IterativeTriclassCore) prepareGrayscaleImage(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer core.debugManager.LogAlgorithmStep("Iterative Triclass", "grayscale_conversion", time.Since(stepTime))

	gray := gocv.NewMat()

	channels := src.Channels()
	switch channels {
	case 1:
		// Already grayscale
		src.CopyTo(&gray)
	case 3:
		// BGR to GRAY using latest API
		gocv.CvtColor(*src, &gray, gocv.ColorBGRToGray)
	case 4:
		// BGRA to GRAY using latest API
		gocv.CvtColor(*src, &gray, gocv.ColorBGRAToGray)
	default:
		return gocv.NewMat(), fmt.Errorf("unsupported number of channels: %d", channels)
	}

	if gray.Empty() {
		return gocv.NewMat(), fmt.Errorf("grayscale conversion failed")
	}

	return gray, nil
}

// applyPreprocessing applies CLAHE and denoising if enabled
func (core *IterativeTriclassCore) applyPreprocessing(src *gocv.Mat) (gocv.Mat, error) {
	stepTime := time.Now()
	defer core.debugManager.LogAlgorithmStep("Iterative Triclass", "preprocessing", time.Since(stepTime))

	result := gocv.NewMat()

	if core.getBoolParam("apply_preprocessing") {
		// Apply CLAHE for contrast enhancement
		clahe := gocv.NewCLAHEWithParams(2.0, image.Point{X: 8, Y: 8})
		defer clahe.Close()

		enhanced := gocv.NewMat()
		defer enhanced.Close()
		clahe.Apply(*src, &enhanced)

		// Apply denoising for better threshold calculation
		if core.getStringParam("quality") == "Best" {
			// Use latest denoising API with optimized parameters
			gocv.FastNlMeansDenoisingWithParams(enhanced, &result, 3.0, 7, 21)
		} else {
			enhanced.CopyTo(&result)
		}
	} else {
		src.CopyTo(&result)
	}

	return result, nil
}

// initializeIterationData prepares data structures for iterative processing
func (core *IterativeTriclassCore) initializeIterationData(src *gocv.Mat) {
	core.iterationData = &IterationData{
		TotalPixels:     src.Rows() * src.Cols(),
		ProcessedPixels: 0,
		ActivePixels:    src.Rows() * src.Cols(),
		IterationCount:  0,
	}

	// Initialize current region with the entire image
	core.iterationData.CurrentRegion = src.Clone()

	// Initialize final result as all background (0)
	core.iterationData.FinalResult = gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	core.iterationData.FinalResult.SetTo(gocv.NewScalar(0, 0, 0, 0))

	// Clear convergence log
	core.convergenceLog = make([]ConvergenceInfo, 0)
}

// cleanupIterationData releases memory allocated for iteration processing
func (core *IterativeTriclassCore) cleanupIterationData() {
	if core.iterationData != nil {
		if !core.iterationData.CurrentRegion.Empty() {
			core.iterationData.CurrentRegion.Close()
		}
		// Note: FinalResult is returned, so don't close it here
		core.iterationData = nil
	}
}

// performIterativeSegmentation executes the main iterative triclass algorithm
func (core *IterativeTriclassCore) performIterativeSegmentation(src *gocv.Mat) error {
	stepTime := time.Now()
	defer core.debugManager.LogAlgorithmStep("Iterative Triclass", "iterative_segmentation", time.Since(stepTime))

	maxIterations := core.getIntParam("max_iterations")
	convergenceEpsilon := core.getFloatParam("convergence_epsilon")
	minTBDFraction := core.getFloatParam("minimum_tbd_fraction")

	var previousThreshold float64 = -1.0

	for iteration := 0; iteration < maxIterations; iteration++ {
		iterStartTime := time.Now()

		// Check if current region has sufficient pixels to process
		activePixels := gocv.CountNonZero(core.iterationData.CurrentRegion)
		if activePixels == 0 {
			break
		}

		// Calculate threshold for current region
		threshold, err := core.calculateRegionThreshold()
		if err != nil {
			return err
		}

		// Check convergence
		convergence := math.Abs(threshold - previousThreshold)
		if previousThreshold >= 0 && convergence < convergenceEpsilon {
			break
		}

		// Perform triclass segmentation
		foreground, background, tbd, err := core.performTriclassSegmentation(threshold)
		if err != nil {
			return err
		}

		// Count pixels in each class
		foregroundCount := gocv.CountNonZero(foreground)
		backgroundCount := gocv.CountNonZero(background)
		tbdCount := gocv.CountNonZero(tbd)

		// Update final result with current classifications
		core.updateFinalResult(&foreground, &background)

		// Calculate TBD fraction
		tbdFraction := float64(tbdCount) / float64(core.iterationData.TotalPixels)

		// Log iteration info
		convInfo := ConvergenceInfo{
			Iteration:        iteration,
			Threshold:        threshold,
			ConvergenceValue: convergence,
			ForegroundCount:  foregroundCount,
			BackgroundCount:  backgroundCount,
			TBDCount:         tbdCount,
			TBDFraction:      tbdFraction,
			ProcessingTime:   float64(time.Since(iterStartTime).Nanoseconds()) / 1e6,
		}
		core.convergenceLog = append(core.convergenceLog, convInfo)

		core.debugManager.LogTriclassIteration(iteration, threshold, convergence,
			foregroundCount, backgroundCount, tbdCount)

		// Check if TBD region is too small to continue
		if tbdFraction < minTBDFraction {
			// Assign remaining TBD pixels using simple threshold
			core.assignRemainingTBDPixels(&tbd, threshold)
			foreground.Close()
			background.Close()
			tbd.Close()
			break
		}

		// Update current region to only include TBD pixels
		err = core.updateCurrentRegion(&tbd)
		foreground.Close()
		background.Close()
		tbd.Close()

		if err != nil {
			return err
		}

		previousThreshold = threshold
		core.iterationData.IterationCount++
	}

	return nil
}

// Utility functions for parameter access with safe defaults
func (core *IterativeTriclassCore) getIntParam(name string) int {
	if value, ok := core.params[name].(int); ok {
		return value
	}
	defaults := map[string]int{
		"max_iterations": 10,
		"histogram_bins": 64,
	}
	if def, exists := defaults[name]; exists {
		return def
	}
	return 0
}

func (core *IterativeTriclassCore) getFloatParam(name string) float64 {
	if value, ok := core.params[name].(float64); ok {
		return value
	}
	defaults := map[string]float64{
		"convergence_epsilon":    1.0,
		"minimum_tbd_fraction":   0.01,
		"lower_upper_gap_factor": 0.5,
	}
	if def, exists := defaults[name]; exists {
		return def
	}
	return 0.0
}

func (core *IterativeTriclassCore) getBoolParam(name string) bool {
	if value, ok := core.params[name].(bool); ok {
		return value
	}
	return false
}

func (core *IterativeTriclassCore) getStringParam(name string) string {
	if value, ok := core.params[name].(string); ok {
		return value
	}
	defaults := map[string]string{
		"initial_threshold_method": "otsu",
		"quality":                  "Fast",
	}
	if def, exists := defaults[name]; exists {
		return def
	}
	return ""
}
