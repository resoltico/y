package components

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	ImageAreaWidth  = 500
	ImageAreaHeight = 400
)

type ImageDisplay struct {
	container     fyne.CanvasObject
	originalImage *canvas.Image
	previewImage  *canvas.Image
}

func NewImageDisplay() *ImageDisplay {
	originalImage := canvas.NewImageFromImage(nil)
	originalImage.FillMode = canvas.ImageFillContain
	originalImage.ScaleMode = canvas.ImageScaleSmooth
	originalImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))

	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.ScaleMode = canvas.ImageScaleSmooth
	previewImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))

	originalContainer := container.NewBorder(
		widget.NewRichTextFromMarkdown("**Original**"),
		nil, nil, nil,
		originalImage,
	)

	previewContainer := container.NewBorder(
		widget.NewRichTextFromMarkdown("**Preview**"),
		nil, nil, nil,
		previewImage,
	)

	splitContainer := container.NewHSplit(originalContainer, previewContainer)
	splitContainer.SetOffset(0.5)

	return &ImageDisplay{
		container:     splitContainer,
		originalImage: originalImage,
		previewImage:  previewImage,
	}
}

func (id *ImageDisplay) GetContainer() fyne.CanvasObject {
	return id.container
}

func (id *ImageDisplay) SetOriginalImage(img image.Image) {
	if img == nil {
		id.originalImage.Image = nil
		id.originalImage.Refresh()
		return
	}

	id.originalImage.Image = img
	id.originalImage.Refresh()
}

func (id *ImageDisplay) SetPreviewImage(img image.Image) {
	if img == nil {
		id.previewImage.Image = nil
		id.previewImage.Refresh()
		return
	}

	id.previewImage.Image = img
	id.previewImage.Refresh()
}
