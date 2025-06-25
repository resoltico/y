package components

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	MinImageWidth  = 640
	MinImageHeight = 480
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
	originalImage.SetMinSize(fyne.NewSize(MinImageWidth, MinImageHeight))

	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.ScaleMode = canvas.ImageScaleSmooth
	previewImage.SetMinSize(fyne.NewSize(MinImageWidth, MinImageHeight))

	originalScroll := container.NewScroll(originalImage)
	originalScroll.SetMinSize(fyne.NewSize(MinImageWidth, MinImageHeight))

	previewScroll := container.NewScroll(previewImage)
	previewScroll.SetMinSize(fyne.NewSize(MinImageWidth, MinImageHeight))

	originalContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		originalScroll,
	)

	previewContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		previewScroll,
	)

	splitContainer := container.NewHSplit(originalContainer, previewContainer)
	splitContainer.SetOffset(0.5)
	splitContainer.Resize(fyne.NewSize(MinImageWidth*2, MinImageHeight+60))

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
		Width:  MinImageWidth*2 + padding,
		Height: MinImageHeight + labelHeight + padding,
	}
}
