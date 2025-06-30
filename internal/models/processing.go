package models

import (
	"fmt"
	"sync"
	"time"
)

// ProcessingState represents the current state of image processing
type ProcessingState struct {
	IsActive          bool
	Algorithm         string
	CurrentStage      string
	Progress          float64
	StartTime         time.Time
	EstimatedDuration time.Duration
	CancellationToken CancellationToken
}

// CancellationToken provides a way to cancel ongoing processing
type CancellationToken struct {
	cancelled bool
	mu        sync.RWMutex
}

// NewCancellationToken creates a new cancellation token
func NewCancellationToken() *CancellationToken {
	return &CancellationToken{}
}

// Cancel marks the token as cancelled
func (ct *CancellationToken) Cancel() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.cancelled = true
}

// IsCancelled returns true if the token has been cancelled
func (ct *CancellationToken) IsCancelled() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.cancelled
}

// Reset clears the cancellation state
func (ct *CancellationToken) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.cancelled = false
}

// AlgorithmParameters contains algorithm-specific configuration
type AlgorithmParameters struct {
	Name       string
	Parameters map[string]interface{}
	Defaults   map[string]interface{}
	Ranges     map[string]ParameterRange
}

// ParameterRange defines valid range for a parameter
type ParameterRange struct {
	Min     interface{}
	Max     interface{}
	Step    interface{}
	Options []interface{}
}

// ProcessingConfiguration manages processing settings
type ProcessingConfiguration struct {
	mu                  sync.RWMutex
	currentAlgorithm    string
	algorithmParameters map[string]AlgorithmParameters
	globalSettings      map[string]interface{}
	performanceSettings PerformanceSettings
}

// PerformanceSettings contains performance-related configuration
type PerformanceSettings struct {
	MaxWorkers            int
	MemoryLimit           int64
	EnableParallelization bool
	UseGPUAcceleration    bool
	CacheSize             int
	GCThreshold           float64
}

// NewProcessingConfiguration creates a new processing configuration
func NewProcessingConfiguration() *ProcessingConfiguration {
	config := &ProcessingConfiguration{
		algorithmParameters: make(map[string]AlgorithmParameters),
		globalSettings:      make(map[string]interface{}),
		performanceSettings: PerformanceSettings{
			MaxWorkers:            4,
			MemoryLimit:           4 * 1024 * 1024 * 1024, // 4GB
			EnableParallelization: true,
			UseGPUAcceleration:    false,
			CacheSize:             100,
			GCThreshold:           0.8,
		},
	}

	config.initializeDefaultAlgorithms()
	config.initializeGlobalSettings()

	return config
}

// initializeDefaultAlgorithms sets up default algorithm configurations
func (pc *ProcessingConfiguration) initializeDefaultAlgorithms() {
	// 2D Otsu algorithm parameters
	pc.algorithmParameters["2D Otsu"] = AlgorithmParameters{
		Name: "2D Otsu",
		Parameters: map[string]interface{}{
			"window_size":            7,
			"histogram_bins":         0,
			"smoothing_strength":     1.0,
			"noise_robustness":       true,
			"gaussian_preprocessing": true,
			"use_clahe":              false,
			"clahe_clip_limit":       3.0,
			"clahe_tile_size":        8,
			"guided_filtering":       false,
			"guided_radius":          4,
			"guided_epsilon":         0.05,
			"parallel_processing":    true,
		},
		Defaults: map[string]interface{}{
			"window_size":            7,
			"histogram_bins":         0,
			"smoothing_strength":     1.0,
			"noise_robustness":       true,
			"gaussian_preprocessing": true,
			"use_clahe":              false,
			"clahe_clip_limit":       3.0,
			"clahe_tile_size":        8,
			"guided_filtering":       false,
			"guided_radius":          4,
			"guided_epsilon":         0.05,
			"parallel_processing":    true,
		},
		Ranges: map[string]ParameterRange{
			"window_size":        {Min: 3, Max: 21, Step: 2},
			"histogram_bins":     {Min: 0, Max: 256, Step: 1},
			"smoothing_strength": {Min: 0.0, Max: 5.0, Step: 0.1},
			"clahe_clip_limit":   {Min: 1.0, Max: 10.0, Step: 0.1},
			"clahe_tile_size":    {Min: 4, Max: 16, Step: 2},
			"guided_radius":      {Min: 1, Max: 10, Step: 1},
			"guided_epsilon":     {Min: 0.01, Max: 1.0, Step: 0.01},
		},
	}

	// Iterative Triclass algorithm parameters
	pc.algorithmParameters["Iterative Triclass"] = AlgorithmParameters{
		Name: "Iterative Triclass",
		Parameters: map[string]interface{}{
			"initial_threshold_method": "otsu",
			"histogram_bins":           0,
			"convergence_precision":    1.0,
			"max_iterations":           8,
			"minimum_tbd_fraction":     0.01,
			"class_separation":         0.5,
			"preprocessing":            true,
			"result_cleanup":           true,
			"preserve_borders":         false,
			"noise_robustness":         true,
			"guided_filtering":         true,
			"guided_radius":            6,
			"guided_epsilon":           0.15,
			"parallel_processing":      true,
		},
		Defaults: map[string]interface{}{
			"initial_threshold_method": "otsu",
			"histogram_bins":           0,
			"convergence_precision":    1.0,
			"max_iterations":           8,
			"minimum_tbd_fraction":     0.01,
			"class_separation":         0.5,
			"preprocessing":            true,
			"result_cleanup":           true,
			"preserve_borders":         false,
			"noise_robustness":         true,
			"guided_filtering":         true,
			"guided_radius":            6,
			"guided_epsilon":           0.15,
			"parallel_processing":      true,
		},
		Ranges: map[string]ParameterRange{
			"initial_threshold_method": {Options: []interface{}{"otsu", "mean", "median", "triangle"}},
			"histogram_bins":           {Min: 0, Max: 256, Step: 1},
			"convergence_precision":    {Min: 0.5, Max: 2.0, Step: 0.1},
			"max_iterations":           {Min: 3, Max: 15, Step: 1},
			"minimum_tbd_fraction":     {Min: 0.001, Max: 0.1, Step: 0.001},
			"class_separation":         {Min: 0.1, Max: 0.8, Step: 0.05},
			"guided_radius":            {Min: 1, Max: 12, Step: 1},
			"guided_epsilon":           {Min: 0.01, Max: 1.0, Step: 0.01},
		},
	}

	pc.currentAlgorithm = "2D Otsu"
}

// initializeGlobalSettings sets up global application settings
func (pc *ProcessingConfiguration) initializeGlobalSettings() {
	pc.globalSettings = map[string]interface{}{
		"auto_preview":        true,
		"save_processing_log": true,
		"show_debug_info":     false,
		"default_save_format": "png",
		"jpeg_quality":        95,
		"enable_undo":         true,
		"max_undo_levels":     5,
		"ui_theme":            "auto",
		"ui_scale":            1.0,
	}
}

// GetCurrentAlgorithm returns the currently selected algorithm
func (pc *ProcessingConfiguration) GetCurrentAlgorithm() string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.currentAlgorithm
}

// SetCurrentAlgorithm changes the current algorithm
func (pc *ProcessingConfiguration) SetCurrentAlgorithm(algorithm string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if _, exists := pc.algorithmParameters[algorithm]; !exists {
		return NewValidationError("algorithm", algorithm, "algorithm not found")
	}

	pc.currentAlgorithm = algorithm
	return nil
}

// GetAlgorithmParameters returns parameters for the specified algorithm
func (pc *ProcessingConfiguration) GetAlgorithmParameters(algorithm string) (AlgorithmParameters, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	params, exists := pc.algorithmParameters[algorithm]
	if !exists {
		return AlgorithmParameters{}, NewValidationError("algorithm", algorithm, "algorithm not found")
	}

	// Return a copy to prevent external modification
	return pc.copyAlgorithmParameters(params), nil
}

// SetAlgorithmParameter updates a specific parameter for an algorithm
func (pc *ProcessingConfiguration) SetAlgorithmParameter(algorithm, paramName string, value interface{}) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	params, exists := pc.algorithmParameters[algorithm]
	if !exists {
		return NewValidationError("algorithm", algorithm, "algorithm not found")
	}

	// Validate parameter
	if err := pc.validateParameter(params, paramName, value); err != nil {
		return err
	}

	// Update parameter
	params.Parameters[paramName] = value
	pc.algorithmParameters[algorithm] = params

	return nil
}

// GetAvailableAlgorithms returns list of available algorithms
func (pc *ProcessingConfiguration) GetAvailableAlgorithms() []string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	algorithms := make([]string, 0, len(pc.algorithmParameters))
	for name := range pc.algorithmParameters {
		algorithms = append(algorithms, name)
	}

	return algorithms
}

// GetGlobalSetting retrieves a global setting value
func (pc *ProcessingConfiguration) GetGlobalSetting(key string) (interface{}, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	value, exists := pc.globalSettings[key]
	return value, exists
}

// SetGlobalSetting updates a global setting
func (pc *ProcessingConfiguration) SetGlobalSetting(key string, value interface{}) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.globalSettings[key] = value
}

// GetPerformanceSettings returns current performance settings
func (pc *ProcessingConfiguration) GetPerformanceSettings() PerformanceSettings {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.performanceSettings
}

// UpdatePerformanceSettings updates performance configuration
func (pc *ProcessingConfiguration) UpdatePerformanceSettings(settings PerformanceSettings) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.performanceSettings = settings
}

// ResetAlgorithmToDefaults resets algorithm parameters to default values
func (pc *ProcessingConfiguration) ResetAlgorithmToDefaults(algorithm string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	params, exists := pc.algorithmParameters[algorithm]
	if !exists {
		return NewValidationError("algorithm", algorithm, "algorithm not found")
	}

	// Copy defaults to current parameters
	for key, value := range params.Defaults {
		params.Parameters[key] = value
	}

	pc.algorithmParameters[algorithm] = params
	return nil
}

// validateParameter checks if a parameter value is valid
func (pc *ProcessingConfiguration) validateParameter(params AlgorithmParameters, paramName string, value interface{}) error {
	paramRange, hasRange := params.Ranges[paramName]
	if !hasRange {
		return nil // No validation rules defined
	}

	// Validate against options if provided
	if len(paramRange.Options) > 0 {
		for _, option := range paramRange.Options {
			if value == option {
				return nil
			}
		}
		return NewValidationError(paramName, value, "value not in allowed options")
	}

	// Validate numeric ranges
	switch v := value.(type) {
	case int:
		if paramRange.Min != nil {
			if min, ok := paramRange.Min.(int); ok && v < min {
				return NewValidationError(paramName, value, "value below minimum")
			}
		}
		if paramRange.Max != nil {
			if max, ok := paramRange.Max.(int); ok && v > max {
				return NewValidationError(paramName, value, "value above maximum")
			}
		}
	case float64:
		if paramRange.Min != nil {
			if min, ok := paramRange.Min.(float64); ok && v < min {
				return NewValidationError(paramName, value, "value below minimum")
			}
		}
		if paramRange.Max != nil {
			if max, ok := paramRange.Max.(float64); ok && v > max {
				return NewValidationError(paramName, value, "value above maximum")
			}
		}
	}

	return nil
}

// copyAlgorithmParameters creates a deep copy of algorithm parameters
func (pc *ProcessingConfiguration) copyAlgorithmParameters(src AlgorithmParameters) AlgorithmParameters {
	dst := AlgorithmParameters{
		Name:       src.Name,
		Parameters: make(map[string]interface{}),
		Defaults:   make(map[string]interface{}),
		Ranges:     make(map[string]ParameterRange),
	}

	// Copy parameters
	for k, v := range src.Parameters {
		dst.Parameters[k] = v
	}

	// Copy defaults
	for k, v := range src.Defaults {
		dst.Defaults[k] = v
	}

	// Copy ranges
	for k, v := range src.Ranges {
		dst.Ranges[k] = v
	}

	return dst
}

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string
	Value     interface{}
	Message   string
}

// NewValidationError creates a new validation error
func NewValidationError(parameter string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Parameter: parameter,
		Value:     value,
		Message:   message,
	}
}

// Error returns the error message
func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for parameter '%s' with value '%v': %s",
		ve.Parameter, ve.Value, ve.Message)
}

// ProcessingStateRepository manages processing state
type ProcessingStateRepository struct {
	mu    sync.RWMutex
	state ProcessingState
}

// NewProcessingStateRepository creates a new processing state repository
func NewProcessingStateRepository() *ProcessingStateRepository {
	return &ProcessingStateRepository{
		state: ProcessingState{
			IsActive:          false,
			CancellationToken: *NewCancellationToken(),
		},
	}
}

// GetState returns the current processing state
func (psr *ProcessingStateRepository) GetState() ProcessingState {
	psr.mu.RLock()
	defer psr.mu.RUnlock()
	return psr.state
}

// StartProcessing marks processing as active
func (psr *ProcessingStateRepository) StartProcessing(algorithm string) {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	psr.state = ProcessingState{
		IsActive:          true,
		Algorithm:         algorithm,
		CurrentStage:      "Initializing",
		Progress:          0.0,
		StartTime:         time.Now(),
		EstimatedDuration: 0,
		CancellationToken: *NewCancellationToken(),
	}
}

// UpdateProgress updates processing progress and stage
func (psr *ProcessingStateRepository) UpdateProgress(stage string, progress float64) {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	if psr.state.IsActive {
		psr.state.CurrentStage = stage
		psr.state.Progress = progress

		// Estimate remaining time based on progress
		if progress > 0 {
			elapsed := time.Since(psr.state.StartTime)
			estimated := time.Duration(float64(elapsed) / progress)
			psr.state.EstimatedDuration = estimated
		}
	}
}

// CompleteProcessing marks processing as complete
func (psr *ProcessingStateRepository) CompleteProcessing() {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	psr.state.IsActive = false
	psr.state.CurrentStage = "Complete"
	psr.state.Progress = 1.0
}

// CancelProcessing cancels ongoing processing
func (psr *ProcessingStateRepository) CancelProcessing() {
	psr.mu.Lock()
	defer psr.mu.Unlock()

	psr.state.CancellationToken.Cancel()
	psr.state.IsActive = false
	psr.state.CurrentStage = "Cancelled"
}

// IsProcessing returns true if processing is currently active
func (psr *ProcessingStateRepository) IsProcessing() bool {
	psr.mu.RLock()
	defer psr.mu.RUnlock()
	return psr.state.IsActive
}

// GetCancellationToken returns the current cancellation token
func (psr *ProcessingStateRepository) GetCancellationToken() *CancellationToken {
	psr.mu.RLock()
	defer psr.mu.RUnlock()
	return &psr.state.CancellationToken
}
