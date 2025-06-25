package app

import (
	"os"

	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	AppName         = "Otsu Obliterator"
	AppID           = "com.imageprocessing.otsuobliterator"
	AppVersion      = "1.0.0"
	WindowWidth     = 1400
	WindowHeight    = 900
	MinWindowWidth  = 800
	MinWindowHeight = 600
)

type Application struct {
	fyneApp       fyne.App
	window        fyne.Window
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	memoryManager *memory.Manager
	debugCoord    debug.Coordinator
	lifecycle     *Lifecycle
}

func NewApplication() (*Application, error) {
	fyneApp := app.NewWithID(AppID)
	window := fyneApp.NewWindow(AppName)

	// Configure window for resizing - order matters in Fyne v2
	window.Resize(fyne.NewSize(WindowWidth, WindowHeight))
	window.SetFixedSize(false)
	window.SetPadded(false) // Remove edge padding for edge-to-edge layout
	window.CenterOnScreen()
	window.SetMaster()

	// Initialize debug system based on environment
	debugConfig := getDebugConfig()
	debugCoord := debug.NewCoordinator(debugConfig)

	logger := debugCoord.Logger()
	memTracker := debugCoord.MemoryTracker()

	logger.Info("Application", "starting application", map[string]interface{}{
		"version":       AppVersion,
		"debug_enabled": debugConfig.EnableLogging,
	})

	memoryManager := memory.NewManager(logger, memTracker)
	coordinator := pipeline.NewCoordinator(memoryManager, debugCoord)

	guiManager, err := gui.NewManager(window, debugCoord)
	if err != nil {
		return nil, err
	}

	lifecycle := NewLifecycle(memoryManager, debugCoord, guiManager)

	application := &Application{
		fyneApp:       fyneApp,
		window:        window,
		guiManager:    guiManager,
		coordinator:   coordinator,
		memoryManager: memoryManager,
		debugCoord:    debugCoord,
		lifecycle:     lifecycle,
	}

	if err := application.setupHandlers(); err != nil {
		return nil, err
	}

	logger.Info("Application", "initialization complete", nil)
	return application, nil
}

func (a *Application) setupHandlers() error {
	handlers := NewHandlers(a.coordinator, a.guiManager, a.debugCoord)

	a.guiManager.SetImageLoadHandler(handlers.HandleImageLoad)
	a.guiManager.SetImageSaveHandler(handlers.HandleImageSave)
	a.guiManager.SetAlgorithmChangeHandler(handlers.HandleAlgorithmChange)
	a.guiManager.SetParameterChangeHandler(handlers.HandleParameterChange)
	a.guiManager.SetGeneratePreviewHandler(handlers.HandleGeneratePreview)

	return nil
}

func (a *Application) Run() error {
	logger := a.debugCoord.Logger()

	a.window.SetCloseIntercept(func() {
		logger.Info("Application", "shutdown requested", nil)
		a.lifecycle.Shutdown()
		a.window.Close()
	})

	a.window.SetContent(a.guiManager.GetMainContainer())
	a.window.Show()

	logger.Info("Application", "GUI displayed", nil)
	a.fyneApp.Run()

	return nil
}

func getDebugConfig() debug.Config {
	// Check environment variables for debug configuration
	if os.Getenv("OTSU_DEBUG_ALL") == "true" {
		return debug.DefaultConfig()
	}

	if os.Getenv("OTSU_PRODUCTION") == "true" {
		return debug.ProductionConfig()
	}

	// Default development configuration
	config := debug.DefaultConfig()

	// Override specific settings based on environment
	if os.Getenv("OTSU_DEBUG_MEMORY") == "true" {
		config.EnableMemoryTracking = true
		config.EnableStackTraces = true
	}

	if os.Getenv("OTSU_DEBUG_FILES") == "true" {
		config.EnableFileTracking = true
	}

	if os.Getenv("OTSU_JSON_LOGS") == "true" {
		config.UseJSONLogging = true
	}

	return config
}
