package debug

import (
	"fmt"
	"time"
)

// Global debug toggle for algorithm execution (set from main package)
var EnableAlgorithmDebug = false

func (dm *Manager) LogAlgorithmStart(algorithm string, params map[string]interface{}) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("Starting %s with params: %+v", algorithm, params))
}

func (dm *Manager) LogAlgorithmStep(algorithm, step string, duration time.Duration) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("%s - %s completed in %v", algorithm, step, duration))
}

func (dm *Manager) LogAlgorithmComplete(algorithm string, totalDuration time.Duration, outputSize string) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("%s completed - Duration: %v, Output: %s", 
		algorithm, totalDuration, outputSize))
}

func (dm *Manager) LogThresholdCalculation(algorithm string, threshold interface{}, method string) {
	if !EnableAlgorithmDebug {
		return
	}
	LogInfo("AlgorithmDebug", fmt.Sprintf("%s threshold calculation - Method: %s, Result: %v", 
		algorithm, method, threshold))
}
