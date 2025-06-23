// +build !matprofile

package debug

import (
	"time"
)

var (
	profilingEnabled = false
)

func Initialize() {
	// No profiling in release builds
}

func IsProfilingEnabled() bool {
	return false
}

func LogInfo(component string, message string) {
	// Silent in release builds
}

func LogError(component string, message string) {
	// Silent in release builds  
}

func LogWarning(component string, message string) {
	// Silent in release builds
}

func LogPerformance(operation string, duration time.Duration) {
	// Silent in release builds
}

func LogMemory(component string, message string) {
	// Silent in release builds
}

func Cleanup() {
	// Nothing to cleanup in release builds
}