//go:build matprofile
// +build matprofile

package debug

import (
	"log"
	_ "net/http/pprof"
	"time"
)

var (
	profilingEnabled = true
)

func Initialize() {
	if profilingEnabled {
		log.Println("MatProfile debugging enabled")
		log.Println("Memory profiling available at http://localhost:6060/debug/pprof/")
		log.Println("Mat profiling available at http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat")
	}
}

func IsProfilingEnabled() bool {
	return profilingEnabled
}

func LogInfo(component string, message string) {
	log.Printf("[INFO] %s: %s", component, message)
}

func LogError(component string, message string) {
	log.Printf("[ERROR] %s: %s", component, message)
}

func LogWarning(component string, message string) {
	log.Printf("[WARN] %s: %s", component, message)
}

func LogPerformance(operation string, duration time.Duration) {
	if profilingEnabled {
		log.Printf("[PERF] %s: %v", operation, duration)
	}
}

func LogMemory(component string, message string) {
	if profilingEnabled {
		log.Printf("[MEMORY] %s: %s", component, message)
	}
}

func Cleanup() {
	if profilingEnabled {
		log.Println("Debug profiling cleanup completed")
	}
}
