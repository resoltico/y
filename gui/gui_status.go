package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type StatusBar struct {
	container        *fyne.Container
	statusLabel      *widget.Label
	psnrLabel        *widget.Label
	ssimLabel        *widget.Label
	metricsContainer *fyne.Container
}

func NewStatusBar() *StatusBar {
	statusBar := &StatusBar{}
	statusBar.setupStatusBar()
	return statusBar
}

func (sb *StatusBar) setupStatusBar() {
	// Status label
	sb.statusLabel = widget.NewLabel("Ready")

	// Metrics labels
	sb.psnrLabel = widget.NewLabel("PSNR: --")
	sb.ssimLabel = widget.NewLabel("SSIM: --")

	// Metrics container
	sb.metricsContainer = container.NewHBox(
		sb.psnrLabel,
		widget.NewSeparator(),
		sb.ssimLabel,
	)

	// Main container with status on left, metrics on right
	sb.container = container.NewBorder(
		nil, nil,
		sb.statusLabel,
		sb.metricsContainer,
	)
}

func (sb *StatusBar) SetStatus(status string) {
	sb.statusLabel.SetText(status)
}

func (sb *StatusBar) SetMetrics(psnr, ssim string) {
	sb.psnrLabel.SetText(psnr)
	sb.ssimLabel.SetText(ssim)
}

func (sb *StatusBar) GetContainer() *fyne.Container {
	return sb.container
}
