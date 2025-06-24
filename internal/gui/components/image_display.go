package components

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	ImageDisplayWidth  = 640
	ImageDisplayHeight = 480
)

type ImageDisplay struct {
	container     *fyne.Container
	originalImage *canvas.Image
	previewImage  *canvas.Image
}

func NewImageDisplay() *ImageDisplay {
	originalImage := canvas.NewImageFromImage(nil)
	originalImage.FillMode = canvas.ImageFillContain
	originalImage.SetMinSize(fyne.NewSize(ImageDisplayWidth, ImageDisplayHeight))

	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.SetMinSize(fyne.NewSize(ImageDisplayWidth, ImageDisplayHeight))

	originalContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		originalImage,
	)

	previewContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		previewImage,
	)

	imageSplit := container.NewHSplit(originalContainer, previewContainer)
	imageSplit.SetOffset(0.5)

	mainContainer := container.NewBorder(
		nil, nil, nil, nil,
		imageSplit,
	)

	return &ImageDisplay{
		container:     mainContainer,
		originalImage: originalImage,
		previewImage:  previewImage,
	}
}

func (id *ImageDisplay) GetContainer() *fyne.Container {
	return id.container
}

func (id *ImageDisplay) SetOriginalImage(img image.Image) {
	if img == nil {
		return
	}

	id.originalImage.Image = img
	id.originalImage.Refresh()
}

func (id *ImageDisplay) SetPreviewImage(img image.Image) {
	if img == nil {
		return
	}

	id.previewImage.Image = img
	id.previewImage.Refresh()
}