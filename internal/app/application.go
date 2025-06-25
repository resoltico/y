package app

import (
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	AppName      = "Otsu Obliterator"
	AppID        = "com.imageprocessing.otsuobliterator"
	AppVersion   = "1.0.0"
	WindowWidth  = 1400
	WindowHeight = 900
)

type Application struct {
	fyneApp       fyne.App
	window        fyne.Window
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	memoryManager *memory.Manager
	debugManager  *debug.Manager
	lifecycle     *Lifecycle
}

func NewApplication() (*Application, error) {
	fyneApp := app.NewWithID(AppID)
	window := fyneApp.NewWindow(AppName)
	window.Resize(fyne.NewSize(WindowWidth, WindowHeight))
	window.SetMaster() // Ensures app exits when main window closes

	debugManager := debug.NewManager()
	memoryManager := memory.NewManager(debugManager)
	coordinator := pipeline.NewCoordinator(memoryManager, debugManager)

	guiManager, err := gui.NewManager(window, debugManager)
	if err != nil {
		return nil, err
	}

	lifecycle := NewLifecycle(memoryManager, debugManager, guiManager)

	application := &Application{
		fyneApp:       fyneApp,
		window:        window,
		guiManager:    guiManager,
		coordinator:   coordinator,
		memoryManager: memoryManager,
		debugManager:  debugManager,
		lifecycle:     lifecycle,
	}

	if err := application.setupHandlers(); err != nil {
		return nil, err
	}

	return application, nil
}

func (a *Application) setupHandlers() error {
	handlers := NewHandlers(a.coordinator, a.guiManager, a.debugManager)

	a.guiManager.SetImageLoadHandler(handlers.HandleImageLoad)
	a.guiManager.SetImageSaveHandler(handlers.HandleImageSave)
	a.guiManager.SetAlgorithmChangeHandler(handlers.HandleAlgorithmChange)
	a.guiManager.SetParameterChangeHandler(handlers.HandleParameterChange)
	a.guiManager.SetGeneratePreviewHandler(handlers.HandleGeneratePreview)

	return nil
}

func (a *Application) Run() error {
	a.window.SetCloseIntercept(func() {
		a.lifecycle.Shutdown()
		a.window.Close()
	})

	a.window.SetContent(a.guiManager.GetMainContainer())
	a.window.Show()
	a.fyneApp.Run()
	return nil
}
