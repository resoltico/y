package gui

import (
	"fmt"
	"image"

	"otsu-obliterator/internal/gui/widgets"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// View handles all UI components and their layout
type View struct {
	window     fyne.Window
	controller *Controller

	// UI components
	toolbar        *widgets.Toolbar
	imageDisplay   *widgets.ImageDisplay
	parameterPanel *widgets.ParameterPanel
	mainContainer  *fyne.Container
}

func NewView(window fyne.Window) *View {
	view := &View{
		window: window,
	}

	view.setupComponents()
	view.setupLayout()

	return view
}

func (v *View) SetController(controller *Controller) {
	v.controller = controller
	v.setupEventHandlers()
}

func (v *View) setupComponents() {
	v.toolbar = widgets.NewToolbar()
	v.imageDisplay = widgets.NewImageDisplay()
	v.parameterPanel = widgets.NewParameterPanel()
}

func (v *View) setupLayout() {
	v.mainContainer = container.NewVBox(
		v.imageDisplay.GetContainer(),
		v.toolbar.GetContainer(),
		v.parameterPanel.GetContainer(),
	)
}

func (v *View) setupEventHandlers() {
	if v.controller == nil {
		return
	}

	// Toolbar event handlers
	v.toolbar.SetLoadHandler(v.controller.LoadImage)
	v.toolbar.SetSaveHandler(v.controller.SaveImage)
	v.toolbar.SetProcessHandler(v.controller.ProcessImage)
	v.toolbar.SetAlgorithmChangeHandler(v.controller.ChangeAlgorithm)

	// Parameter panel event handlers
	v.parameterPanel.SetParameterChangeHandler(v.controller.UpdateParameter)
}

// Public interface for controller
func (v *View) GetMainContainer() *fyne.Container {
	return v.mainContainer
}

func (v *View) SetOriginalImage(img image.Image) {
	v.imageDisplay.SetOriginalImage(img)
}

func (v *View) SetPreviewImage(img image.Image) {
	v.imageDisplay.SetPreviewImage(img)
}

func (v *View) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	v.parameterPanel.UpdateParameters(algorithm, params)
}

func (v *View) SetStatus(status string) {
	v.toolbar.SetStatus(status)
}

func (v *View) SetProgress(progress float64) {
	if progress > 0 && progress < 1 {
		v.toolbar.SetProgress(fmt.Sprintf("[%.0f%%]", progress*100))
	} else {
		v.toolbar.SetProgress("")
	}
}

func (v *View) SetMetrics(psnr, ssim float64) {
	v.toolbar.SetMetrics(psnr, ssim)
}

func (v *View) ShowError(title string, err error) {
	dialog.ShowError(err, v.window)
}

func (v *View) ShowFileDialog(callback func(fyne.URIReadCloser, error)) {
	dialog.ShowFileOpen(callback, v.window)
}

func (v *View) ShowSaveDialog(callback func(fyne.URIWriteCloser, error)) {
	dialog.ShowFileSave(callback, v.window)
}

func (v *View) ShowFormatSelectionDialog(callback func(string, bool)) {
	content := widget.NewLabel("No file extension detected. Please choose a format:")

	formatSelect := widget.NewSelect([]string{"PNG", "JPEG"}, nil)
	formatSelect.SetSelected("PNG")

	form := container.NewVBox(
		content,
		formatSelect,
	)

	dialog.ShowCustomConfirm("Choose File Format", "Save", "Cancel",
		form, func(confirmed bool) {
			if confirmed && formatSelect.Selected != "" {
				callback(formatSelect.Selected, true)
			} else {
				callback("", false)
			}
		}, v.window)
}

// Window management
func (v *View) GetWindow() fyne.Window {
	return v.window
}

func (v *View) Show() {
	v.window.SetContent(v.mainContainer)
	v.window.Show()
}

func (v *View) Shutdown() {
	// View cleanup if needed
}
