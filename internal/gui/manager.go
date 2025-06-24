package gui

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui/components"
	"otsu-obliterator/internal/gui/layout"
	"otsu-obliterator/internal/gui/sync"
)

type Manager struct {
	window       fyne.Window
	layoutMgr    *layout.Manager
	syncCoord    *sync.Coordinator
	debugMgr     *debug.Manager
	isShutdown   bool
	
	// Event handlers
	imageLoadHandler       func()
	imageSaveHandler       func()
	algorithmChangeHandler func(string)
	parameterChangeHandler func(string, interface{})
	generatePreviewHandler func()
}

func NewManager(window fyne.Window, debugMgr *debug.Manager) (*Manager, error) {
	syncCoord := sync.NewCoordinator()
	
	layoutMgr, err := layout.NewManager(syncCoord)
	if err != nil {
		return nil, err
	}
	
	manager := &Manager{
		window:     window,
		layoutMgr:  layoutMgr,
		syncCoord:  syncCoord,
		debugMgr:   debugMgr,
		isShutdown: false,
	}
	
	// Start sync coordinator
	go syncCoord.Run()
	
	return manager, nil
}

func (m *Manager) GetMainContainer() *fyne.Container {
	return m.layoutMgr.GetMainContainer()
}

func (m *Manager) GetWindow() fyne.Window {
	return m.window
}

func (m *Manager) SetImageLoadHandler(handler func()) {
	m.imageLoadHandler = handler
	m.layoutMgr.SetImageLoadHandler(handler)
}

func (m *Manager) SetImageSaveHandler(handler func()) {
	m.imageSaveHandler = handler
	m.layoutMgr.SetImageSaveHandler(handler)
}

func (m *Manager) SetAlgorithmChangeHandler(handler func(string)) {
	m.algorithmChangeHandler = handler
	m.layoutMgr.SetAlgorithmChangeHandler(handler)
}

func (m *Manager) SetParameterChangeHandler(handler func(string, interface{})) {
	m.parameterChangeHandler = handler
	m.layoutMgr.SetParameterChangeHandler(handler)
}

func (m *Manager) SetGeneratePreviewHandler(handler func()) {
	m.generatePreviewHandler = handler
	m.layoutMgr.SetGeneratePreviewHandler(handler)
}

func (m *Manager) SetOriginalImage(img image.Image) {
	update := &sync.Update{
		Type: sync.UpdateTypeImageDisplay,
		Data: &components.ImageDisplayUpdate{
			Type:  components.ImageTypeOriginal,
			Image: img,
		},
	}
	m.syncCoord.ScheduleUpdate(update)
}

func (m *Manager) SetPreviewImage(img image.Image) {
	update := &sync.Update{
		Type: sync.UpdateTypeImageDisplay,
		Data: &components.ImageDisplayUpdate{
			Type:  components.ImageTypePreview,
			Image: img,
		},
	}
	m.syncCoord.ScheduleUpdate(update)
}

func (m *Manager) UpdateParameterPanel(algorithm string, params map[string]interface{}) {
	update := &sync.Update{
		Type: sync.UpdateTypeParameterPanel,
		Data: &components.ParameterPanelUpdate{
			Algorithm:  algorithm,
			Parameters: params,
		},
	}
	m.syncCoord.ScheduleUpdate(update)
}

func (m *Manager) UpdateStatus(status string) {
	update := &sync.Update{
		Type: sync.UpdateTypeStatus,
		Data: &components.StatusUpdate{
			Status: status,
		},
	}
	m.syncCoord.ScheduleUpdate(update)
}

func (m *Manager) UpdateProgress(progress float64) {
	update := &sync.Update{
		Type: sync.UpdateTypeProgress,
		Data: &components.ProgressUpdate{
			Progress: progress,
		},
	}
	m.syncCoord.ScheduleUpdate(update)
}

func (m *Manager) UpdateMetrics(psnr, ssim float64) {
	update := &sync.Update{
		Type: sync.UpdateTypeMetrics,
		Data: &components.MetricsUpdate{
			PSNR: psnr,
			SSIM: ssim,
		},
	}
	m.syncCoord.ScheduleUpdate(update)
}

func (m *Manager) ShowError(title string, err error) {
	m.debugMgr.LogError("GUIManager", err)
	fyne.Do(func() {
		dialog.ShowError(err, m.window)
	})
}

func (m *Manager) Shutdown() {
	if m.isShutdown {
		return
	}
	
	m.isShutdown = true
	
	if m.syncCoord != nil {
		m.syncCoord.Stop()
	}
}