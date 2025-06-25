package app

import (
	"fmt"
	"image/jpeg"
	"image/png"
	"os"
	"strings"

	"otsu-obliterator/internal/algorithms"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/gui"
	"otsu-obliterator/internal/pipeline"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type Handlers struct {
	coordinator  pipeline.ProcessingCoordinator
	guiManager   *gui.Manager
	debugCoord   debug.Coordinator
	logger       debug.Logger
	algorithmMgr *algorithms.Manager
}

func NewHandlers(coord pipeline.ProcessingCoordinator, gm *gui.Manager, debugCoord debug.Coordinator) *Handlers {
	return &Handlers{
		coordinator:  coord,
		guiManager:   gm,
		debugCoord:   debugCoord,
		logger:       debugCoord.Logger(),
		algorithmMgr: algorithms.NewManager(),
	}
}

func (h *Handlers) HandleImageLoad() {
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			h.showError("File Load Error", err)
			return
		}
		if reader == nil {
			return
		}

		h.guiManager.UpdateStatus("Loading image...")

		managedFile, wrapErr := h.wrapWithManagedFile(reader)
		if wrapErr != nil {
			h.showError("File Management Error", wrapErr)
			reader.Close()
			return
		}

		go func() {
			defer managedFile.Close()

			ctx := h.debugCoord.TimingTracker().StartTiming("image_load")
			defer h.debugCoord.TimingTracker().EndTiming(ctx)

			imageData, loadErr := h.coordinator.LoadImage(managedFile)

			fyne.Do(func() {
				if loadErr != nil {
					h.showError("Image Load Error", loadErr)
					h.guiManager.UpdateStatus("Ready")
					return
				}

				if imageData != nil {
					h.guiManager.SetOriginalImage(imageData.Image)
					h.guiManager.UpdateStatus("Image loaded successfully")

					h.logger.Info("Handlers", "image loaded", map[string]interface{}{
						"width":  imageData.Width,
						"height": imageData.Height,
						"format": imageData.Format,
					})
				}
			})
		}()
	}, h.guiManager.GetWindow())
}

func (h *Handlers) HandleImageSave() {
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

		ext := strings.ToLower(writer.URI().Extension())
		if ext == "" {
			h.showExtensionDialog(writer, processedImg)
			return
		}

		h.saveImageWithWriter(writer, processedImg)
	}, h.guiManager.GetWindow())
}

func (h *Handlers) HandleAlgorithmChange(algorithm string) {
	h.logger.Debug("Handlers", "algorithm change", map[string]interface{}{
		"from": h.algorithmMgr.GetCurrentAlgorithm(),
		"to":   algorithm,
	})

	h.algorithmMgr.SetCurrentAlgorithm(algorithm)
	params := h.algorithmMgr.GetParameters(algorithm)
	h.guiManager.UpdateParameterPanel(algorithm, params)
}

func (h *Handlers) HandleParameterChange(name string, value interface{}) {
	currentAlgorithm := h.algorithmMgr.GetCurrentAlgorithm()
	h.algorithmMgr.SetParameter(currentAlgorithm, name, value)

	h.logger.Debug("Handlers", "parameter changed", map[string]interface{}{
		"algorithm": currentAlgorithm,
		"parameter": name,
		"value":     value,
	})
}

func (h *Handlers) HandleGeneratePreview() {
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
		ctx := h.debugCoord.TimingTracker().StartTiming("image_processing")
		defer h.debugCoord.TimingTracker().EndTiming(ctx)

		currentAlgorithm := h.algorithmMgr.GetCurrentAlgorithm()
		params := h.algorithmMgr.GetAllParameters(currentAlgorithm)

		h.logger.Info("Handlers", "processing started", map[string]interface{}{
			"algorithm": currentAlgorithm,
		})

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
				h.logger.Info("Handlers", "processing completed", map[string]interface{}{
					"algorithm": currentAlgorithm,
					"width":     processedImg.Width,
					"height":    processedImg.Height,
				})
			}
		})
	}()
}

func (h *Handlers) showExtensionDialog(writer fyne.URIWriteCloser, processedImg *pipeline.ImageData) {
	content := widget.NewLabel("No file extension detected. Please choose a format:")

	formatSelect := widget.NewSelect([]string{"PNG", "JPEG"}, nil)
	formatSelect.SetSelected("PNG")

	form := container.NewVBox(
		content,
		formatSelect,
	)

	dialog.ShowCustomConfirm("Choose File Format", "Save", "Cancel",
		form, func(save bool) {
			if save && formatSelect.Selected != "" {
				basePath := writer.URI().Path()
				ext := ".png"
				if formatSelect.Selected == "JPEG" {
					ext = ".jpg"
				}

				// Remove the empty file created by Fyne
				os.Remove(basePath)
				writer.Close()

				h.logger.Info("Handlers", "adding extension", map[string]interface{}{
					"original":  basePath,
					"format":    formatSelect.Selected,
					"extension": ext,
				})

				h.saveImageWithExtension(basePath+ext, processedImg, formatSelect.Selected)
			} else {
				// Remove empty file if user cancels
				os.Remove(writer.URI().Path())
				writer.Close()
			}
		}, h.guiManager.GetWindow())
}

func (h *Handlers) saveImageWithExtension(filepath string, processedImg *pipeline.ImageData, format string) {
	h.guiManager.UpdateStatus("Saving image...")

	go func() {
		ctx := h.debugCoord.TimingTracker().StartTiming("image_save")
		defer h.debugCoord.TimingTracker().EndTiming(ctx)

		file, err := os.Create(filepath)
		if err != nil {
			fyne.Do(func() {
				h.showError("File Create Error", err)
			})
			return
		}
		defer file.Close()

		// Save directly using image package
		var saveErr error
		formatLower := strings.ToLower(format)

		switch formatLower {
		case "jpeg":
			saveErr = jpeg.Encode(file, processedImg.Image, &jpeg.Options{Quality: 95})
		case "png":
			fallthrough
		default:
			saveErr = png.Encode(file, processedImg.Image)
		}

		fyne.Do(func() {
			if saveErr != nil {
				h.showError("Image Save Error", saveErr)
			} else {
				h.guiManager.UpdateStatus("Image saved successfully")
				h.logger.Info("Handlers", "image saved with extension", map[string]interface{}{
					"path":   filepath,
					"format": format,
				})
			}
		})
	}()
}

func (h *Handlers) saveImageWithWriter(writer fyne.URIWriteCloser, processedImg *pipeline.ImageData) {
	h.guiManager.UpdateStatus("Saving image...")

	managedWriter, wrapErr := h.wrapWithManagedWriter(writer)
	if wrapErr != nil {
		h.showError("File Management Error", wrapErr)
		writer.Close()
		return
	}

	go func() {
		defer managedWriter.Close()

		ctx := h.debugCoord.TimingTracker().StartTiming("image_save")
		defer h.debugCoord.TimingTracker().EndTiming(ctx)

		saveErr := h.coordinator.SaveImage(managedWriter, processedImg)

		fyne.Do(func() {
			if saveErr != nil {
				h.showError("Image Save Error", saveErr)
			} else {
				h.guiManager.UpdateStatus("Image saved successfully")
				h.logger.Info("Handlers", "image saved", map[string]interface{}{
					"path": writer.URI().Path(),
				})
			}
		})
	}()
}

func (h *Handlers) showError(title string, err error) {
	h.logger.Error("Handlers", err, map[string]interface{}{
		"title": title,
	})
	dialog.ShowError(err, h.guiManager.GetWindow())
}

func (h *Handlers) wrapWithManagedFile(reader fyne.URIReadCloser) (*ManagedReadCloser, error) {
	return NewManagedReadCloser(reader, h.debugCoord.FileTracker()), nil
}

func (h *Handlers) wrapWithManagedWriter(writer fyne.URIWriteCloser) (*ManagedWriteCloser, error) {
	return NewManagedWriteCloser(writer, h.debugCoord.FileTracker()), nil
}

type FileTracker interface {
	TrackOpen(path string, handle uintptr)
	TrackClose(path string, handle uintptr)
}

type ManagedReadCloser struct {
	reader      fyne.URIReadCloser
	fileTracker FileTracker
	closed      bool
}

func NewManagedReadCloser(reader fyne.URIReadCloser, tracker FileTracker) *ManagedReadCloser {
	mrc := &ManagedReadCloser{
		reader:      reader,
		fileTracker: tracker,
		closed:      false,
	}

	if tracker != nil {
		tracker.TrackOpen(reader.URI().Path(), uintptr(0))
	}

	return mrc
}

func (mrc *ManagedReadCloser) Read(p []byte) (n int, err error) {
	if mrc.closed {
		return 0, fmt.Errorf("file already closed")
	}
	return mrc.reader.Read(p)
}

func (mrc *ManagedReadCloser) Close() error {
	if mrc.closed {
		return nil
	}

	mrc.closed = true

	if mrc.fileTracker != nil {
		mrc.fileTracker.TrackClose(mrc.reader.URI().Path(), uintptr(0))
	}

	return mrc.reader.Close()
}

func (mrc *ManagedReadCloser) URI() fyne.URI {
	return mrc.reader.URI()
}

type ManagedWriteCloser struct {
	writer      fyne.URIWriteCloser
	fileTracker FileTracker
	closed      bool
}

func NewManagedWriteCloser(writer fyne.URIWriteCloser, tracker FileTracker) *ManagedWriteCloser {
	mwc := &ManagedWriteCloser{
		writer:      writer,
		fileTracker: tracker,
		closed:      false,
	}

	if tracker != nil {
		tracker.TrackOpen(writer.URI().Path(), uintptr(0))
	}

	return mwc
}

func (mwc *ManagedWriteCloser) Write(p []byte) (n int, err error) {
	if mwc.closed {
		return 0, fmt.Errorf("file already closed")
	}
	return mwc.writer.Write(p)
}

func (mwc *ManagedWriteCloser) Close() error {
	if mwc.closed {
		return nil
	}

	mwc.closed = true

	if mwc.fileTracker != nil {
		mwc.fileTracker.TrackClose(mwc.writer.URI().Path(), uintptr(0))
	}

	return mwc.writer.Close()
}

func (mwc *ManagedWriteCloser) URI() fyne.URI {
	return mwc.writer.URI()
}
