package main

// This is the main entry point that simply imports and uses the modularized app components. The actual application logic is split across:
// - app_core.go: Core application structure and initialization
// - app_handlers.go: Event handlers for user interactions
// - app_menus.go: Menu setup and handlers

// Debug component toggles
// make run-profile - Basic profiling (format debug OFF)
// make run-debug-format - Format detection only
// make run-debug-gui - GUI interactions only
// make run-debug-algorithms - Algorithm execution only
// make run-debug-all - Everything enabled
var (
	DebugFormatDetection = false
	DebugImageProcessing = true
	DebugMemoryTracking  = true
	DebugPerformance     = true
	DebugGUI             = false
	DebugAlgorithms      = false
)
