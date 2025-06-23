package main

import (
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"otsu-obliterator/debug"
	"otsu-obliterator/gui"
	"otsu-obliterator/otsu"
	"otsu-obliterator/pipeline"
)

// DataReader implements fyne.URIReadCloser for in-memory data
type DataReader struct {
	data []byte
	pos  int
	uri  fyne.URI
}

func (dr *DataReader) Read(p []byte) (n int, err error) {
	if dr.pos >= len(dr.data) {
		return 0, io.EOF
	}
	n = copy(p, dr.data[dr.pos:])
	dr.pos += n
	return n, nil
}

func (dr *DataReader) Close() error {
	return nil
}

func (dr *DataReader) URI() fyne.URI {
	return dr.uri
}

const (
	AppName      = "Otsu Obliterator"
	AppID        = "com.imageprocessing.otsuobliterator"
	AppVersion   = "1.0.0"
	WindowWidth  = 1200
	WindowHeight = 800
)

// Debug component toggles
var (
	DebugFormatDetection = false // Format detection and signature analysis
	DebugImageProcessing = true  // Image loading, processing, and metrics
	DebugMemoryTracking  = true  // Memory usage and Mat profiling
	DebugPerformance     = true  // Timing and performance metrics
	DebugGUI             = false // GUI events and interactions
	DebugAlgorithms      = false // Algorithm parameter changes and execution
)

type OtsuApp struct {
	fyneApp      fyne.App
	window       fyne.Window
	mainGUI      *gui.MainInterface
	pipeline     *pipeline.ImagePipeline
	otsuManager  *otsu.AlgorithmManager
	debugManager *debug.Manager
}

func NewOtsuApp() *OtsuApp {
	fyneApp := app.NewWithID(AppID)

	window := fyneApp.NewWindow(AppName)
	window.Resize(fyne.NewSize(WindowWidth, WindowHeight))
	window.SetFixedSize(true)

	// Initialize profiling if enabled
	debug.Initialize()

	// Set debug component toggles
	debug.EnableFormatDebug = DebugFormatDetection
	debug.EnableImageDebug = DebugImageProcessing
	debug.EnablePerformanceDebug = DebugPerformance
	debug.EnableMemoryDebug = DebugMemoryTracking

	// Initialize managers
	debugManager := debug.NewManager()
	otsuManager := otsu.NewAlgorithmManager()
	pipelineManager := pipeline.NewImagePipeline(debugManager)

	app := &OtsuApp{
		fyneApp:      fyneApp,
		window:       window,
		debugManager: debugManager,
		otsuManager:  otsuManager,
		pipeline:     pipelineManager,
	}

	// Initialize GUI
	mainGUI := gui.NewMainInterface(window, app.handleImageLoad, app.handleImageSave,
		app.handleAlgorithmChange, app.handleParameterChange, app.handleGeneratePreview)
	app.mainGUI = mainGUI

	// Connect pipeline to GUI updates
	app.pipeline.SetProgressCallback(app.mainGUI.UpdateProgress)
	app.pipeline.SetStatusCallback(app.mainGUI.UpdateStatus)

	// Initialize GUI components after everything is set up
	app.mainGUI.Initialize()

	return app
}

func (app *OtsuApp) Run() {
	app.setupMenus()

	// Set main content
	content := app.mainGUI.GetMainContainer()
	app.window.SetContent(content)

	// Handle window close
	app.window.SetCloseIntercept(func() {
		app.cleanup()
		app.window.Close()
	})

	app.window.ShowAndRun()
}

func (app *OtsuApp) cleanup() {
	if app.pipeline != nil {
		app.pipeline.Cleanup()
	}

	if app.debugManager != nil {
		app.debugManager.Cleanup()
	}

	debug.Cleanup()
}

func main() {
	// Start profiling server if enabled
	if debug.IsProfilingEnabled() {
		go func() {
			log.Println("Starting profiling server on :6060")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// Create and run app
	app := NewOtsuApp()
	app.Run()
}
