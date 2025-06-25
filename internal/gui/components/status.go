package components

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type StatusBar struct {
	container     *fyne.Container
	statusLabel   *widget.Label
	progressLabel *widget.Label
	psnrLabel     *widget.Label
	ssimLabel     *widget.Label
	spinner       *TextSpinner
}

type TextSpinner struct {
	chars   []string
	current int
	active  bool
}

func NewTextSpinner() *TextSpinner {
	return &TextSpinner{
		chars:   []string{"|", "/", "-", "\\"},
		current: 0,
		active:  false,
	}
}

func (ts *TextSpinner) Next() string {
	if !ts.active {
		return ""
	}
	char := ts.chars[ts.current]
	ts.current = (ts.current + 1) % len(ts.chars)
	return char
}

func (ts *TextSpinner) Start() {
	ts.active = true
}

func (ts *TextSpinner) Stop() {
	ts.active = false
}

func NewStatusBar() *StatusBar {
	statusLabel := widget.NewLabel("Ready")
	progressLabel := widget.NewLabel("")
	psnrLabel := widget.NewLabel("PSNR: --")
	ssimLabel := widget.NewLabel("SSIM: --")
	spinner := NewTextSpinner()

	progressContainer := container.NewHBox(
		statusLabel,
		progressLabel,
	)

	metricsContainer := container.NewHBox(
		psnrLabel,
		widget.NewSeparator(),
		ssimLabel,
	)

	mainContainer := container.NewBorder(
		nil, nil,
		progressContainer,
		metricsContainer,
	)

	statusBar := &StatusBar{
		container:     mainContainer,
		statusLabel:   statusLabel,
		progressLabel: progressLabel,
		psnrLabel:     psnrLabel,
		ssimLabel:     ssimLabel,
		spinner:       spinner,
	}

	return statusBar
}

func (sb *StatusBar) GetContainer() *fyne.Container {
	return sb.container
}

func (sb *StatusBar) SetStatus(status string) {
	sb.statusLabel.SetText(status)

	if strings.Contains(strings.ToLower(status), "processing") ||
		strings.Contains(strings.ToLower(status), "loading") ||
		strings.Contains(strings.ToLower(status), "generating") {
		sb.spinner.Start()
		sb.startSpinnerAnimation()
	} else {
		sb.spinner.Stop()
		sb.progressLabel.SetText("")
	}
}

func (sb *StatusBar) SetProgress(progress float64) {
	if progress > 0 && progress < 1 {
		percentage := int(progress * 100)
		sb.progressLabel.SetText(fmt.Sprintf(" [%d%%]", percentage))
		sb.spinner.Start()
		sb.startSpinnerAnimation()
	} else {
		sb.progressLabel.SetText("")
		sb.spinner.Stop()
	}
}

func (sb *StatusBar) SetMetrics(psnr, ssim float64) {
	sb.psnrLabel.SetText(fmt.Sprintf("PSNR: %.2f dB", psnr))
	sb.ssimLabel.SetText(fmt.Sprintf("SSIM: %.4f", ssim))
}

func (sb *StatusBar) startSpinnerAnimation() {
	if !sb.spinner.active {
		return
	}

	go func() {
		for sb.spinner.active {
			fyne.Do(func() {
				if sb.spinner.active {
					currentText := sb.progressLabel.Text
					if len(currentText) > 0 && (strings.HasSuffix(currentText, "|") ||
						strings.HasSuffix(currentText, "/") ||
						strings.HasSuffix(currentText, "-") ||
						strings.HasSuffix(currentText, "\\")) {
						currentText = currentText[:len(currentText)-1]
					}
					sb.progressLabel.SetText(currentText + sb.spinner.Next())
				}
			})
			time.Sleep(250 * time.Millisecond)
		}
		fyne.Do(func() {
			currentText := sb.progressLabel.Text
			if len(currentText) > 0 && (strings.HasSuffix(currentText, "|") ||
				strings.HasSuffix(currentText, "/") ||
				strings.HasSuffix(currentText, "-") ||
				strings.HasSuffix(currentText, "\\")) {
				sb.progressLabel.SetText(currentText[:len(currentText)-1])
			}
		})
	}()
}
