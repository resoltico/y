package components

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const (
	ScrollViewportWidth  = 500
	ScrollViewportHeight = 400
	ImageDisplayWidth    = 800
	ImageDisplayHeight   = 600
)

type ImageDisplay struct {
	container       *fyne.Container
	originalImage   *canvas.Image
	previewImage    *canvas.Image
	scrollContainer *container.Scroll
}

func NewImageDisplay() *ImageDisplay {
	// Create placeholder images with scrollable dimensions
	originalImage := canvas.NewImageFromImage(nil)
	originalImage.FillMode = canvas.ImageFillOriginal
	originalImage.SetMinSize(fyne.NewSize(ImageDisplayWidth, ImageDisplayHeight))

	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillOriginal
	previewImage.SetMinSize(fyne.NewSize(ImageDisplayWidth, ImageDisplayHeight))

	// Create labeled containers for each image
	originalContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		originalImage,
	)

	previewContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		previewImage,
	)

	// Layout images horizontally for side-by-side display
	imageLayout := container.New(
		layout.NewHBoxLayout(),
		originalContainer,
		previewContainer,
	)

	// Create scroll container for horizontal and vertical scrolling
	scrollContainer := container.NewScroll(imageLayout)
	scrollContainer.SetMinSize(fyne.NewSize(ScrollViewportWidth, ScrollViewportHeight))

	// Wrap scroll container in border layout for expansion behavior
	mainContainer := container.NewBorder(
		nil, nil, nil, nil,
		scrollContainer,
	)

	return &ImageDisplay{
		container:       mainContainer,
		originalImage:   originalImage,
		previewImage:    previewImage,
		scrollContainer: scrollContainer,
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

	// Update image size based on actual image dimensions
	bounds := img.Bounds()
	actualWidth := float32(bounds.Dx())
	actualHeight := float32(bounds.Dy())

	// Set minimum size to actual image size to enable scrolling
	id.originalImage.SetMinSize(fyne.NewSize(actualWidth, actualHeight))
	id.originalImage.Refresh()
}

func (id *ImageDisplay) SetPreviewImage(img image.Image) {
	if img == nil {
		return
	}

	id.previewImage.Image = img

	// Update image size based on actual image dimensions
	bounds := img.Bounds()
	actualWidth := float32(bounds.Dx())
	actualHeight := float32(bounds.Dy())

	// Set minimum size to actual image size to enable scrolling
	id.previewImage.SetMinSize(fyne.NewSize(actualWidth, actualHeight))
	id.previewImage.Refresh()
}
