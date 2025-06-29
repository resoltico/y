package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	AppName    = "Otsu Obliterator"
	AppID      = "com.imageprocessing.otsu-obliterator"
	AppVersion = "1.0.0"
)

type Application struct {
	fyneApp       fyne.App
	window        fyne.Window
	controller    *gui.Controller
	coordinator   *pipeline.Coordinator
	memoryManager *memory.Manager
	logger        logger.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	shutdown      chan struct{}
}

func NewApplication(ctx context.Context) (*Application, error) {
	app.SetMetadata(fyne.AppMetadata{
		ID:      AppID,
		Name:    AppName,
		Version: AppVersion,
	})

	fyneApp := app.NewWithID(AppID)
	window := fyneApp.NewWindow(AppName)

	// Calculate responsive window size
	windowSize := calculateWindowSize()
	window.Resize(windowSize)
	window.CenterOnScreen()

	appCtx, cancel := context.WithCancel(ctx)

	log := logger.NewStructuredLogger(determineLogLevel())

	log.Info("Application starting", map[string]interface{}{
		"version":     AppVersion,
		"window_size": fmt.Sprintf("%.0fx%.0f", windowSize.Width, windowSize.Height),
		"go_version":  runtime.Version(),
		"num_cpu":     runtime.NumCPU(),
	})

	memManager := memory.NewManager(log)
	coordinator := pipeline.NewCoordinator(memManager, log)
	controller := gui.NewController(coordinator, log)

	application := &Application{
		fyneApp:       fyneApp,
		window:        window,
		controller:    controller,
		coordinator:   coordinator,
		memoryManager: memManager,
		logger:        log,
		ctx:           appCtx,
		cancel:        cancel,
		shutdown:      make(chan struct{}),
	}

	application.setupSignalHandling()
	application.setupWindow()

	log.Info("Application initialized", nil)
	return application, nil
}

func (a *Application) setupWindow() {
	a.window.SetFixedSize(false)
	a.window.SetPadded(true)

	// Setup content using Fyne v2.6 patterns
	content := a.controller.CreateMainContent()
	a.window.SetContent(content)

	a.window.SetCloseIntercept(func() {
		a.logger.Info("Shutdown requested via window close", nil)
		a.initiateShutdown()
		a.window.Close()
	})
}

func (a *Application) SetupMenus() {
	aboutAction := func() {
		a.logger.Info("About dialog requested", nil)
		// Use fyne.Do for UI thread safety in Fyne v2.6+
		fyne.Do(func() {
			a.showAboutDialog()
		})
	}

	fileMenu := fyne.NewMenu("File")
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", aboutAction),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	a.window.SetMainMenu(mainMenu)

	a.logger.Info("Menu system configured", map[string]interface{}{
		"menus": []string{"File", "Help"},
	})
}

func (a *Application) showAboutDialog() {
	metadata := a.fyneApp.Metadata()

	aboutContent := container.NewVBox(
		widget.NewLabel(metadata.Name),
		widget.NewLabel(fmt.Sprintf("Version: %s", metadata.Version)),
		widget.NewLabel(""),
		widget.NewLabel("Modern Go image processing with 2D Otsu thresholding"),
		widget.NewLabel(""),
		widget.NewLabel("Runtime Information:"),
		widget.NewLabel(fmt.Sprintf("Go: %s", runtime.Version())),
		widget.NewLabel(fmt.Sprintf("Platform: %s/%s", runtime.GOOS, runtime.GOARCH)),
		widget.NewLabel(fmt.Sprintf("CPUs: %d", runtime.NumCPU())),
		widget.NewLabel("OpenCV: 4.11.0+"),
		widget.NewLabel("Fyne: v2.6.1"),
	)

	dialog.ShowCustom("About", "Close", aboutContent, a.window)
}

func (a *Application) Run(ctx context.Context) error {
	a.logger.Info("Starting UI display", nil)

	// Use fyne.Do for thread-safe UI operations
	fyne.Do(func() {
		a.window.Show()
	})

	go func() {
		select {
		case <-a.shutdown:
			a.logger.Info("Shutdown signal received", nil)
			fyne.Do(func() {
				a.fyneApp.Quit()
			})
		case <-ctx.Done():
			a.logger.Info("Context cancelled", nil)
			a.initiateShutdown()
		}
	}()

	a.fyneApp.Run()
	a.wg.Wait()
	return nil
}

func (a *Application) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case sig := <-sigChan:
			a.logger.Info("System signal received", map[string]interface{}{
				"signal": sig.String(),
			})
			a.initiateShutdown()
		case <-a.ctx.Done():
			return
		}
	}()
}

func (a *Application) initiateShutdown() {
	select {
	case <-a.shutdown:
		return
	default:
		close(a.shutdown)
	}

	a.logger.Info("Shutdown sequence starting", nil)
	a.cancel()

	// Shutdown components with timeout
	shutdownTimeout := 15 * time.Second
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	shutdownComponents := []func(){
		a.controller.Shutdown,
		a.coordinator.Shutdown,
		a.memoryManager.Shutdown,
	}

	for i, shutdownFunc := range shutdownComponents {
		done := make(chan struct{})
		go func(fn func()) {
			defer close(done)
			fn()
		}(shutdownFunc)

		select {
		case <-done:
		case <-shutdownCtx.Done():
			a.logger.Warning("Component shutdown timeout", map[string]interface{}{
				"component_index": i,
			})
		}
	}

	a.logger.Info("Shutdown sequence completed", nil)
}

func calculateWindowSize() fyne.Size {
	// Modern responsive sizing for high-DPI displays
	baseWidth := float32(1200)
	baseHeight := float32(800)

	// Adjust for system capabilities
	if runtime.NumCPU() >= 8 {
		baseWidth *= 1.2
		baseHeight *= 1.2
	}

	return fyne.Size{
		Width:  baseWidth,
		Height: baseHeight,
	}
}

func determineLogLevel() logger.LogLevel {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		return logger.DebugLevel
	case "info":
		return logger.InfoLevel
	case "warn":
		return logger.WarnLevel
	case "error":
		return logger.ErrorLevel
	default:
		if os.Getenv("DEBUG") == "1" {
			return logger.DebugLevel
		}
		return logger.InfoLevel
	}
}
