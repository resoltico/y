package otsu

import (
	"sync"
)

type AlgorithmManager struct {
	mu               sync.RWMutex
	currentAlgorithm string
	parameters       map[string]map[string]interface{}
}

func NewAlgorithmManager() *AlgorithmManager {
	manager := &AlgorithmManager{
		currentAlgorithm: "2D Otsu",
		parameters:       make(map[string]map[string]interface{}),
	}
	
	manager.initializeDefaultParameters()
	return manager
}

func (am *AlgorithmManager) initializeDefaultParameters() {
	// 2D Otsu default parameters
	am.parameters["2D Otsu"] = map[string]interface{}{
		"quality":                    "Fast",
		"window_size":                7,
		"histogram_bins":             64,
		"neighbourhood_metric":       "mean",
		"pixel_weight_factor":        0.5,
		"smoothing_sigma":           1.0,
		"use_log_histogram":         false,
		"normalize_histogram":       true,
		"apply_contrast_enhancement": false,
	}
	
	// Iterative Triclass default parameters
	am.parameters["Iterative Triclass"] = map[string]interface{}{
		"quality":                   "Fast",
		"initial_threshold_method":  "otsu",
		"histogram_bins":            64,
		"convergence_epsilon":       1.0,
		"max_iterations":            10,
		"minimum_tbd_fraction":      0.01,
		"lower_upper_gap_factor":    0.5,
		"apply_preprocessing":       false,
		"apply_cleanup":             true,
		"preserve_borders":          false,
	}
}

func (am *AlgorithmManager) SetCurrentAlgorithm(algorithm string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if _, exists := am.parameters[algorithm]; exists {
		am.currentAlgorithm = algorithm
	}
}

func (am *AlgorithmManager) GetCurrentAlgorithm() string {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.currentAlgorithm
}

func (am *AlgorithmManager) GetParameters(algorithm string) map[string]interface{} {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	if params, exists := am.parameters[algorithm]; exists {
		// Return a copy to prevent external modification
		result := make(map[string]interface{})
		for k, v := range params {
			result[k] = v
		}
		return result
	}
	
	return make(map[string]interface{})
}

func (am *AlgorithmManager) GetAllParameters(algorithm string) map[string]interface{} {
	return am.GetParameters(algorithm)
}

func (am *AlgorithmManager) SetParameter(algorithm, name string, value interface{}) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if params, exists := am.parameters[algorithm]; exists {
		params[name] = value
	}
}

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