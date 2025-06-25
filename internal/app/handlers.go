package app

import (
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/pipeline"
)

type Handlers struct {
	coordinator    pipeline.ProcessingCoordinator
	guiManager     *gui.Manager
	debugManager   *debug.Manager
	algorithmMgr   *algorithms.Manager
}

func NewHandlers(coord pipeline.ProcessingCoordinator, gm *gui.Manager, dm *debug.Manager) *Handlers {
	return &Handlers{
		coordinator:  coord,
		guiManager:   gm,
		debugManager: dm,
		algorithmMgr: algorithms.NewManager(),
	}
}

func (h *Handlers) HandleImageLoad() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			h.showError("File Load Error", err)
			return
		}
		if reader == nil {
			return
		}

		h.guiManager.UpdateStatus("Loading image...")

		go func() {
			imageData, loadErr := h.coordinator.LoadImage(reader)
			reader.Close()

			fyne.Do(func() {
				if loadErr != nil {
					h.showError("Image Load Error", loadErr)
					h.guiManager.UpdateStatus("Ready")
					return
				}

				if imageData != nil {
					h.guiManager.SetOriginalImage(imageData.Image)
					h.guiManager.UpdateStatus("Image loaded successfully")
				}
			})
		}()
	}, h.guiManager.GetWindow())
}

func (h *Handlers) HandleImageSave() {
	processedImg := h.coordinator.GetProcessedImage()
	if processedImg == nil {
		h.showError("Save Error", fmt.Errorf("no processed image to save"))
		return
	}

	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			h.showError("File Save Error", err)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		h.guiManager.UpdateStatus("Saving image...")

		go func() {
			err := h.coordinator.SaveImage(writer, processedImg)
			fyne.Do(func() {
				if err != nil {
					h.showError("Image Save Error", err)
				} else {
					h.guiManager.UpdateStatus("Image saved successfully")
				}
			})
		}()
	}, h.guiManager.GetWindow())
}

func (h *Handlers) HandleAlgorithmChange(algorithm string) {
	h.algorithmMgr.SetCurrentAlgorithm(algorithm)
	params := h.algorithmMgr.GetParameters(algorithm)
	h.guiManager.UpdateParameterPanel(algorithm, params)
}

func (h *Handlers) HandleParameterChange(name string, value interface{}) {
	currentAlgorithm := h.algorithmMgr.GetCurrentAlgorithm()
	h.algorithmMgr.SetParameter(currentAlgorithm, name, value)
}

func (h *Handlers) HandleGeneratePreview() {
	originalImg := h.coordinator.GetOriginalImage()
	if originalImg == nil {
		h.guiManager.ShowError("Processing Error", fmt.Errorf("no image loaded"))
		return
	}

	fyne.Do(func() {
		h.guiManager.UpdateStatus("Generating preview...")
		h.guiManager.UpdateProgress(0.0)
	})

	go func() {
		currentAlgorithm := h.algorithmMgr.GetCurrentAlgorithm()
		params := h.algorithmMgr.GetAllParameters(currentAlgorithm)

		processedImg, err := h.coordinator.ProcessImage(currentAlgorithm, params)

		fyne.Do(func() {
			h.guiManager.UpdateProgress(1.0)
			if err != nil {
				h.guiManager.ShowError("Processing Error", err)
				h.guiManager.UpdateStatus("Processing failed")
				return
			}

			if processedImg != nil {
				h.guiManager.SetPreviewImage(processedImg.Image)

				originalData := h.coordinator.GetOriginalImage()
				if originalData != nil {
					psnr := h.coordinator.CalculatePSNR(originalData, processedImg)
					ssim := h.coordinator.CalculateSSIM(originalData, processedImg)
					h.guiManager.UpdateMetrics(psnr, ssim)
				}

				h.guiManager.UpdateStatus("Preview generated successfully")
			}
		})
	}()
}

func (h *Handlers) showError(title string, err error) {
	h.debugManager.LogError("Handlers", err)
	dialog.ShowError(err, h.guiManager.GetWindow())
}

type DataReader struct {
	data []byte
	pos  int
	uri  fyne.URI
}

func (dr *DataReader) Read(p []byte) (n int, err error) {
	if dr.pos >= len(dr.data) {
		return 0, io.EOF
	}
	n = copy(p, dr.data[dr.pos:])
	dr.pos += n
	return n, nil
}

func (dr *DataReader) Close() error {
	return nil
}

func (dr *DataReader) URI() fyne.URI {
	return dr.uri
}
