package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/gui/widgets"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/rs/zerolog"
)

const (
	AppName    = "Otsu Obliterator"
	AppID      = "com.imageprocessing.otsuobliterator"
	AppVersion = "1.0.0"
)

type shutdownHandler interface {
	Shutdown()
}

type Application struct {
	fyneApp       fyne.App
	window        fyne.Window
	guiManager    *gui.Manager
	coordinator   pipeline.ProcessingCoordinator
	memoryManager *memory.Manager
	logger        logger.Logger
	shutdownables []shutdownHandler
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	shutdown      chan struct{}
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

	ctx, cancel := context.WithCancel(context.Background())
	logLevel := getLogLevel()
	log := logger.NewConsoleLogger(logLevel)

	log.Info("Application", "starting application", map[string]interface{}{
		"version":       AppVersion,
		"window_width":  windowSize.Width,
		"window_height": windowSize.Height,
		"log_level":     logLevel.String(),
	})

	memoryManager := memory.NewManager(log)
	coordinator := pipeline.NewCoordinator(memoryManager, log)

	guiManager, err := gui.NewManager(window, log)
	if err != nil {
		cancel()
		return nil, err
	}

	guiManager.SetProcessingCoordinator(coordinator)

	application := &Application{
		fyneApp:       fyneApp,
		window:        window,
		guiManager:    guiManager,
		coordinator:   coordinator,
		memoryManager: memoryManager,
		logger:        log,
		ctx:           ctx,
		cancel:        cancel,
		shutdown:      make(chan struct{}),
		shutdownables: []shutdownHandler{
			memoryManager,
			coordinator,
			guiManager,
		},
	}

	application.setupSignalHandling()
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

func (a *Application) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case sig := <-sigChan:
			a.logger.Info("Application", "shutdown signal received", map[string]interface{}{
				"signal": sig.String(),
			})
			a.initiateShutdown()
		case <-a.ctx.Done():
			return
		}
	}()
}

func (a *Application) Run() error {
	a.window.SetCloseIntercept(func() {
		a.logger.Info("Application", "shutdown requested via window close", nil)
		a.initiateShutdown()
		a.window.Close()
	})

	fyne.Do(func() {
		a.guiManager.Show()
		a.logger.Info("Application", "GUI displayed", nil)
	})

	go func() {
		<-a.shutdown
		fyne.Do(func() {
			a.fyneApp.Quit()
		})
	}()

	a.fyneApp.Run()
	a.wg.Wait()
	return nil
}

func (a *Application) initiateShutdown() {
	select {
	case <-a.shutdown:
		return // Already shutting down
	default:
		close(a.shutdown)
	}

	a.logger.Info("Application", "shutdown sequence initiated", map[string]interface{}{
		"components": len(a.shutdownables),
	})

	a.cancel()

	// Shutdown components in reverse order with timeout
	for i := len(a.shutdownables) - 1; i >= 0; i-- {
		component := a.shutdownables[i]

		done := make(chan struct{})
		go func() {
			defer close(done)
			component.Shutdown()
		}()

		select {
		case <-done:
			// Component shut down successfully
		case <-time.After(10 * time.Second):
			a.logger.Warning("Application", "component shutdown timeout", map[string]interface{}{
				"component_index": i,
			})
		}
	}

	a.logger.Info("Application", "shutdown sequence completed", nil)
}

func (a *Application) Shutdown(ctx context.Context) error {
	a.initiateShutdown()

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
