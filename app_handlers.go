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

				// Update GUI with loaded image
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
	app.otsuManager.SetCurrentAlgorithm(algorithm)
	params := app.otsuManager.GetParameters(algorithm)
	app.mainGUI.UpdateParameterPanel(algorithm, params)
}

func (app *OtsuApp) handleParameterChange(name string, value interface{}) {
	currentAlgorithm := app.otsuManager.GetCurrentAlgorithm()
	app.otsuManager.SetParameter(currentAlgorithm, name, value)
}

func (app *OtsuApp) handleGeneratePreview() {
	originalImg := app.pipeline.GetOriginalImage()
	if originalImg == nil {
		app.showError("Processing Error", fmt.Errorf("no image loaded"))
		return
	}

	fyne.Do(func() {
		app.mainGUI.UpdateStatus("Generating preview...")
		app.mainGUI.UpdateProgress(0.0)
	})

	go func() {
		currentAlgorithm := app.otsuManager.GetCurrentAlgorithm()
		params := app.otsuManager.GetAllParameters(currentAlgorithm)

		var processedImg *pipeline.ImageData
		var err error

		switch currentAlgorithm {
		case "2D Otsu":
			processedImg, err = app.pipeline.Process2DOtsu(params)
		case "Iterative Triclass":
			processedImg, err = app.pipeline.ProcessIterativeTriclass(params)
		default:
			err = fmt.Errorf("unknown algorithm: %s", currentAlgorithm)
		}

		fyne.Do(func() {
			app.mainGUI.UpdateProgress(1.0)
			if err != nil {
				app.showError("Processing Error", err)
				app.mainGUI.UpdateStatus("Processing failed")
				return
			}

			if processedImg != nil {
				app.mainGUI.SetPreviewImage(processedImg)

				// Calculate and display metrics
				originalData := app.pipeline.GetOriginalImage()
				if originalData != nil {
					psnr := app.pipeline.CalculatePSNR(originalData, processedImg)
					ssim := app.pipeline.CalculateSSIM(originalData, processedImg)
					app.mainGUI.UpdateMetrics(psnr, ssim)
				}

				app.mainGUI.UpdateStatus("Preview generated successfully")
			}
		})
	}()
}

func (app *OtsuApp) showError(title string, err error) {
	app.debugManager.LogError("UI", err)
	dialog.ShowError(err, app.window)
}
