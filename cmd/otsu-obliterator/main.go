package main

import (
	"context"
	"log"
	"runtime"
	"time"

	"otsu-obliterator/internal/app"
)

func main() {
	log.Println("Starting Otsu Obliterator v1.0.0")

	// Configure runtime for Go 1.24 performance
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Set GC target for image processing workloads
	runtime.SetGCPercent(200)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	application, err := app.NewApplication(ctx)
	if err != nil {
		log.Fatalf("Application creation failed: %v", err)
	}

	log.Println("Initializing menu system")
	application.SetupMenus()
	log.Println("Menu system ready")

	log.Println("Starting application")
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Application execution failed: %v", err)
	}
	log.Println("Application terminated successfully")
}
