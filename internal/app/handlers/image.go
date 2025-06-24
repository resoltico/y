package handlers

import (
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/pipeline"
)

type ImageHandler struct {
	coordinator  *pipeline.Coordinator
	guiManager   *gui.Manager
	debugManager *debug.Manager
}

func NewImageHandler(coord *pipeline.Coordinator, gm *gui.Manager, dm *debug.Manager) *ImageHandler {
	return &ImageHandler{
		coordinator:  coord,
		guiManager:   gm,
		debugManager: dm,
	}
}

func (h *ImageHandler) HandleLoad() {
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
			data, readErr := io.ReadAll(reader)
			originalURI := reader.URI()
			reader.Close()

			if readErr != nil {
				fyne.Do(func() {
					h.showError("Image Read Error", readErr)
					h.guiManager.UpdateStatus("Ready")
				})
				return
			}

			dataReader := &DataReader{data: data, uri: originalURI}
			imageData, loadErr := h.coordinator.LoadImage(dataReader)

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

func (h *ImageHandler) HandleSave() {
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

func (h *ImageHandler) showError(title string, err error) {
	h.debugManager.LogError("ImageHandler", err)
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