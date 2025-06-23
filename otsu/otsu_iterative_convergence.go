package otsu

import (
	"fmt"
	"math"
	"time"

	"otsu-obliterator/debug"
)

// IterativeConvergenceMonitor tracks and analyzes convergence behavior
type IterativeConvergenceMonitor struct {
	params             map[string]interface{}
	debugManager       *debug.Manager
	convergenceHistory []ConvergenceRecord
	thresholdHistory   []float64
	tbdFractionHistory []float64
}

// ConvergenceRecord stores information about a single iteration
type ConvergenceRecord struct {
	Iteration        int
	Threshold        float64
	ConvergenceValue float64
	TBDFraction      float64
	ProcessingTime   time.Duration
	PixelCounts      TriclassPixelCounts
}

// TriclassPixelCounts holds pixel counts for each class
type TriclassPixelCounts struct {
	Foreground int
	Background int
	TBD        int
	Total      int
}

// NewIterativeConvergenceMonitor creates a new convergence monitor
func NewIterativeConvergenceMonitor(params map[string]interface{}) *IterativeConvergenceMonitor {
	return &IterativeConvergenceMonitor{
		params:             params,
		debugManager:       debug.NewManager(),
		convergenceHistory: make([]ConvergenceRecord, 0),
		thresholdHistory:   make([]float64, 0),
		tbdFractionHistory: make([]float64, 0),
	}
}

// RecordIteration records information from a single iteration
func (monitor *IterativeConvergenceMonitor) RecordIteration(iteration int, threshold float64,
	convergenceValue float64, tbdFraction float64, processingTime time.Duration,
	foregroundCount, backgroundCount, tbdCount int) {

	totalPixels := foregroundCount + backgroundCount + tbdCount

	record := ConvergenceRecord{
		Iteration:        iteration,
		Threshold:        threshold,
		ConvergenceValue: convergenceValue,
		TBDFraction:      tbdFraction,
		ProcessingTime:   processingTime,
		PixelCounts: TriclassPixelCounts{
			Foreground: foregroundCount,
			Background: backgroundCount,
			TBD:        tbdCount,
			Total:      totalPixels,
		},
	}

	monitor.convergenceHistory = append(monitor.convergenceHistory, record)
	monitor.thresholdHistory = append(monitor.thresholdHistory, threshold)
	monitor.tbdFractionHistory = append(monitor.tbdFractionHistory, tbdFraction)

	monitor.debugManager.LogTriclassIteration(iteration, threshold, convergenceValue,
		foregroundCount, backgroundCount, tbdCount)
}

// CheckConvergence determines if the iterative process should terminate
func (monitor *IterativeConvergenceMonitor) CheckConvergence() (bool, string, error) {
	if len(monitor.convergenceHistory) == 0 {
		return false, "no_iterations", nil
	}

	currentRecord := monitor.convergenceHistory[len(monitor.convergenceHistory)-1]

	// Check convergence criteria
	converged, reason := monitor.evaluateConvergenceCriteria(currentRecord)

	if converged {
		monitor.debugManager.LogInfo("Iterative Convergence",
			fmt.Sprintf("Convergence achieved: %s at iteration %d", reason, currentRecord.Iteration))
	}

	return converged, reason, nil
}

// evaluateConvergenceCriteria checks multiple convergence criteria
func (monitor *IterativeConvergenceMonitor) evaluateConvergenceCriteria(current ConvergenceRecord) (bool, string) {
	epsilon := monitor.getFloatParam("convergence_epsilon", 1.0)
	maxIterations := monitor.getIntParam("max_iterations", 10)
	minTBDFraction := monitor.getFloatParam("minimum_tbd_fraction", 0.01)

	// 1. Threshold convergence
	if current.ConvergenceValue < epsilon {
		return true, "threshold_convergence"
	}

	// 2. Maximum iterations reached
	if current.Iteration >= maxIterations-1 { // Zero-based indexing
		return true, "max_iterations"
	}

	// 3. TBD region too small
	if current.TBDFraction < minTBDFraction {
		return true, "tbd_depletion"
	}

	// 4. Oscillation detection
	if monitor.detectOscillation() {
		return true, "oscillation_detected"
	}

	// 5. Stagnation detection
	if monitor.detectStagnation() {
		return true, "stagnation_detected"
	}

	return false, "continuing"
}

// detectOscillation identifies oscillatory behavior in threshold values
func (monitor *IterativeConvergenceMonitor) detectOscillation() bool {
	if len(monitor.thresholdHistory) < 6 {
		return false // Need at least 6 iterations to detect oscillation
	}

	n := len(monitor.thresholdHistory)
	tolerance := 0.1 // Threshold for considering values "equal"

	// Check for period-2 oscillation (most common)
	period2 := true
	for i := n - 4; i < n-2; i++ {
		diff1 := math.Abs(monitor.thresholdHistory[i] - monitor.thresholdHistory[i+2])
		if diff1 > tolerance {
			period2 = false
			break
		}
	}

	if period2 {
		// Verify the oscillation is significant
		diff := math.Abs(monitor.thresholdHistory[n-1] - monitor.thresholdHistory[n-2])
		if diff > tolerance {
			monitor.debugManager.LogInfo("Iterative Convergence",
				"Period-2 oscillation detected in threshold values")
			return true
		}
	}

	// Check for period-3 oscillation
	if n >= 9 {
		period3 := true
		for i := n - 6; i < n-3; i++ {
			diff := math.Abs(monitor.thresholdHistory[i] - monitor.thresholdHistory[i+3])
			if diff > tolerance {
				period3 = false
				break
			}
		}

		if period3 {
			monitor.debugManager.LogInfo("Iterative Convergence",
				"Period-3 oscillation detected in threshold values")
			return true
		}
	}

	return false
}

// detectStagnation identifies when convergence rate becomes too slow
func (monitor *IterativeConvergenceMonitor) detectStagnation() bool {
	if len(monitor.convergenceHistory) < 5 {
		return false
	}

	// Check if convergence rate has been very slow for several iterations
	n := len(monitor.convergenceHistory)
	recentRecords := monitor.convergenceHistory[n-5:]

	slowCount := 0
	maxConvergence := monitor.getFloatParam("convergence_epsilon", 1.0) * 10 // 10x epsilon

	for _, record := range recentRecords {
		if record.ConvergenceValue < maxConvergence && record.ConvergenceValue > 0 {
			// Calculate relative improvement
			if len(monitor.convergenceHistory) > 1 {
				prevConvergence := monitor.convergenceHistory[record.Iteration-1].ConvergenceValue
				if prevConvergence > 0 {
					improvement := (prevConvergence - record.ConvergenceValue) / prevConvergence
					if improvement < 0.01 { // Less than 1% improvement
						slowCount++
					}
				}
			}
		}
	}

	// If most recent iterations show very slow improvement
	if slowCount >= 4 {
		monitor.debugManager.LogInfo("Iterative Convergence",
			fmt.Sprintf("Stagnation detected: %d of 5 recent iterations with <1%% improvement", slowCount))
		return true
	}

	return false
}

// AnalyzeConvergenceBehavior provides detailed analysis of convergence characteristics
func (monitor *IterativeConvergenceMonitor) AnalyzeConvergenceBehavior() map[string]interface{} {
	analysis := make(map[string]interface{})

	if len(monitor.convergenceHistory) == 0 {
		analysis["status"] = "no_data"
		return analysis
	}

	// Basic statistics
	analysis["total_iterations"] = len(monitor.convergenceHistory)
	analysis["final_threshold"] = monitor.thresholdHistory[len(monitor.thresholdHistory)-1]
	analysis["final_convergence"] = monitor.convergenceHistory[len(monitor.convergenceHistory)-1].ConvergenceValue
	analysis["final_tbd_fraction"] = monitor.tbdFractionHistory[len(monitor.tbdFractionHistory)-1]

	// Convergence rate analysis
	convergenceRates := monitor.calculateConvergenceRates()
	if len(convergenceRates) > 0 {
		analysis["mean_convergence_rate"] = monitor.calculateMean(convergenceRates)
		analysis["convergence_rate_variance"] = monitor.calculateVariance(convergenceRates)
	}

	// Threshold stability analysis
	thresholdVariance := monitor.calculateVariance(monitor.thresholdHistory)
	analysis["threshold_variance"] = thresholdVariance
	analysis["threshold_stability"] = monitor.classifyStability(thresholdVariance)

	// TBD fraction analysis
	tbdVariance := monitor.calculateVariance(monitor.tbdFractionHistory)
	analysis["tbd_fraction_variance"] = tbdVariance

	// Processing time analysis
	processingTimes := make([]float64, len(monitor.convergenceHistory))
	for i, record := range monitor.convergenceHistory {
		processingTimes[i] = float64(record.ProcessingTime.Nanoseconds()) / 1e6 // Convert to milliseconds
	}
	analysis["mean_processing_time_ms"] = monitor.calculateMean(processingTimes)
	analysis["total_processing_time_ms"] = monitor.calculateSum(processingTimes)

	// Oscillation and stagnation flags
	analysis["has_oscillation"] = monitor.detectOscillation()
	analysis["has_stagnation"] = monitor.detectStagnation()

	// Convergence efficiency
	analysis["convergence_efficiency"] = monitor.calculateConvergenceEfficiency()

	return analysis
}

// calculateConvergenceRates computes rate of convergence between iterations
func (monitor *IterativeConvergenceMonitor) calculateConvergenceRates() []float64 {
	if len(monitor.convergenceHistory) < 2 {
		return []float64{}
	}

	rates := make([]float64, 0)
	for i := 1; i < len(monitor.convergenceHistory); i++ {
		current := monitor.convergenceHistory[i].ConvergenceValue
		previous := monitor.convergenceHistory[i-1].ConvergenceValue

		if previous > 0 && current >= 0 {
			rate := current / previous
			rates = append(rates, rate)
		}
	}

	return rates
}

// calculateConvergenceEfficiency measures how efficiently convergence was achieved
func (monitor *IterativeConvergenceMonitor) calculateConvergenceEfficiency() float64 {
	if len(monitor.convergenceHistory) == 0 {
		return 0.0
	}

	maxIterations := monitor.getIntParam("max_iterations", 10)
	actualIterations := len(monitor.convergenceHistory)
	finalTBDFraction := monitor.tbdFractionHistory[len(monitor.tbdFractionHistory)-1]

	// Efficiency based on iterations used and TBD pixels processed
	iterationEfficiency := 1.0 - (float64(actualIterations) / float64(maxIterations))
	tbdEfficiency := 1.0 - finalTBDFraction

	// Weighted combination
	return 0.6*iterationEfficiency + 0.4*tbdEfficiency
}

// PredictRemainingIterations estimates iterations needed for convergence
func (monitor *IterativeConvergenceMonitor) PredictRemainingIterations() int {
	if len(monitor.convergenceHistory) < 2 {
		return monitor.getIntParam("max_iterations", 10)
	}

	convergenceRates := monitor.calculateConvergenceRates()
	if len(convergenceRates) == 0 {
		return monitor.getIntParam("max_iterations", 10)
	}

	// Use recent convergence rate for prediction
	recentRate := convergenceRates[len(convergenceRates)-1]
	currentConvergence := monitor.convergenceHistory[len(monitor.convergenceHistory)-1].ConvergenceValue
	epsilon := monitor.getFloatParam("convergence_epsilon", 1.0)

	if recentRate <= 0 || recentRate >= 1.0 || currentConvergence <= epsilon {
		return 0
	}

	// Geometric series prediction
	predicted := math.Log(epsilon/currentConvergence) / math.Log(recentRate)

	// Add safety margin and clamp to reasonable bounds
	estimated := int(math.Ceil(predicted * 1.2)) // 20% safety margin

	maxRemaining := monitor.getIntParam("max_iterations", 10) - len(monitor.convergenceHistory)
	if estimated > maxRemaining {
		return maxRemaining
	}
	if estimated < 0 {
		return 0
	}

	return estimated
}

// GenerateConvergenceReport creates a detailed convergence report
func (monitor *IterativeConvergenceMonitor) GenerateConvergenceReport() string {
	if len(monitor.convergenceHistory) == 0 {
		return "No convergence data available"
	}

	analysis := monitor.AnalyzeConvergenceBehavior()

	report := fmt.Sprintf(`Iterative Triclass Convergence Report:

Basic Statistics:
- Total Iterations: %d
- Final Threshold: %.3f
- Final Convergence Value: %.6f
- Final TBD Fraction: %.6f

Convergence Analysis:
- Mean Convergence Rate: %.4f
- Threshold Variance: %.6f
- Threshold Stability: %s
- Convergence Efficiency: %.3f

Behavioral Flags:
- Oscillation Detected: %t
- Stagnation Detected: %t

Performance:
- Mean Processing Time: %.2f ms
- Total Processing Time: %.2f ms

Iteration Details:`,
		analysis["total_iterations"],
		analysis["final_threshold"],
		analysis["final_convergence"],
		analysis["final_tbd_fraction"],
		analysis["mean_convergence_rate"],
		analysis["threshold_variance"],
		analysis["threshold_stability"],
		analysis["convergence_efficiency"],
		analysis["has_oscillation"],
		analysis["has_stagnation"],
		analysis["mean_processing_time_ms"],
		analysis["total_processing_time_ms"])

	// Add iteration-by-iteration details
	for i, record := range monitor.convergenceHistory {
		report += fmt.Sprintf("\n  Iteration %d: threshold=%.3f, convergence=%.6f, TBD=%.4f, time=%.1fms",
			i, record.Threshold, record.ConvergenceValue, record.TBDFraction,
			float64(record.ProcessingTime.Nanoseconds())/1e6)
	}

	return report
}

// Utility functions for statistical calculations
func (monitor *IterativeConvergenceMonitor) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (monitor *IterativeConvergenceMonitor) calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	mean := monitor.calculateMean(values)
	sumSquaredDiffs := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}
	return sumSquaredDiffs / float64(len(values)-1)
}

func (monitor *IterativeConvergenceMonitor) calculateSum(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum
}

func (monitor *IterativeConvergenceMonitor) classifyStability(variance float64) string {
	if variance < 0.01 {
		return "very_stable"
	} else if variance < 0.1 {
		return "stable"
	} else if variance < 1.0 {
		return "moderately_stable"
	} else {
		return "unstable"
	}
}

// GetConvergenceHistory returns the complete convergence history
func (monitor *IterativeConvergenceMonitor) GetConvergenceHistory() []ConvergenceRecord {
	return monitor.convergenceHistory
}

// Reset clears all convergence data for a new iteration sequence
func (monitor *IterativeConvergenceMonitor) Reset() {
	monitor.convergenceHistory = make([]ConvergenceRecord, 0)
	monitor.thresholdHistory = make([]float64, 0)
	monitor.tbdFractionHistory = make([]float64, 0)
}

// Parameter access utilities
func (monitor *IterativeConvergenceMonitor) getIntParam(name string, defaultValue int) int {
	if value, ok := monitor.params[name].(int); ok {
		return value
	}
	return defaultValue
}

func (monitor *IterativeConvergenceMonitor) getFloatParam(name string, defaultValue float64) float64 {
	if value, ok := monitor.params[name].(float64); ok {
		return value
	}
	return defaultValue
}
