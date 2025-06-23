package main

import (
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"otsu-obliterator/pipeline"
)

func (app *OtsuApp) handleImageLoad() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			app.showError("File Load Error", err)
			return
		}
		if reader == nil {
			return
		}

		// Use fyne.Do for thread safety in v2.6+
		fyne.Do(func() {
			app.mainGUI.UpdateStatus("Loading image...")
		})

		go func() {
			// Read all data before closing
			data, readErr := io.ReadAll(reader)
			originalURI := reader.URI() // Capture URI before closing
			reader.Close()              // Close immediately after reading

			if readErr != nil {
				fyne.Do(func() {
					app.showError("Image Read Error", readErr)
					app.mainGUI.UpdateStatus("Ready")
				})
				return
			}

			// Create a new reader from the data with original URI
			dataReader := &DataReader{data: data, uri: originalURI}
			err := app.pipeline.LoadImage(dataReader)

			fyne.Do(func() {
				if err != nil {
					app.showError("Image Load Error", err)
					app.mainGUI.UpdateStatus("Ready")
					return
				}

				// Update GUI with loaded image using thread-safe calls
				originalImg := app.pipeline.GetOriginalImage()
				if originalImg != nil {
					app.mainGUI.SetOriginalImage(originalImg)
					app.mainGUI.UpdateStatus("Image loaded successfully")
				}
			})
		}()
	}, app.window)
}

func (app *OtsuApp) handleImageSave() {
	processedImg := app.pipeline.GetProcessedImage()
	if processedImg == nil {
		app.showError("Save Error", fmt.Errorf("no processed image to save"))
		return
	}

	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			app.showError("File Save Error", err)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		// Use fyne.Do for thread safety in v2.6+
		fyne.Do(func() {
			app.mainGUI.UpdateStatus("Saving image...")
		})

		go func() {
			err := app.pipeline.SaveImage(writer)
			fyne.Do(func() {
				if err != nil {
					app.showError("Image Save Error", err)
				} else {
					app.mainGUI.UpdateStatus("Image saved successfully")
				}
			})
		}()
	}, app.window)
}

func (app *OtsuApp) handleAlgorithmChange(algorithm string) {
	// Use fyne.Do for thread safety when updating GUI components
	fyne.Do(func() {
		app.otsuManager.SetCurrentAlgorithm(algorithm)
		params := app.otsuManager.GetParameters(algorithm)
		app.mainGUI.UpdateParameterPanel(algorithm, params)
		app.mainGUI.UpdateStatus(fmt.Sprintf("Switched to %s algorithm", algorithm))
	})
}

func (app *OtsuApp) handleParameterChange(name string, value interface{}) {
	currentAlgorithm := app.otsuManager.GetCurrentAlgorithm()

	// Validate parameter before setting
	tempParams := app.otsuManager.GetAllParameters(currentAlgorithm)
	tempParams[name] = value

	err := app.otsuManager.ValidateParameters(currentAlgorithm, tempParams)
	if err != nil {
		fyne.Do(func() {
			app.showError("Parameter Error", err)
		})
		return
	}

	// Use fyne.Do for thread safety when updating parameters
	fyne.Do(func() {
		app.otsuManager.SetParameter(currentAlgorithm, name, value)
		app.mainGUI.UpdateStatus(fmt.Sprintf("Updated %s parameter", name))
	})
}

func (app *OtsuApp) handleGeneratePreview() {
	originalImg := app.pipeline.GetOriginalImage()
	if originalImg == nil {
		app.showError("Processing Error", fmt.Errorf("no image loaded"))
		return
	}

	// Use fyne.Do for thread safety when updating GUI
	fyne.Do(func() {
		app.mainGUI.UpdateStatus("Generating preview...")
		app.mainGUI.UpdateProgress(0.0)
	})

	go func() {
		currentAlgorithm := app.otsuManager.GetCurrentAlgorithm()
		params := app.otsuManager.GetAllParameters(currentAlgorithm)

		var processedImg *pipeline.ImageData
		var err error

		// Process using the modularized algorithms
		switch currentAlgorithm {
		case "2D Otsu":
			processedImg, err = app.pipeline.Process2DOtsu(params)
		case "Iterative Triclass":
			processedImg, err = app.pipeline.ProcessIterativeTriclass(params)
		default:
			err = fmt.Errorf("unknown algorithm: %s", currentAlgorithm)
		}

		// All GUI updates must be wrapped in fyne.Do for v2.6+ thread safety
		fyne.Do(func() {
			app.mainGUI.UpdateProgress(1.0)
			if err != nil {
				app.showError("Processing Error", err)
				app.mainGUI.UpdateStatus("Processing failed")
				return
			}

			if processedImg != nil {
				app.mainGUI.SetPreviewImage(processedImg)

				// Calculate and display metrics using thread-safe operations
				originalData := app.pipeline.GetOriginalImage()
				if originalData != nil {
					// Run metrics calculation in background but update GUI in fyne.Do
					go func() {
						psnr := app.pipeline.CalculatePSNR(originalData, processedImg)
						ssim := app.pipeline.CalculateSSIM(originalData, processedImg)

						fyne.Do(func() {
							app.mainGUI.UpdateMetrics(psnr, ssim)
						})
					}()
				}

				app.mainGUI.UpdateStatus("Preview generated successfully")
			}
		})
	}()
}

func (app *OtsuApp) showError(title string, err error) {
	// Log error using debug manager
	app.debugManager.LogError("UI", err)

	// Use fyne.Do to ensure dialog is shown on main thread for v2.6+
	fyne.Do(func() {
		dialog.ShowError(err, app.window)
	})
}
