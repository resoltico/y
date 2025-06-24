package layout

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"otsu-obliterator/internal/gui/components"
	"otsu-obliterator/internal/gui/sync"
)

type Manager struct {
	mainContainer     *fyne.Container
	imageDisplay      *components.ImageDisplay
	controlsPanel     *components.ControlsPanel
	statusBar         *components.StatusBar
	syncCoord         *sync.Coordinator
}

func NewManager(syncCoord *sync.Coordinator) (*Manager, error) {
	imageDisplay := components.NewImageDisplay()
	controlsPanel := components.NewControlsPanel()
	statusBar := components.NewStatusBar()

	mainContainer := container.NewBorder(
		imageDisplay.GetContainer(),
		statusBar.GetContainer(),
		nil,
		nil,
		controlsPanel.GetContainer(),
	)

	manager := &Manager{
		mainContainer: mainContainer,
		imageDisplay:  imageDisplay,
		controlsPanel: controlsPanel,
		statusBar:     statusBar,
		syncCoord:     syncCoord,
	}

	syncCoord.SetImageDisplay(imageDisplay)
	syncCoord.SetParameterPanel(controlsPanel)
	syncCoord.SetStatusBar(statusBar)
	syncCoord.SetProgressBar(controlsPanel)

	return manager, nil
}

func (m *Manager) GetMainContainer() *fyne.Container {
	return m.mainContainer
}

func (m *Manager) SetImageLoadHandler(handler func()) {
	m.controlsPanel.SetImageLoadHandler(handler)
}

func (m *Manager) SetImageSaveHandler(handler func()) {
	m.controlsPanel.SetImageSaveHandler(handler)
}

func (m *Manager) SetAlgorithmChangeHandler(handler func(string)) {
	m.controlsPanel.SetAlgorithmChangeHandler(handler)
}

func (m *Manager) SetParameterChangeHandler(handler func(string, interface{})) {
	m.controlsPanel.SetParameterChangeHandler(handler)
}

func (m *Manager) SetGeneratePreviewHandler(handler func()) {
	m.controlsPanel.SetGeneratePreviewHandler(handler)
}