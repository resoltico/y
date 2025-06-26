package main

import (
	"log"
	"otsu-obliterator/internal/app"
)

func main() {
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	if err := application.Run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}
