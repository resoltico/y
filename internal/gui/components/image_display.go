package components

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	ImageConstraintWidth  = 640
	ImageConstraintHeight = 480
	MinViewportWidth      = 400
	MinViewportHeight     = 300
)

type ImageDisplay struct {
	container      *fyne.Container
	originalImage  *canvas.Image
	previewImage   *canvas.Image
	originalScroll *container.Scroll
	previewScroll  *container.Scroll
}

func NewImageDisplay() *ImageDisplay {
	// Create constrained image displays
	originalImage := canvas.NewImageFromImage(nil)
	originalImage.FillMode = canvas.ImageFillContain
	originalImage.ScaleMode = canvas.ImageScaleSmooth
	originalImage.SetMinSize(fyne.NewSize(ImageConstraintWidth, ImageConstraintHeight))

	previewImage := canvas.NewImageFromImage(nil)
	previewImage.FillMode = canvas.ImageFillContain
	previewImage.ScaleMode = canvas.ImageScaleSmooth
	previewImage.SetMinSize(fyne.NewSize(ImageConstraintWidth, ImageConstraintHeight))

	// Create scrollable containers for each image
	originalScroll := container.NewScroll(originalImage)
	originalScroll.SetMinSize(fyne.NewSize(MinViewportWidth, MinViewportHeight))

	previewScroll := container.NewScroll(previewImage)
	previewScroll.SetMinSize(fyne.NewSize(MinViewportWidth, MinViewportHeight))

	// Create labeled containers
	originalContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		originalScroll,
	)

	previewContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		previewScroll,
	)

	// Create horizontal split layout for dual-pane display
	splitContainer := container.NewHSplit(originalContainer, previewContainer)
	splitContainer.SetOffset(0.5) // Equal split

	// Wrap split container in a regular container
	mainContainer := container.NewWithoutLayout(splitContainer)
	mainContainer.Add(splitContainer)
	splitContainer.Resize(fyne.NewSize(ImageConstraintWidth*2, ImageConstraintHeight+60))

	return &ImageDisplay{
		container:      mainContainer,
		originalImage:  originalImage,
		previewImage:   previewImage,
		originalScroll: originalScroll,
		previewScroll:  previewScroll,
	}
}

func (id *ImageDisplay) GetContainer() *fyne.Container {
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

func (id *ImageDisplay) GetRequiredWindowSize(leftPanelWidth, rightPanelWidth float32) fyne.Size {
	// Calculate total window size needed for 640x480 image areas
	totalImageWidth := float32(ImageConstraintWidth * 2)
	totalPanelWidth := leftPanelWidth + rightPanelWidth
	totalImageHeight := float32(ImageConstraintHeight)

	// Add space for labels and padding
	labelHeight := float32(30)
	padding := float32(20)

	return fyne.Size{
		Width:  totalImageWidth + totalPanelWidth + padding,
		Height: totalImageHeight + labelHeight + padding,
	}
}
