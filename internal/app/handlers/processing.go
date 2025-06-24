package handlers

import (
	"fmt"

	"fyne.io/fyne/v2"
	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/pipeline"
)

type ProcessingHandler struct {
	coordinator    *pipeline.Coordinator
	guiManager     *gui.Manager
	debugManager   *debug.Manager
	algorithmMgr   *algorithms.Manager
}

func NewProcessingHandler(coord *pipeline.Coordinator, gm *gui.Manager, dm *debug.Manager) *ProcessingHandler {
	return &ProcessingHandler{
		coordinator:    coord,
		guiManager:     gm,
		debugManager:   dm,
		algorithmMgr:   algorithms.NewManager(),
	}
}

func (h *ProcessingHandler) HandleAlgorithmChange(algorithm string) {
	h.algorithmMgr.SetCurrentAlgorithm(algorithm)
	params := h.algorithmMgr.GetParameters(algorithm)
	h.guiManager.UpdateParameterPanel(algorithm, params)
}

func (h *ProcessingHandler) HandleParameterChange(name string, value interface{}) {
	currentAlgorithm := h.algorithmMgr.GetCurrentAlgorithm()
	h.algorithmMgr.SetParameter(currentAlgorithm, name, value)
}

func (h *ProcessingHandler) HandleGeneratePreview() {
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