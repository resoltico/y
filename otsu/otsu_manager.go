package otsu

import (
	"fmt"
	"sync"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// AlgorithmManager coordinates between different Otsu algorithm implementations
type AlgorithmManager struct {
	mu               sync.RWMutex
	currentAlgorithm string
	parameters       map[string]map[string]interface{}
	debugManager     *debug.Manager
}

// ProcessingResult encapsulates algorithm processing results with metadata
type ProcessingResult struct {
	Result      gocv.Mat
	Algorithm   string
	Parameters  map[string]interface{}
	Statistics  map[string]interface{}
	ProcessTime float64 // in milliseconds
}

// NewAlgorithmManager creates a new algorithm manager with debug support
func NewAlgorithmManager() *AlgorithmManager {
	manager := &AlgorithmManager{
		currentAlgorithm: "2D Otsu",
		parameters:       make(map[string]map[string]interface{}),
		debugManager:     debug.NewManager(),
	}

	manager.initializeDefaultParameters()
	return manager
}

// initializeDefaultParameters sets up algorithm-specific parameter defaults
func (am *AlgorithmManager) initializeDefaultParameters() {
	// 2D Otsu default parameters with mathematical justifications
	am.parameters["2D Otsu"] = map[string]interface{}{
		"quality":                    "Fast",          // Processing mode
		"window_size":                7,               // Odd-sized window for neighborhood analysis
		"histogram_bins":             64,              // Balance between granularity and computation
		"neighbourhood_metric":       "mean",          // Statistical measure for neighborhood
		"pixel_weight_factor":        0.5,             // Equal weighting between pixel and neighborhood
		"smoothing_sigma":           1.0,              // Gaussian smoothing parameter
		"use_log_histogram":         false,            // Logarithmic histogram scaling
		"normalize_histogram":       true,             // Convert to probability distribution
		"apply_contrast_enhancement": false,           // CLAHE preprocessing
	}

	// Iterative Triclass default parameters based on research literature
	am.parameters["Iterative Triclass"] = map[string]interface{}{
		"quality":                   "Fast",           // Processing mode
		"initial_threshold_method":  "otsu",           // Bootstrap threshold method
		"histogram_bins":            64,               // Histogram resolution
		"convergence_epsilon":       1.0,              // Convergence criterion (intensity units)
		"max_iterations":            10,               // Maximum iteration limit
		"minimum_tbd_fraction":      0.01,             // Stop when TBD region is small enough
		"lower_upper_gap_factor":    0.5,              // Gap between lower/upper thresholds
		"apply_preprocessing":       false,            // CLAHE and denoising
		"apply_cleanup":             true,             // Morphological post-processing
		"preserve_borders":          false,            // Border pixel preservation
	}
}

// Process2DOtsu executes 2D Otsu thresholding with current parameters
func (am *AlgorithmManager) Process2DOtsu(src gocv.Mat, params map[string]interface{}) (*ProcessingResult, error) {
	am.mu.RLock()
	mergedParams := am.mergeParameters("2D Otsu", params)
	am.mu.RUnlock()

	startTime := am.debugManager.StartTiming("2d_otsu_manager_process")
	defer am.debugManager.EndTiming("2d_otsu_manager_process", startTime)

	// Create processor instance
	processor := NewTwoDOtsuCore(mergedParams)

	// Process image
	result, err := processor.Process(src)
	if err != nil {
		return nil, fmt.Errorf("2D Otsu processing failed: %w", err)
	}

	// Gather statistics
	statistics := am.gather2DOtsuStatistics(processor, &result)

	return &ProcessingResult{
		Result:      result,
		Algorithm:   "2D Otsu",
		Parameters:  mergedParams,
		Statistics:  statistics,
		ProcessTime: float64(am.debugManager.StartTiming("2d_otsu_manager_process").UnixNano()) / 1e6,
	}, nil
}

// ProcessIterativeTriclass executes iterative triclass thresholding with current parameters
func (am *AlgorithmManager) ProcessIterativeTriclass(src gocv.Mat, params map[string]interface{}) (*ProcessingResult, error) {
	am.mu.RLock()
	mergedParams := am.mergeParameters("Iterative Triclass", params)
	am.mu.RUnlock()

	startTime := am.debugManager.StartTiming("iterative_triclass_manager_process")
	defer am.debugManager.EndTiming("iterative_triclass_manager_process", startTime)

	// Create processor instance
	processor := NewIterativeTriclassCore(mergedParams)

	// Process image
	result, err := processor.Process(src)
	if err != nil {
		return nil, fmt.Errorf("Iterative Triclass processing failed: %w", err)
	}

	// Gather statistics
	statistics := am.gatherTriclassStatistics(processor, &result)

	return &ProcessingResult{
		Result:      result,
		Algorithm:   "Iterative Triclass",
		Parameters:  mergedParams,
		Statistics:  statistics,
		ProcessTime: float64(am.debugManager.StartTiming("iterative_triclass_manager_process").UnixNano()) / 1e6,
	}, nil
}

// gather2DOtsuStatistics collects processing statistics from 2D Otsu processor
func (am *AlgorithmManager) gather2DOtsuStatistics(processor *TwoDOtsuCore, result *gocv.Mat) map[string]interface{} {
	statistics := make(map[string]interface{})

	// Basic image statistics
	statistics["output_dimensions"] = fmt.Sprintf("%dx%d", result.Cols(), result.Rows())
	statistics["output_channels"] = result.Channels()
	statistics["total_pixels"] = result.Rows() * result.Cols()

	// Binary image statistics
	foregroundPixels := gocv.CountNonZero(*result)
	backgroundPixels := statistics["total_pixels"].(int) - foregroundPixels

	statistics["foreground_pixels"] = foregroundPixels
	statistics["background_pixels"] = backgroundPixels
	statistics["foreground_ratio"] = float64(foregroundPixels) / float64(statistics["total_pixels"].(int))

	// Histogram information
	if processor.histogramData != nil {
		statistics["histogram_bins"] = processor.histogramData.bins
		statistics["histogram_smoothed"] = processor.histogramData.smoothed
		statistics["histogram_normalized"] = processor.histogramData.normalized
		statistics["histogram_log_scaled"] = processor.histogramData.logScaled
	}

	statistics["algorithm_type"] = "2D_Otsu_Thresholding"
	statistics["processing_mode"] = processor.getStringParam("quality")

	return statistics
}

// gatherTriclassStatistics collects processing statistics from iterative triclass processor
func (am *AlgorithmManager) gatherTriclassStatistics(processor *IterativeTriclassCore, result *gocv.Mat) map[string]interface{} {
	// Get base statistics from processor
	statistics := processor.GetProcessingStatistics()

	// Add additional manager-level statistics
	statistics["output_dimensions"] = fmt.Sprintf("%dx%d", result.Cols(), result.Rows())
	statistics["output_channels"] = result.Channels()
	statistics["algorithm_type"] = "Iterative_Triclass_Thresholding"
	statistics["processing_mode"] = processor.getStringParam("quality")

	// Add convergence information
	convergenceLog := processor.GetConvergenceLog()
	if len(convergenceLog) > 0 {
		statistics["convergence_log_length"] = len(convergenceLog)
		statistics["final_threshold"] = convergenceLog[len(convergenceLog)-1].Threshold
		statistics["final_tbd_fraction"] = convergenceLog[len(convergenceLog)-1].TBDFraction
	}

	return statistics
}

// mergeParameters combines default parameters with user-provided overrides
func (am *AlgorithmManager) mergeParameters(algorithm string, overrides map[string]interface{}) map[string]interface{} {
	// Start with defaults
	merged := make(map[string]interface{})
	if defaults, exists := am.parameters[algorithm]; exists {
		for k, v := range defaults {
			merged[k] = v
		}
	}

	// Apply overrides
	for k, v := range overrides {
		merged[k] = v
	}

	return merged
}

// SetCurrentAlgorithm updates the active algorithm
func (am *AlgorithmManager) SetCurrentAlgorithm(algorithm string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.parameters[algorithm]; exists {
		am.currentAlgorithm = algorithm
		am.debugManager.LogAlgorithmSwitch("", algorithm)
	}
}

// GetCurrentAlgorithm returns the currently selected algorithm
func (am *AlgorithmManager) GetCurrentAlgorithm() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.currentAlgorithm
}

// GetParameters returns a copy of parameters for the specified algorithm
func (am *AlgorithmManager) GetParameters(algorithm string) map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if params, exists := am.parameters[algorithm]; exists {
		// Return deep copy to prevent external modification
		result := make(map[string]interface{})
		for k, v := range params {
			result[k] = v
		}
		return result
	}

	return make(map[string]interface{})
}

// GetAllParameters returns all parameters for the specified algorithm
func (am *AlgorithmManager) GetAllParameters(algorithm string) map[string]interface{} {
	return am.GetParameters(algorithm)
}

// SetParameter updates a specific parameter for an algorithm
func (am *AlgorithmManager) SetParameter(algorithm, name string, value interface{}) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if params, exists := am.parameters[algorithm]; exists {
		oldValue := params[name]
		params[name] = value
		am.debugManager.LogParameterChange(algorithm, name, oldValue, value)
	}
}

// GetParameter retrieves a specific parameter value
func (am *AlgorithmManager) GetParameter(algorithm, name string) (interface{}, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if params, exists := am.parameters[algorithm]; exists {
		if value, exists := params[name]; exists {
			return value, true
		}
	}

	return nil, false
}

// ValidateParameters checks if parameters are within acceptable ranges
func (am *AlgorithmManager) ValidateParameters(algorithm string, params map[string]interface{}) error {
	switch algorithm {
	case "2D Otsu":
		return am.validate2DOtsuParameters(params)
	case "Iterative Triclass":
		return am.validateTriclassParameters(params)
	default:
		return fmt.Errorf("unknown algorithm: %s", algorithm)
	}
}

// validate2DOtsuParameters validates 2D Otsu parameter ranges
func (am *AlgorithmManager) validate2DOtsuParameters(params map[string]interface{}) error {
	// Window size validation
	if windowSize, ok := params["window_size"].(int); ok {
		if windowSize < 3 || windowSize > 31 || windowSize%2 == 0 {
			return fmt.Errorf("window_size must be odd and between 3-31, got %d", windowSize)
		}
	}

	// Histogram bins validation
	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins < 16 || histBins > 256 {
			return fmt.Errorf("histogram_bins must be between 16-256, got %d", histBins)
		}
	}

	// Pixel weight factor validation
	if pixelWeight, ok := params["pixel_weight_factor"].(float64); ok {
		if pixelWeight < 0.0 || pixelWeight > 1.0 {
			return fmt.Errorf("pixel_weight_factor must be between 0.0-1.0, got %f", pixelWeight)
		}
	}

	// Smoothing sigma validation
	if sigma, ok := params["smoothing_sigma"].(float64); ok {
		if sigma < 0.0 || sigma > 5.0 {
			return fmt.Errorf("smoothing_sigma must be between 0.0-5.0, got %f", sigma)
		}
	}

	return nil
}

// validateTriclassParameters validates iterative triclass parameter ranges
func (am *AlgorithmManager) validateTriclassParameters(params map[string]interface{}) error {
	// Max iterations validation
	if maxIter, ok := params["max_iterations"].(int); ok {
		if maxIter < 1 || maxIter > 50 {
			return fmt.Errorf("max_iterations must be between 1-50, got %d", maxIter)
		}
	}

	// Convergence epsilon validation
	if epsilon, ok := params["convergence_epsilon"].(float64); ok {
		if epsilon < 0.1 || epsilon > 20.0 {
			return fmt.Errorf("convergence_epsilon must be between 0.1-20.0, got %f", epsilon)
		}
	}

	// Minimum TBD fraction validation
	if minTBD, ok := params["minimum_tbd_fraction"].(float64); ok {
		if minTBD < 0.001 || minTBD > 0.5 {
			return fmt.Errorf("minimum_tbd_fraction must be between 0.001-0.5, got %f", minTBD)
		}
	}

	// Gap factor validation
	if gapFactor, ok := params["lower_upper_gap_factor"].(float64); ok {
		if gapFactor < 0.0 || gapFactor > 1.0 {
			return fmt.Errorf("lower_upper_gap_factor must be between 0.0-1.0, got %f", gapFactor)
		}
	}

	return nil
}

// GetAvailableAlgorithms returns list of supported algorithms
func (am *AlgorithmManager) GetAvailableAlgorithms() []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	algorithms := make([]string, 0, len(am.parameters))
	for algorithm := range am.parameters {
		algorithms = append(algorithms, algorithm)
	}
	return algorithms
}

// GetPerformanceReport returns performance analysis of recent processing
func (am *AlgorithmManager) GetPerformanceReport() string {
	return am.debugManager.GetPerformanceReport()
}

// Cleanup releases resources used by the algorithm manager
func (am *AlgorithmManager) Cleanup() {
	am.mu.Lock()
	defer am.mu.Unlock()

	if am.debugManager != nil {
		am.debugManager.Cleanup()
	}
}