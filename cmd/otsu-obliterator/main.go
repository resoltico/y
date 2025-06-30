package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"otsu-obliterator/internal/controllers"
	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/models"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/services"
	"otsu-obliterator/internal/views"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

const (
	AppName    = "Otsu Obliterator"
	AppID      = "com.imageprocessing.otsu-obliterator"
	AppVersion = "1.0.0"
)

// Application represents the main application using modern MVC architecture
type Application struct {
	// Core components
	fyneApp fyne.App
	window  fyne.Window
	logger  logger.Logger

	// MVC Components
	controller *controllers.MainController
	view       *views.MainView

	// Services
	imageService      *services.ImageService
	processingService *services.ProcessingService

	// Models/Repositories
	imageRepo     *models.ImageRepository
	configRepo    *models.ProcessingConfiguration
	stateRepo     *models.ProcessingStateRepository
	memoryManager *memory.Manager

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
}

func main() {
	// Configure Go 1.24 runtime for image processing workloads
	configureRuntime()

	// Create application context with graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application
	application, err := NewApplication(ctx)
	if err != nil {
		log.Fatalf("Application initialization failed: %v", err)
	}

	// Setup graceful shutdown
	setupGracefulShutdown(application, cancel)

	// Run application
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application execution failed: %v", err)
	}

	log.Println("Application terminated successfully")
}

// configureRuntime optimizes Go runtime for image processing with Go 1.24 features
func configureRuntime() {
	// Set GOMAXPROCS to utilize all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Configure GC for image processing workloads (higher threshold for large allocations)
	runtime.SetGCPercent(200)

	// Set memory limit if available (Go 1.19+)
	if os.Getenv("GOMEMLIMIT") == "" {
		// Set default memory limit to 4GB for image processing
		os.Setenv("GOMEMLIMIT", "4GiB")
	}

	log.Printf("Runtime configured: GOMAXPROCS=%d, GC target=200%%", runtime.NumCPU())
}

// NewApplication creates and initializes the application using dependency injection
func NewApplication(ctx context.Context) (*Application, error) {
	// Create Fyne application with modern metadata
	fyneApp := app.NewWithID(AppID)
	fyneApp.SetMetadata(&fyne.AppMetadata{
		ID:      AppID,
		Name:    AppName,
		Version: AppVersion,
		Icon:    nil, // Load from resources if available
	})

	// Create main window with responsive sizing
	window := fyneApp.NewWindow(AppName)
	windowSize := calculateResponsiveWindowSize()
	window.Resize(windowSize)
	window.CenterOnScreen()

	// Create application context
	appCtx, appCancel := context.WithCancel(ctx)

	// Initialize logger
	logLevel := determineLogLevel()
	appLogger := logger.NewStructuredLogger(logLevel)

	appLogger.Info("Application starting", map[string]interface{}{
		"version":     AppVersion,
		"window_size": fmt.Sprintf("%.0fx%.0f", windowSize.Width, windowSize.Height),
		"go_version":  runtime.Version(),
		"num_cpu":     runtime.NumCPU(),
		"log_level":   logLevel,
	})

	// Initialize repositories/models
	imageRepo := models.NewImageRepository()
	configRepo := models.NewProcessingConfiguration()
	stateRepo := models.NewProcessingStateRepository()
	memManager := memory.NewManager(appLogger)

	// Initialize services
	imageService := services.NewImageService(memManager, imageRepo)
	processingService := services.NewProcessingService(memManager, imageRepo, configRepo, stateRepo)

	// Initialize MVC components
	mainController := controllers.NewMainController(
		imageService, processingService,
		imageRepo, configRepo, stateRepo,
	)
	mainView := views.NewMainView(window)

	// Wire MVC components together
	mainController.SetMainView(mainView)
	mainController.SetWindow(window)

	// Create application instance
	application := &Application{
		fyneApp:           fyneApp,
		window:            window,
		logger:            appLogger,
		controller:        mainController,
		view:              mainView,
		imageService:      imageService,
		processingService: processingService,
		imageRepo:         imageRepo,
		configRepo:        configRepo,
		stateRepo:         stateRepo,
		memoryManager:     memManager,
		ctx:               appCtx,
		cancel:            appCancel,
	}

	// Setup window lifecycle events
	application.setupWindowEvents()

	appLogger.Info("Application initialized successfully", map[string]interface{}{
		"components":     []string{"models", "services", "controllers", "views"},
		"mvc_pattern":    true,
		"memory_manager": true,
		"fyne_version":   "v2.6.1",
		"gocv_version":   "v0.41.0",
	})

	return application, nil
}

// Run starts the application with modern event loop management
func (app *Application) Run(ctx context.Context) error {
	app.logger.Info("Starting application UI", nil)

	// Show main window using thread-safe Fyne v2.6+ patterns
	fyne.Do(func() {
		app.view.Show()
	})

	// Setup context cancellation monitoring
	go func() {
		select {
		case <-ctx.Done():
			app.logger.Info("Context cancelled, initiating shutdown", nil)
			app.initiateShutdown()
		case <-app.ctx.Done():
			app.logger.Info("Application context cancelled", nil)
		}
	}()

	// Start performance monitoring
	go app.startPerformanceMonitoring()

	// Run Fyne application (blocking)
	app.fyneApp.Run()

	return nil
}

// setupWindowEvents configures window lifecycle events with Fyne v2.6+ patterns
func (app *Application) setupWindowEvents() {
	// Window close intercept for graceful shutdown
	app.window.SetCloseIntercept(func() {
		app.logger.Info("Window close requested", nil)

		// Use fyne.Do for thread-safe operations
		fyne.Do(func() {
			app.view.ShowConfirm(
				"Exit Application",
				"Are you sure you want to exit?",
				func(confirmed bool) {
					if confirmed {
						app.initiateShutdown()
						app.window.Close()
					}
				},
			)
		})
	})

	// Window resize handler for responsive layout
	app.window.SetOnClosed(func() {
		app.logger.Info("Window closed, performing cleanup", nil)
		app.performCleanup()
	})
}

// setupGracefulShutdown configures signal handling for graceful shutdown
func setupGracefulShutdown(application *Application, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, initiating graceful shutdown", sig)

		application.logger.Info("System signal received", map[string]interface{}{
			"signal": sig.String(),
		})

		cancel()
		application.initiateShutdown()
	}()
}

// startPerformanceMonitoring monitors application performance with Go 1.24 features
func (app *Application) startPerformanceMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			app.logPerformanceMetrics()
		case <-app.ctx.Done():
			return
		}
	}
}

// logPerformanceMetrics logs current performance statistics
func (app *Application) logPerformanceMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	allocCount, deallocCount, usedMemory := app.memoryManager.GetStats()
	processingStats := app.processingService.GetProcessingStats()
	imageStats := app.imageRepo.GetImageStats()

	app.logger.Debug("Performance metrics", map[string]interface{}{
		"go_memory_mb":        memStats.Alloc / 1024 / 1024,
		"go_total_alloc_mb":   memStats.TotalAlloc / 1024 / 1024,
		"go_gc_runs":          memStats.NumGC,
		"opencv_allocs":       allocCount,
		"opencv_deallocs":     deallocCount,
		"opencv_memory_mb":    usedMemory / 1024 / 1024,
		"images_processed":    processingStats.TotalProcessed,
		"avg_process_time_ms": processingStats.AverageTime.Milliseconds(),
		"images_in_memory":    imageStats.ProcessedCount,
		"worker_count":        app.processingService.GetWorkerCount(),
		"goroutine_count":     runtime.NumGoroutine(),
	})

	// Update UI with memory information (thread-safe)
	fyne.Do(func() {
		app.view.SetMemoryInfo(usedMemory, memStats.Alloc)
	})
}

// initiateShutdown begins the graceful shutdown process
func (app *Application) initiateShutdown() {
	app.logger.Info("Shutdown sequence initiated", nil)

	// Cancel application context
	app.cancel()

	// Perform cleanup in background to avoid blocking UI
	go func() {
		shutdownTimeout := 15 * time.Second
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()

		app.performShutdownSequence(shutdownCtx)
	}()
}

// performShutdownSequence executes the shutdown sequence with timeout
func (app *Application) performShutdownSequence(ctx context.Context) {
	shutdownSteps := []struct {
		name string
		fn   func()
	}{
		{"controller", app.controller.Shutdown},
		{"processing service", app.processingService.Shutdown},
		{"image service", app.imageService.Cleanup},
		{"memory manager", app.memoryManager.Shutdown},
	}

	for _, step := range shutdownSteps {
		select {
		case <-ctx.Done():
			app.logger.Warning("Shutdown timeout exceeded", map[string]interface{}{
				"step": step.name,
			})
			return
		default:
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			step.fn()
		}()

		select {
		case <-done:
			app.logger.Debug("Shutdown step completed", map[string]interface{}{
				"step": step.name,
			})
		case <-ctx.Done():
			app.logger.Warning("Shutdown step timeout", map[string]interface{}{
				"step": step.name,
			})
		}
	}

	app.performCleanup()
}

// performCleanup performs final cleanup operations
func (app *Application) performCleanup() {
	// Final garbage collection with Go 1.24 optimizations
	runtime.GC()
	runtime.GC() // Double collection for image processing cleanup

	app.logger.Info("Application cleanup completed", nil)
}

// calculateResponsiveWindowSize determines appropriate window size
func calculateResponsiveWindowSize() fyne.Size {
	baseWidth := float32(1200)
	baseHeight := float32(800)

	// Adjust for high-performance systems
	if runtime.NumCPU() >= 8 {
		baseWidth *= 1.2
		baseHeight *= 1.2
	}

	// Ensure minimum viable size
	if baseWidth < 800 {
		baseWidth = 800
	}
	if baseHeight < 600 {
		baseHeight = 600
	}

	return fyne.NewSize(baseWidth, baseHeight)
}

// determineLogLevel determines appropriate log level from environment
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
