package handlers

import (
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/pipeline"
)

type Handlers struct {
	imageHandler      *ImageHandler
	processingHandler *ProcessingHandler
}

func NewHandlers(coord *pipeline.Coordinator, gm *gui.Manager, dm *debug.Manager) *Handlers {
	return &Handlers{
		imageHandler:      NewImageHandler(coord, gm, dm),
		processingHandler: NewProcessingHandler(coord, gm, dm),
	}
}

func (h *Handlers) HandleImageLoad() {
	h.imageHandler.HandleLoad()
}

func (h *Handlers) HandleImageSave() {
	h.imageHandler.HandleSave()
}

func (h *Handlers) HandleAlgorithmChange(algorithm string) {
	h.processingHandler.HandleAlgorithmChange(algorithm)
}

func (h *Handlers) HandleParameterChange(name string, value interface{}) {
	h.processingHandler.HandleParameterChange(name, value)
}

func (h *Handlers) HandleGeneratePreview() {
	h.processingHandler.HandleGeneratePreview()
}