package components

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type StatusBar struct {
	container   *fyne.Container
	statusLabel *widget.Label
	psnrLabel   *widget.Label
	ssimLabel   *widget.Label
}

func NewStatusBar() *StatusBar {
	statusLabel := widget.NewLabel("Ready")
	psnrLabel := widget.NewLabel("PSNR: --")
	ssimLabel := widget.NewLabel("SSIM: --")

	metricsContainer := container.NewHBox(
		psnrLabel,
		widget.NewSeparator(),
		ssimLabel,
	)

	mainContainer := container.NewBorder(
		nil, nil,
		statusLabel,
		metricsContainer,
	)

	return &StatusBar{
		container:   mainContainer,
		statusLabel: statusLabel,
		psnrLabel:   psnrLabel,
		ssimLabel:   ssimLabel,
	}
}

func (sb *StatusBar) GetContainer() *fyne.Container {
	return sb.container
}

func (sb *StatusBar) SetStatus(status string) {
	sb.statusLabel.SetText(status)
}

func (sb *StatusBar) SetMetrics(psnr, ssim float64) {
	sb.psnrLabel.SetText(fmt.Sprintf("PSNR: %.2f dB", psnr))
	sb.ssimLabel.SetText(fmt.Sprintf("SSIM: %.4f", ssim))
}