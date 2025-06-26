package app

import (
	"context"
	"os"
	"sync"

	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/gui/widgets"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"
	"otsu-obliterator/internal/shutdown"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/rs/zerolog"
)

const (
	AppName    = "Otsu Obliterator"
	AppID      = "com.imageprocessing.otsuobliterator"
	AppVersion = "1.0.0"
)

type Application struct {
	fyneApp       fyne.App
	window        fyne.Window
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	memoryManager *memory.Manager
	logger        logger.Logger
	shutdownMgr   *shutdown.Manager
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewApplication() (*Application, error) {
	fyneApp := app.NewWithID(AppID)
	window := fyneApp.NewWindow(AppName)

	windowSize := calculateMinimumWindowSize()
	window.Resize(windowSize)
	window.SetFixedSize(false)
	window.SetPadded(false)
	window.CenterOnScreen()
	window.SetMaster()

	// Initialize context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize logging with context
	logLevel := getLogLevel()
	log := logger.NewConsoleLogger(logLevel)

	log.Info("Application", "starting application", map[string]interface{}{
		"version":       AppVersion,
		"window_width":  windowSize.Width,
		"window_height": windowSize.Height,
		"log_level":     logLevel.String(),
	})

	// Initialize shutdown manager
	shutdownMgr := shutdown.NewManager(log)
	shutdownMgr.Listen()

	// Initialize memory manager with monitoring
	memoryManager := memory.NewManager(log)
	memoryManager.MonitorMemory()
	shutdownMgr.Register(memoryManager)

	// Initialize processing coordinator
	coordinator := pipeline.NewCoordinator(memoryManager, log)
	shutdownMgr.Register(coordinator)

	// Initialize GUI manager
	guiManager, err := gui.NewManager(window, log)
	if err != nil {
		cancel()
		return nil, err
	}
	shutdownMgr.Register(guiManager)

	// Connect the processing coordinator to GUI
	guiManager.SetProcessingCoordinator(coordinator)

	application := &Application{
		fyneApp:       fyneApp,
		window:        window,
		guiManager:    guiManager,
		coordinator:   coordinator,
		memoryManager: memoryManager,
		logger:        log,
		shutdownMgr:   shutdownMgr,
		ctx:           ctx,
		cancel:        cancel,
	}

	log.Info("Application", "initialization complete", nil)
	return application, nil
}

func calculateMinimumWindowSize() fyne.Size {
	imageDisplayWidth := widgets.ImageAreaWidth * 2
	toolbarHeight := float32(50)
	parametersHeight := float32(150)

	minimumWidth := float32(imageDisplayWidth + 100)
	minimumHeight := float32(widgets.ImageAreaHeight + toolbarHeight + parametersHeight + 100)

	return fyne.Size{
		Width:  minimumWidth,
		Height: minimumHeight,
	}
}

func (a *Application) Run() error {
	a.window.SetCloseIntercept(func() {
		a.logger.Info("Application", "shutdown requested via window close", nil)
		go a.shutdownMgr.Shutdown()
		a.window.Close()
	})

	// Show the GUI using the MVC pattern with thread-safe operation
	fyne.Do(func() {
		a.guiManager.Show()
		a.logger.Info("Application", "GUI displayed", nil)
	})

	// Run in a goroutine to handle shutdown signals
	go func() {
		<-a.shutdownMgr.Done()

		// Thread-safe app quit
		fyne.Do(func() {
			a.fyneApp.Quit()
		})
	}()

	a.fyneApp.Run()

	// Wait for all goroutines to complete
	a.wg.Wait()

	return nil
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.cancel()

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func getLogLevel() zerolog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		if os.Getenv("OTSU_DEBUG_ALL") == "true" {
			return zerolog.DebugLevel
		}
		return zerolog.InfoLevel
	}
}
