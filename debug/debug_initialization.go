package debug

import (
	"os"
)

// Global debug component toggles
// These are set during initialization based on environment variables
var (
	// Core debug categories
	EnableFormatDetection  = false // Format detection and signature analysis
	EnableImageDebug       = true  // Image loading, processing, and metrics
	EnablePerformanceDebug = true  // Timing and performance metrics
	EnableMemoryDebug      = true  // Memory usage and Mat profiling
	EnableGUIDebug         = false // GUI events and interactions
	EnableAlgorithmDebug   = false // Algorithm parameter changes and execution

	// Algorithm-specific debug categories
	EnableTriclassDebug      = false // Iterative Triclass algorithm debugging
	EnablePixelAnalysisDebug = false // Pixel-level analysis and sampling
)

// getEnvBool reads a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true"
	}
	return defaultValue
}

// InitializeDebugComponents sets up all debug components based on environment variables
func InitializeDebugComponents() {
	EnableFormatDetection = getEnvBool("OTSU_DEBUG_FORMAT", false)
	EnableImageDebug = getEnvBool("OTSU_DEBUG_IMAGE", true)
	EnablePerformanceDebug = getEnvBool("OTSU_DEBUG_PERFORMANCE", true)
	EnableMemoryDebug = getEnvBool("OTSU_DEBUG_MEMORY", true)
	EnableGUIDebug = getEnvBool("OTSU_DEBUG_GUI", false)
	EnableAlgorithmDebug = getEnvBool("OTSU_DEBUG_ALGORITHMS", false)
	EnableTriclassDebug = getEnvBool("OTSU_DEBUG_TRICLASS", false)
	EnablePixelAnalysisDebug = getEnvBool("OTSU_DEBUG_PIXELS", false)

	if EnableTriclassDebug {
		LogInfo("DebugInit", "Iterative Triclass debugging enabled")
	}

	if EnablePixelAnalysisDebug {
		LogInfo("DebugInit", "Pixel analysis debugging enabled")
	}

	if EnableFormatDetection {
		LogInfo("DebugInit", "Format detection debugging enabled")
	}

	if EnableImageDebug {
		LogInfo("DebugInit", "Image processing debugging enabled")
	}

	if EnablePerformanceDebug {
		LogInfo("DebugInit", "Performance debugging enabled")
	}

	if EnableMemoryDebug {
		LogInfo("DebugInit", "Memory debugging enabled")
	}

	if EnableGUIDebug {
		LogInfo("DebugInit", "GUI debugging enabled")
	}

	if EnableAlgorithmDebug {
		LogInfo("DebugInit", "Algorithm debugging enabled")
	}
}
