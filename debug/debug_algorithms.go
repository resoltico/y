package debug

import (
	"fmt"
	"time"
)

// Global debug toggle for algorithm execution debugging
var EnableAlgorithmDebug = false

// AlgorithmDebugInfo contains detailed algorithm execution information
type AlgorithmDebugInfo struct {
	AlgorithmName    string
	Parameters       map[string]interface{}
	ProcessingSteps  []string
	TimingData       map[string]time.Duration
	MemoryUsage      map[string]int64
	InputDimensions  string
	OutputDimensions string
	TotalProcessTime time.Duration
}

// LogAlgorithmStart logs the beginning of algorithm execution
func (dm *Manager) LogAlgorithmStart(algorithm string, params map[string]interface{}) {
	if !EnableAlgorithmDebug {
		return
	}

	paramStr := formatParameters(params)
	LogInfo("AlgorithmDebug", fmt.Sprintf("Starting %s with parameters: %s", algorithm, paramStr))
}

// LogAlgorithmStep logs individual processing steps within an algorithm
func (dm *Manager) LogAlgorithmStep(algorithm, step string, duration time.Duration) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("%s - %s completed in %v", algorithm, step, duration))
}

// LogAlgorithmComplete logs algorithm completion with comprehensive statistics
func (dm *Manager) LogAlgorithmComplete(algorithm string, totalDuration time.Duration, outputSize string) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("%s completed - Duration: %v, Output: %s",
		algorithm, totalDuration, outputSize))
}

// LogThresholdCalculation logs threshold computation details
func (dm *Manager) LogThresholdCalculation(algorithm string, threshold interface{}, method string) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("%s threshold calculation - Method: %s, Result: %v",
		algorithm, method, threshold))
}

// LogAlgorithmSwitch logs algorithm selection changes
func (dm *Manager) LogAlgorithmSwitch(fromAlgorithm, toAlgorithm string) {
	if !EnableAlgorithmDebug {
		return
	}
	if fromAlgorithm == "" {
		LogInfo("AlgorithmDebug", fmt.Sprintf("Algorithm initialized: %s", toAlgorithm))
	} else {
		LogInfo("AlgorithmDebug", fmt.Sprintf("Algorithm switched: %s -> %s", fromAlgorithm, toAlgorithm))
	}
}

// LogParameterValidation logs parameter validation results
func (dm *Manager) LogParameterValidation(algorithm string, params map[string]interface{}, isValid bool, errorMsg string) {
	if !EnableAlgorithmDebug {
		return
	}

	status := "PASSED"
	if !isValid {
		status = fmt.Sprintf("FAILED: %s", errorMsg)
	}

	LogInfo("AlgorithmDebug", fmt.Sprintf("%s parameter validation %s - Parameters: %s",
		algorithm, status, formatParameters(params)))
}

// LogHistogramStatistics logs histogram construction and processing statistics
func (dm *Manager) LogHistogramStatistics(algorithm string, bins int, totalCount int, smoothed bool, normalized bool) {
	if !EnableAlgorithmDebug {
		return
	}

	LogInfo("AlgorithmDebug", fmt.Sprintf("%s histogram - Bins: %d, Total Count: %d, Smoothed: %t, Normalized: %t",
		algorithm, bins, totalCount, smoothed, normalized))
}

// LogConvergenceInfo logs iterative algorithm convergence information
func (dm *Manager) LogConvergenceInfo(algorithm string, iteration int, threshold float64, convergenceValue float64, converged bool) {
	if !EnableAlgorithmDebug {
		return
	}

	status := "continuing"
	if converged {
		status = "converged"
	}

	LogInfo("AlgorithmDebug", fmt.Sprintf("%s iteration %d - Threshold: %.3f, Convergence: %.6f (%s)",
		algorithm, iteration, threshold, convergenceValue, status))
}

// LogMemoryUsage logs algorithm memory usage statistics
func (dm *Manager) LogMemoryUsage(algorithm string, component string, beforeBytes int64, afterBytes int64) {
	if !EnableAlgorithmDebug {
		return
	}

	delta := afterBytes - beforeBytes
	deltaStr := ""
	if delta > 0 {
		deltaStr = fmt.Sprintf(" (+%d bytes)", delta)
	} else if delta < 0 {
		deltaStr = fmt.Sprintf(" (%d bytes)", delta)
	}

	LogInfo("AlgorithmDebug", fmt.Sprintf("%s %s memory - Before: %d bytes, After: %d bytes%s",
		algorithm, component, beforeBytes, afterBytes, deltaStr))
}

// LogAlgorithmError logs algorithm execution errors with context
func (dm *Manager) LogAlgorithmError(algorithm string, step string, err error, context map[string]interface{}) {
	if !EnableAlgorithmDebug {
		return
	}

	contextStr := ""
	if context != nil {
		contextStr = fmt.Sprintf(" - Context: %s", formatParameters(context))
	}

	LogError("AlgorithmDebug", fmt.Sprintf("%s error in %s: %v%s", algorithm, step, err, contextStr))
}

// LogPerformanceMetrics logs detailed performance analysis
func (dm *Manager) LogPerformanceMetrics(algorithm string, metrics map[string]interface{}) {
	if !EnableAlgorithmDebug {
		return
	}

	LogInfo("AlgorithmDebug", fmt.Sprintf("%s performance metrics: %s", algorithm, formatParameters(metrics)))
}

// LogAlgorithmComparison logs comparison between different algorithm results
func (dm *Manager) LogAlgorithmComparison(algorithm1, algorithm2 string, metric string, value1, value2 float64) {
	if !EnableAlgorithmDebug {
		return
	}

	betterAlgorithm := algorithm1
	if value2 > value1 {
		betterAlgorithm = algorithm2
	}

	LogInfo("AlgorithmDebug", fmt.Sprintf("Algorithm comparison - %s: %s=%.4f, %s=%.4f (Better: %s)",
		metric, algorithm1, value1, algorithm2, value2, betterAlgorithm))
}

// formatParameters converts parameter map to readable string
func formatParameters(params map[string]interface{}) string {
	if params == nil || len(params) == 0 {
		return "{}"
	}

	result := "{"
	first := true
	for key, value := range params {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s: %v", key, value)
		first = false
	}
	result += "}"

	return result
}

// CreateAlgorithmDebugInfo creates a comprehensive debug information structure
func (dm *Manager) CreateAlgorithmDebugInfo(algorithm string, params map[string]interface{}) *AlgorithmDebugInfo {
	return &AlgorithmDebugInfo{
		AlgorithmName:    algorithm,
		Parameters:       params,
		ProcessingSteps:  make([]string, 0),
		TimingData:       make(map[string]time.Duration),
		MemoryUsage:      make(map[string]int64),
		TotalProcessTime: 0,
	}
}

// AddProcessingStep adds a step to the algorithm debug info
func (info *AlgorithmDebugInfo) AddProcessingStep(step string, duration time.Duration) {
	info.ProcessingSteps = append(info.ProcessingSteps, step)
	info.TimingData[step] = duration
	info.TotalProcessTime += duration
}

// SetDimensions sets input and output dimensions for the algorithm
func (info *AlgorithmDebugInfo) SetDimensions(inputDims, outputDims string) {
	info.InputDimensions = inputDims
	info.OutputDimensions = outputDims
}

// AddMemoryUsage adds memory usage information for a component
func (info *AlgorithmDebugInfo) AddMemoryUsage(component string, bytes int64) {
	info.MemoryUsage[component] = bytes
}

// GenerateReport creates a comprehensive debug report
func (info *AlgorithmDebugInfo) GenerateReport() string {
	report := fmt.Sprintf(`Algorithm Debug Report: %s
Input Dimensions: %s
Output Dimensions: %s
Total Processing Time: %v
Parameters: %s

Processing Steps:`,
		info.AlgorithmName,
		info.InputDimensions,
		info.OutputDimensions,
		info.TotalProcessTime,
		formatParameters(info.Parameters))

	for i, step := range info.ProcessingSteps {
		if duration, exists := info.TimingData[step]; exists {
			report += fmt.Sprintf("\n  %d. %s (%v)", i+1, step, duration)
		} else {
			report += fmt.Sprintf("\n  %d. %s", i+1, step)
		}
	}

	if len(info.MemoryUsage) > 0 {
		report += "\n\nMemory Usage:"
		for component, bytes := range info.MemoryUsage {
			report += fmt.Sprintf("\n  %s: %d bytes", component, bytes)
		}
	}

	return report
}
