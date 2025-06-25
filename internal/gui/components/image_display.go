package components

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	DefaultImageWidth  = 320
	DefaultImageHeight = 240
)

type ImageDisplay struct {
	container      fyne.CanvasObject
	originalImage  *canvas.Image
	previewImage   *canvas.Image
	originalScroll *container.Scroll
	previewScroll  *container.Scroll
	splitContainer *container.Split
}

func NewImageDisplay() *ImageDisplay {
	originalImage := canvas.NewImageFromImage(nil)
	originalImage.FillMode = canvas.ImageFillContain
	originalImage.ScaleMode = canvas.ImageScaleSmooth

	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.ScaleMode = canvas.ImageScaleSmooth

	originalScroll := container.NewScroll(originalImage)
	originalScroll.SetMinSize(fyne.NewSize(DefaultImageWidth, DefaultImageHeight))

	previewScroll := container.NewScroll(previewImage)
	previewScroll.SetMinSize(fyne.NewSize(DefaultImageWidth, DefaultImageHeight))

	originalContainer := container.NewMax(originalScroll)
	originalWithLabel := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		originalContainer,
	)

	previewContainer := container.NewMax(previewScroll)
	previewWithLabel := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		previewContainer,
	)

	splitContainer := container.NewHSplit(originalWithLabel, previewWithLabel)
	splitContainer.SetOffset(0.5)

	return &ImageDisplay{
		container:      splitContainer,
		originalImage:  originalImage,
		previewImage:   previewImage,
		originalScroll: originalScroll,
		previewScroll:  previewScroll,
		splitContainer: splitContainer,
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

func (id *ImageDisplay) GetMinimumSize() fyne.Size {
	labelHeight := float32(30)
	padding := float32(20)

	return fyne.Size{
		Width:  DefaultImageWidth*2 + padding,
		Height: DefaultImageHeight + labelHeight + padding,
	}
}
