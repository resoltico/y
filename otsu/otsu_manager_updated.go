package otsu

import (
	"fmt"
	"image"
	"math"
	"sync"
	"time"

	"gocv.io/x/gocv"

	"otsu-obliterator/debug"
)

// AlgorithmManagerEnhanced coordinates between different Otsu algorithm implementations
// This replaces the original AlgorithmManager with better modularization and mathematical accuracy
type AlgorithmManagerEnhanced struct {
	mu               sync.RWMutex
	currentAlgorithm string
	parameters       map[string]map[string]interface{}
	debugManager     *debug.Manager

	// Processor instances for reuse
	twoDProcessor     *TwoDOtsuProcessor
	triclassProcessor *IterativeTriclassProcessor
}

// ProcessingResultEnhanced encapsulates algorithm processing results with enhanced metadata
type ProcessingResultEnhanced struct {
	Result         gocv.Mat
	Algorithm      string
	Parameters     map[string]interface{}
	Statistics     map[string]interface{}
	ProcessTime    float64 // in milliseconds
	QualityMetrics map[string]interface{}
	DebugInfo      map[string]interface{}
}

// NewAlgorithmManagerEnhanced creates a new enhanced algorithm manager
func NewAlgorithmManagerEnhanced() *AlgorithmManagerEnhanced {
	manager := &AlgorithmManagerEnhanced{
		currentAlgorithm: "2D Otsu",
		parameters:       make(map[string]map[string]interface{}),
		debugManager:     debug.NewManager(),
	}

	manager.initializeDefaultParameters()
	return manager
}

// initializeDefaultParameters sets up algorithm-specific parameter defaults with mathematical justification
func (manager *AlgorithmManagerEnhanced) initializeDefaultParameters() {
	// Enhanced 2D Otsu parameters with mathematical validation
	manager.parameters["2D Otsu"] = map[string]interface{}{
		"quality":                    "Fast", // Processing mode: Fast/Best
		"window_size":                7,      // Odd-sized window (3-31) for neighborhood analysis
		"histogram_bins":             64,     // Power of 2 for efficient computation (8-512)
		"neighbourhood_metric":       "mean", // Statistical measure: mean/median/gaussian
		"pixel_weight_factor":        0.5,    // Balanced weighting between pixel and neighborhood
		"smoothing_sigma":            1.0,    // Gaussian smoothing parameter (0.0-10.0)
		"use_log_histogram":          false,  // Logarithmic histogram scaling for high dynamic range
		"normalize_histogram":        true,   // Convert to probability distribution
		"apply_contrast_enhancement": false,  // CLAHE preprocessing
	}

	// Enhanced Iterative Triclass parameters with convergence analysis
	manager.parameters["Iterative Triclass"] = map[string]interface{}{
		"quality":                  "Fast", // Processing mode: Fast/Best
		"initial_threshold_method": "otsu", // Bootstrap method: otsu/mean/median
		"histogram_bins":           64,     // Histogram resolution (8-256)
		"convergence_epsilon":      1.0,    // Convergence criterion in intensity units
		"max_iterations":           10,     // Maximum iteration safety limit (1-100)
		"minimum_tbd_fraction":     0.01,   // Early termination threshold (0.0001-0.5)
		"lower_upper_gap_factor":   0.5,    // Gap between lower/upper thresholds (0.0-1.0)
		"apply_preprocessing":      false,  // CLAHE and denoising
		"apply_cleanup":            true,   // Morphological post-processing
		"preserve_borders":         false,  // Border pixel preservation
	}
}

// Process2DOtsuEnhanced executes enhanced 2D Otsu thresholding
func (manager *AlgorithmManagerEnhanced) Process2DOtsuEnhanced(src gocv.Mat, params map[string]interface{}) (*ProcessingResultEnhanced, error) {
	manager.mu.RLock()
	mergedParams := manager.mergeParameters("2D Otsu", params)
	manager.mu.RUnlock()

	startTime := time.Now()

	// Create or reuse processor instance
	if manager.twoDProcessor == nil {
		manager.twoDProcessor = NewTwoDOtsuProcessor(mergedParams)
	}

	// Validate parameters before processing
	if err := manager.twoDProcessor.ValidateParameters(); err != nil {
		return nil, fmt.Errorf("2D Otsu parameter validation failed: %w", err)
	}

	// Process image with enhanced algorithm
	result, err := manager.twoDProcessor.Process(src)
	if err != nil {
		return nil, fmt.Errorf("2D Otsu enhanced processing failed: %w", err)
	}

	processingTime := time.Since(startTime)

	// Gather comprehensive statistics
	statistics := manager.gather2DOtsuStatisticsEnhanced(manager.twoDProcessor, &result, &src)

	// Calculate quality metrics
	qualityMetrics := manager.calculate2DOtsuQualityMetrics(&src, &result, mergedParams)

	// Gather debug information
	debugInfo := manager.gather2DOtsuDebugInfo(manager.twoDProcessor, processingTime)

	return &ProcessingResultEnhanced{
		Result:         result,
		Algorithm:      "2D Otsu Enhanced",
		Parameters:     mergedParams,
		Statistics:     statistics,
		ProcessTime:    float64(processingTime.Nanoseconds()) / 1e6,
		QualityMetrics: qualityMetrics,
		DebugInfo:      debugInfo,
	}, nil
}

// ProcessIterativeTriclassEnhanced executes enhanced iterative triclass thresholding
func (manager *AlgorithmManagerEnhanced) ProcessIterativeTriclassEnhanced(src gocv.Mat, params map[string]interface{}) (*ProcessingResultEnhanced, error) {
	manager.mu.RLock()
	mergedParams := manager.mergeParameters("Iterative Triclass", params)
	manager.mu.RUnlock()

	startTime := time.Now()

	// Create or reuse processor instance
	if manager.triclassProcessor == nil {
		manager.triclassProcessor = NewIterativeTriclassProcessor(mergedParams)
	}

	// Validate parameters before processing
	if err := manager.triclassProcessor.ValidateParameters(); err != nil {
		return nil, fmt.Errorf("Iterative Triclass parameter validation failed: %w", err)
	}

	// Process image with enhanced algorithm
	result, err := manager.triclassProcessor.Process(src)
	if err != nil {
		return nil, fmt.Errorf("Iterative Triclass enhanced processing failed: %w", err)
	}

	processingTime := time.Since(startTime)

	// Gather comprehensive statistics
	statistics := manager.gatherTriclassStatisticsEnhanced(manager.triclassProcessor, &result, &src)

	// Calculate quality metrics
	qualityMetrics := manager.calculateTriclassQualityMetrics(&src, &result, mergedParams)

	// Gather debug information including convergence analysis
	debugInfo := manager.gatherTriclassDebugInfo(manager.triclassProcessor, processingTime)

	return &ProcessingResultEnhanced{
		Result:         result,
		Algorithm:      "Iterative Triclass Enhanced",
		Parameters:     mergedParams,
		Statistics:     statistics,
		ProcessTime:    float64(processingTime.Nanoseconds()) / 1e6,
		QualityMetrics: qualityMetrics,
		DebugInfo:      debugInfo,
	}, nil
}

// gather2DOtsuStatisticsEnhanced collects comprehensive statistics from enhanced 2D Otsu processor
func (manager *AlgorithmManagerEnhanced) gather2DOtsuStatisticsEnhanced(processor *TwoDOtsuProcessor, result, original *gocv.Mat) map[string]interface{} {
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

	// Algorithm-specific statistics
	processingStats := processor.GetProcessingStatistics()
	for key, value := range processingStats {
		statistics[key] = value
	}

	statistics["algorithm_type"] = "2D_Otsu_Enhanced"
	statistics["enhancement_features"] = []string{"modular_design", "mathematical_validation", "quality_analysis"}

	return statistics
}

// gatherTriclassStatisticsEnhanced collects comprehensive statistics from enhanced iterative triclass processor
func (manager *AlgorithmManagerEnhanced) gatherTriclassStatisticsEnhanced(processor *IterativeTriclassProcessor, result, original *gocv.Mat) map[string]interface{} {
	// Get base statistics from processor
	statistics := processor.GetProcessingStatistics()

	// Add manager-level statistics
	statistics["output_dimensions"] = fmt.Sprintf("%dx%d", result.Cols(), result.Rows())
	statistics["output_channels"] = result.Channels()
	statistics["algorithm_type"] = "Iterative_Triclass_Enhanced"
	statistics["enhancement_features"] = []string{"modular_design", "convergence_monitoring", "mathematical_validation"}

	// Add convergence information
	convergenceHistory := processor.GetConvergenceHistory()
	if len(convergenceHistory) > 0 {
		statistics["convergence_history_length"] = len(convergenceHistory)
		finalRecord := convergenceHistory[len(convergenceHistory)-1]
		statistics["final_threshold"] = finalRecord.Threshold
		statistics["final_convergence_value"] = finalRecord.ConvergenceValue
		statistics["final_tbd_fraction"] = finalRecord.TBDFraction
		statistics["actual_iterations"] = finalRecord.Iteration + 1
	}

	return statistics
}

// calculate2DOtsuQualityMetrics computes quality assessment metrics for 2D Otsu results
func (manager *AlgorithmManagerEnhanced) calculate2DOtsuQualityMetrics(original, result *gocv.Mat, params map[string]interface{}) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Calculate basic quality metrics
	foregroundPixels := gocv.CountNonZero(*result)
	totalPixels := result.Rows() * result.Cols()
	foregroundRatio := float64(foregroundPixels) / float64(totalPixels)

	metrics["foreground_ratio"] = foregroundRatio
	metrics["background_ratio"] = 1.0 - foregroundRatio

	// Class balance entropy (higher is more balanced)
	if foregroundRatio > 0 && foregroundRatio < 1 {
		entropy := -(foregroundRatio*manager.log2(foregroundRatio) + (1.0-foregroundRatio)*manager.log2(1.0-foregroundRatio))
		metrics["class_balance_entropy"] = entropy
	} else {
		metrics["class_balance_entropy"] = 0.0
	}

	// Connectivity analysis
	connectivity := manager.analyzeConnectivity(*result)
	metrics["connectivity_components"] = connectivity["component_count"]
	metrics["largest_component_ratio"] = connectivity["largest_ratio"]

	// Edge coherence (measure of edge preservation)
	edgeCoherence := manager.calculateEdgeCoherence(*original, *result)
	metrics["edge_coherence"] = edgeCoherence

	metrics["quality_assessment"] = manager.assessOverallQuality(metrics)

	return metrics
}

// calculateTriclassQualityMetrics computes quality assessment metrics for iterative triclass results
func (manager *AlgorithmManagerEnhanced) calculateTriclassQualityMetrics(original, result *gocv.Mat, params map[string]interface{}) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Basic quality metrics
	foregroundPixels := gocv.CountNonZero(*result)
	totalPixels := result.Rows() * result.Cols()
	foregroundRatio := float64(foregroundPixels) / float64(totalPixels)

	metrics["foreground_ratio"] = foregroundRatio
	metrics["background_ratio"] = 1.0 - foregroundRatio

	// Iterative quality specific metrics
	metrics["processing_efficiency"] = manager.calculateProcessingEfficiency(params)

	// Boundary smoothness
	boundarySmootness := manager.calculateBoundarySmoothness(*result)
	metrics["boundary_smoothness"] = boundarySmootness

	// Regional consistency
	regionalConsistency := manager.calculateRegionalConsistency(*result)
	metrics["regional_consistency"] = regionalConsistency

	metrics["quality_assessment"] = manager.assessOverallQuality(metrics)

	return metrics
}

// gather2DOtsuDebugInfo collects debug information for 2D Otsu processing
func (manager *AlgorithmManagerEnhanced) gather2DOtsuDebugInfo(processor *TwoDOtsuProcessor, processingTime time.Duration) map[string]interface{} {
	debugInfo := make(map[string]interface{})

	debugInfo["processing_time_ms"] = float64(processingTime.Nanoseconds()) / 1e6
	debugInfo["algorithm_version"] = "enhanced_modular"
	debugInfo["components_used"] = []string{"preprocessing", "histogram_builder", "math_processor"}

	// Memory usage estimation
	debugInfo["estimated_memory_mb"] = manager.estimateMemoryUsage("2D Otsu")

	return debugInfo
}

// gatherTriclassDebugInfo collects debug information for iterative triclass processing
func (manager *AlgorithmManagerEnhanced) gatherTriclassDebugInfo(processor *IterativeTriclassProcessor, processingTime time.Duration) map[string]interface{} {
	debugInfo := make(map[string]interface{})

	debugInfo["processing_time_ms"] = float64(processingTime.Nanoseconds()) / 1e6
	debugInfo["algorithm_version"] = "enhanced_modular"
	debugInfo["components_used"] = []string{"math_processor", "convergence_monitor", "postprocessor"}

	// Convergence analysis
	convergenceHistory := processor.GetConvergenceHistory()
	if len(convergenceHistory) > 0 {
		debugInfo["convergence_iterations"] = len(convergenceHistory)
		debugInfo["convergence_efficiency"] = manager.calculateConvergenceEfficiency(convergenceHistory)
	}

	// Memory usage estimation
	debugInfo["estimated_memory_mb"] = manager.estimateMemoryUsage("Iterative Triclass")

	return debugInfo
}

// Utility functions for quality analysis
func (manager *AlgorithmManagerEnhanced) log2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log(x) / math.Log(2)
}

func (manager *AlgorithmManagerEnhanced) analyzeConnectivity(binary gocv.Mat) map[string]interface{} {
	// Use connected components analysis
	labels := gocv.NewMat()
	defer labels.Close()

	numComponents := gocv.ConnectedComponents(binary, &labels)

	connectivity := make(map[string]interface{})
	connectivity["component_count"] = numComponents

	// Calculate largest component ratio (simplified)
	if numComponents > 0 {
		connectivity["largest_ratio"] = 1.0 / float64(numComponents) // Simplified
	} else {
		connectivity["largest_ratio"] = 0.0
	}

	return connectivity
}

func (manager *AlgorithmManagerEnhanced) calculateEdgeCoherence(original, result gocv.Mat) float64 {
	// Calculate edge preservation metric (simplified implementation)
	originalEdges := gocv.NewMat()
	defer originalEdges.Close()
	resultEdges := gocv.NewMat()
	defer resultEdges.Close()

	// Apply Canny edge detection
	gocv.Canny(original, &originalEdges, 50, 150)
	gocv.Canny(result, &resultEdges, 50, 150)

	// Calculate overlap
	intersection := gocv.NewMat()
	defer intersection.Close()
	gocv.BitwiseAnd(originalEdges, resultEdges, &intersection)

	intersectionPixels := gocv.CountNonZero(intersection)
	originalEdgePixels := gocv.CountNonZero(originalEdges)

	if originalEdgePixels > 0 {
		return float64(intersectionPixels) / float64(originalEdgePixels)
	}
	return 0.0
}

func (manager *AlgorithmManagerEnhanced) calculateBoundarySmoothness(binary gocv.Mat) float64 {
	// Calculate boundary smoothness using morphological operations
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	opened := gocv.NewMat()
	defer opened.Close()
	closed := gocv.NewMat()
	defer closed.Close()

	gocv.MorphologyEx(binary, &opened, gocv.MorphOpen, kernel)
	gocv.MorphologyEx(opened, &closed, gocv.MorphClose, kernel)

	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(binary, closed, &diff)

	diffPixels := gocv.CountNonZero(diff)
	totalPixels := binary.Rows() * binary.Cols()

	// Higher smoothness = fewer differences
	return 1.0 - (float64(diffPixels) / float64(totalPixels))
}

func (manager *AlgorithmManagerEnhanced) calculateRegionalConsistency(binary gocv.Mat) float64 {
	// Calculate regional consistency using variance of local regions
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Point{X: 5, Y: 5})
	defer kernel.Close()

	filtered := gocv.NewMat()
	defer filtered.Close()

	// Apply morphological filter
	gocv.MorphologyEx(binary, &filtered, gocv.MorphClose, kernel)

	// Calculate similarity
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(binary, filtered, &diff)

	diffPixels := gocv.CountNonZero(diff)
	totalPixels := binary.Rows() * binary.Cols()

	return 1.0 - (float64(diffPixels) / float64(totalPixels))
}

func (manager *AlgorithmManagerEnhanced) calculateProcessingEfficiency(params map[string]interface{}) float64 {
	// Calculate efficiency based on parameter settings
	efficiency := 1.0

	if quality, ok := params["quality"].(string); ok && quality == "Best" {
		efficiency *= 0.8 // Best quality is less efficient but higher quality
	}

	if maxIter, ok := params["max_iterations"].(int); ok {
		// Fewer iterations = higher efficiency
		efficiency *= 1.0 - (float64(maxIter)/20.0)*0.2
	}

	return efficiency
}

func (manager *AlgorithmManagerEnhanced) calculateConvergenceEfficiency(history []ConvergenceRecord) float64 {
	if len(history) == 0 {
		return 0.0
	}

	// Calculate efficiency based on convergence rate
	totalIterations := len(history)
	finalTBDFraction := history[len(history)-1].TBDFraction

	// Higher efficiency if fewer iterations and lower final TBD fraction
	iterationEfficiency := 1.0 - (float64(totalIterations) / 20.0)
	tbdEfficiency := 1.0 - finalTBDFraction

	return 0.6*iterationEfficiency + 0.4*tbdEfficiency
}

func (manager *AlgorithmManagerEnhanced) assessOverallQuality(metrics map[string]interface{}) string {
	score := 0.0
	weights := 0.0

	// Weighted quality assessment
	if entropy, ok := metrics["class_balance_entropy"].(float64); ok {
		score += entropy * 0.3
		weights += 0.3
	}

	if coherence, ok := metrics["edge_coherence"].(float64); ok {
		score += coherence * 0.4
		weights += 0.4
	}

	if smoothness, ok := metrics["boundary_smoothness"].(float64); ok {
		score += smoothness * 0.3
		weights += 0.3
	}

	if weights > 0 {
		finalScore := score / weights
		if finalScore >= 0.8 {
			return "excellent"
		} else if finalScore >= 0.6 {
			return "good"
		} else if finalScore >= 0.4 {
			return "fair"
		} else {
			return "poor"
		}
	}

	return "unknown"
}

func (manager *AlgorithmManagerEnhanced) estimateMemoryUsage(algorithm string) float64 {
	// Estimate memory usage in MB (simplified)
	switch algorithm {
	case "2D Otsu":
		return 15.0 // Histogram + neighborhood calculations
	case "Iterative Triclass":
		return 8.0 // Simpler per-iteration memory usage
	default:
		return 10.0
	}
}

// mergeParameters combines default parameters with user-provided overrides
func (manager *AlgorithmManagerEnhanced) mergeParameters(algorithm string, overrides map[string]interface{}) map[string]interface{} {
	// Start with defaults
	merged := make(map[string]interface{})
	if defaults, exists := manager.parameters[algorithm]; exists {
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

// ValidateParametersEnhanced checks if parameters are within acceptable ranges with enhanced validation
func (manager *AlgorithmManagerEnhanced) ValidateParametersEnhanced(algorithm string, params map[string]interface{}) error {
	switch algorithm {
	case "2D Otsu":
		return manager.validate2DOtsuParametersEnhanced(params)
	case "Iterative Triclass":
		return manager.validateTriclassParametersEnhanced(params)
	default:
		return fmt.Errorf("unknown algorithm: %s", algorithm)
	}
}

// validate2DOtsuParametersEnhanced validates 2D Otsu parameter ranges with mathematical justification
func (manager *AlgorithmManagerEnhanced) validate2DOtsuParametersEnhanced(params map[string]interface{}) error {
	// Window size validation with mathematical bounds
	if windowSize, ok := params["window_size"].(int); ok {
		if windowSize < 3 || windowSize > 31 || windowSize%2 == 0 {
			return fmt.Errorf("window_size must be odd and between 3-31, got %d", windowSize)
		}
	}

	// Histogram bins validation with power-of-2 preference
	if histBins, ok := params["histogram_bins"].(int); ok {
		if histBins < 8 || histBins > 512 {
			return fmt.Errorf("histogram_bins must be between 8-512, got %d", histBins)
		}
		// Check if power of 2 for optimal performance
		if histBins&(histBins-1) != 0 {
			manager.debugManager.LogWarning("Parameter Validation",
				fmt.Sprintf("histogram_bins %d is not a power of 2, performance may be suboptimal", histBins))
		}
	}

	// Pixel weight factor validation with mathematical bounds
	if pixelWeight, ok := params["pixel_weight_factor"].(float64); ok {
		if pixelWeight < 0.0 || pixelWeight > 1.0 {
			return fmt.Errorf("pixel_weight_factor must be between 0.0-1.0, got %f", pixelWeight)
		}
	}

	// Smoothing sigma validation with reasonable bounds
	if sigma, ok := params["smoothing_sigma"].(float64); ok {
		if sigma < 0.0 || sigma > 10.0 {
			return fmt.Errorf("smoothing_sigma must be between 0.0-10.0, got %f", sigma)
		}
	}

	// Neighbourhood metric validation
	if metric, ok := params["neighbourhood_metric"].(string); ok {
		validMetrics := []string{"mean", "median", "gaussian"}
		isValid := false
		for _, valid := range validMetrics {
			if metric == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("neighbourhood_metric must be one of %v, got %s", validMetrics, metric)
		}
	}

	return nil
}

// validateTriclassParametersEnhanced validates iterative triclass parameter ranges with convergence analysis
func (manager *AlgorithmManagerEnhanced) validateTriclassParametersEnhanced(params map[string]interface{}) error {
	// Max iterations validation with computational bounds
	if maxIter, ok := params["max_iterations"].(int); ok {
		if maxIter < 1 || maxIter > 100 {
			return fmt.Errorf("max_iterations must be between 1-100, got %d", maxIter)
		}
	}

	// Convergence epsilon validation with mathematical significance
	if epsilon, ok := params["convergence_epsilon"].(float64); ok {
		if epsilon < 0.01 || epsilon > 50.0 {
			return fmt.Errorf("convergence_epsilon must be between 0.01-50.0, got %f", epsilon)
		}
	}

	// Minimum TBD fraction validation with convergence theory
	if minTBD, ok := params["minimum_tbd_fraction"].(float64); ok {
		if minTBD < 0.0001 || minTBD > 0.5 {
			return fmt.Errorf("minimum_tbd_fraction must be between 0.0001-0.5, got %f", minTBD)
		}
	}

	// Gap factor validation with mathematical bounds
	if gapFactor, ok := params["lower_upper_gap_factor"].(float64); ok {
		if gapFactor < 0.0 || gapFactor > 1.0 {
			return fmt.Errorf("lower_upper_gap_factor must be between 0.0-1.0, got %f", gapFactor)
		}
	}

	// Initial threshold method validation
	if method, ok := params["initial_threshold_method"].(string); ok {
		validMethods := []string{"otsu", "mean", "median"}
		isValid := false
		for _, valid := range validMethods {
			if method == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("initial_threshold_method must be one of %v, got %s", validMethods, method)
		}
	}

	return nil
}

// Legacy compatibility methods for existing pipeline
func (manager *AlgorithmManagerEnhanced) Process2DOtsu(src gocv.Mat, params map[string]interface{}) (*ProcessingResult, error) {
	enhanced, err := manager.Process2DOtsuEnhanced(src, params)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format
	return &ProcessingResult{
		Result:      enhanced.Result,
		Algorithm:   enhanced.Algorithm,
		Parameters:  enhanced.Parameters,
		Statistics:  enhanced.Statistics,
		ProcessTime: enhanced.ProcessTime,
	}, nil
}

func (manager *AlgorithmManagerEnhanced) ProcessIterativeTriclass(src gocv.Mat, params map[string]interface{}) (*ProcessingResult, error) {
	enhanced, err := manager.ProcessIterativeTriclassEnhanced(src, params)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format
	return &ProcessingResult{
		Result:      enhanced.Result,
		Algorithm:   enhanced.Algorithm,
		Parameters:  enhanced.Parameters,
		Statistics:  enhanced.Statistics,
		ProcessTime: enhanced.ProcessTime,
	}, nil
}

// Legacy compatibility methods
func (manager *AlgorithmManagerEnhanced) SetCurrentAlgorithm(algorithm string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if _, exists := manager.parameters[algorithm]; exists {
		manager.currentAlgorithm = algorithm
		manager.debugManager.LogAlgorithmSwitch("", algorithm)
	}
}

func (manager *AlgorithmManagerEnhanced) GetCurrentAlgorithm() string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return manager.currentAlgorithm
}

func (manager *AlgorithmManagerEnhanced) GetParameters(algorithm string) map[string]interface{} {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	if params, exists := manager.parameters[algorithm]; exists {
		result := make(map[string]interface{})
		for k, v := range params {
			result[k] = v
		}
		return result
	}

	return make(map[string]interface{})
}

func (manager *AlgorithmManagerEnhanced) SetParameter(algorithm, name string, value interface{}) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if params, exists := manager.parameters[algorithm]; exists {
		oldValue := params[name]
		params[name] = value
		manager.debugManager.LogParameterChange(algorithm, name, oldValue, value)
	}
}

func (manager *AlgorithmManagerEnhanced) ValidateParameters(algorithm string, params map[string]interface{}) error {
	return manager.ValidateParametersEnhanced(algorithm, params)
}

func (manager *AlgorithmManagerEnhanced) GetAvailableAlgorithms() []string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()

	algorithms := make([]string, 0, len(manager.parameters))
	for algorithm := range manager.parameters {
		algorithms = append(algorithms, algorithm)
	}
	return algorithms
}

func (manager *AlgorithmManagerEnhanced) GetAllParameters(algorithm string) map[string]interface{} {
	return manager.GetParameters(algorithm)
}

func (manager *AlgorithmManagerEnhanced) GetPerformanceReport() string {
	return manager.debugManager.GetPerformanceReport()
}

// Cleanup releases resources used by the enhanced algorithm manager
func (manager *AlgorithmManagerEnhanced) Cleanup() {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.twoDProcessor != nil {
		manager.twoDProcessor.Cleanup()
		manager.twoDProcessor = nil
	}

	if manager.triclassProcessor != nil {
		manager.triclassProcessor.Cleanup()
		manager.triclassProcessor = nil
	}

	if manager.debugManager != nil {
		manager.debugManager.Cleanup()
	}
}

// NewAlgorithmManager creates a legacy-compatible algorithm manager
func NewAlgorithmManager() *AlgorithmManagerEnhanced {
	return NewAlgorithmManagerEnhanced()
}
