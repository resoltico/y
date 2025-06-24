package sync

import (
	"image"
	"otsu-obliterator/internal/gui/components"
)

type ImageDisplayHandler interface {
	SetOriginalImage(image.Image)
	SetPreviewImage(image.Image)
}

type ParameterPanelHandler interface {
	UpdateParameters(string, map[string]interface{})
}

type StatusBarHandler interface {
	SetStatus(string)
	SetMetrics(float64, float64)
}

type ProgressBarHandler interface {
	SetProgress(float64)
}

type UpdateProcessor struct {
	imageDisplay  ImageDisplayHandler
	parameterPanel ParameterPanelHandler
	statusBar     StatusBarHandler
	progressBar   ProgressBarHandler
}

func NewUpdateProcessor() *UpdateProcessor {
	return &UpdateProcessor{}
}

func (p *UpdateProcessor) SetImageDisplay(display ImageDisplayHandler) {
	p.imageDisplay = display
}

func (p *UpdateProcessor) SetParameterPanel(panel ParameterPanelHandler) {
	p.parameterPanel = panel
}

func (p *UpdateProcessor) SetStatusBar(statusBar StatusBarHandler) {
	p.statusBar = statusBar
}

func (p *UpdateProcessor) SetProgressBar(progressBar ProgressBarHandler) {
	p.progressBar = progressBar
}

func (p *UpdateProcessor) ProcessUpdate(update *Update) {
	switch update.Type {
	case UpdateTypeImageDisplay:
		if p.imageDisplay != nil {
			if data, ok := update.Data.(*components.ImageDisplayUpdate); ok {
				switch data.Type {
				case components.ImageTypeOriginal:
					p.imageDisplay.SetOriginalImage(data.Image)
				case components.ImageTypePreview:
					p.imageDisplay.SetPreviewImage(data.Image)
				}
			}
		}
		
	case UpdateTypeParameterPanel:
		if p.parameterPanel != nil {
			if data, ok := update.Data.(*components.ParameterPanelUpdate); ok {
				p.parameterPanel.UpdateParameters(data.Algorithm, data.Parameters)
			}
		}
		
	case UpdateTypeStatus:
		if p.statusBar != nil {
			if data, ok := update.Data.(*components.StatusUpdate); ok {
				p.statusBar.SetStatus(data.Status)
			}
		}
		
	case UpdateTypeProgress:
		if p.progressBar != nil {
			if data, ok := update.Data.(*components.ProgressUpdate); ok {
				p.progressBar.SetProgress(data.Progress)
			}
		}
		
	case UpdateTypeMetrics:
		if p.statusBar != nil {
			if data, ok := update.Data.(*components.MetricsUpdate); ok {
				p.statusBar.SetMetrics(data.PSNR, data.SSIM)
			}
		}
	}
}