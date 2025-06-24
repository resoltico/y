package gui

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"otsu-obliterator/pipeline"
)

// ImageDisplay handles the top section with Original and Preview images
type ImageDisplay struct {
	container     *fyne.Container
	originalImage *canvas.Image
	previewImage  *canvas.Image
}

const (
	ImageDisplayWidth  = 640
	ImageDisplayHeight = 480
)

func NewImageDisplay() *ImageDisplay {
	display := &ImageDisplay{}
	display.setupImages()
	return display
}

func (id *ImageDisplay) setupImages() {
	// Create image canvases with fixed sizes
	id.originalImage = canvas.NewImageFromImage(nil)
	id.originalImage.FillMode = canvas.ImageFillContain
	id.originalImage.SetMinSize(fyne.NewSize(ImageDisplayWidth, ImageDisplayHeight))

	id.previewImage = canvas.NewImageFromImage(nil)
	id.previewImage.FillMode = canvas.ImageFillContain
	id.previewImage.SetMinSize(fyne.NewSize(ImageDisplayWidth, ImageDisplayHeight))

	// Create containers with labels for each image
	originalContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Original**"),
		id.originalImage,
	)

	previewContainer := container.NewVBox(
		widget.NewRichTextFromMarkdown("**Preview**"),
		id.previewImage,
	)

	// Create horizontal split with fixed 50/50 ratio
	imageSplit := container.NewHSplit(originalContainer, previewContainer)
	imageSplit.SetOffset(0.5) // Fixed 50/50 split

	// Wrap in container to enforce size constraints
	id.container = container.NewBorder(
		nil, nil, nil, nil, // no borders
		imageSplit, // center content
	)
}

func (id *ImageDisplay) GetContainer() *fyne.Container {
	return id.container
}

func (id *ImageDisplay) SetOriginalImage(imageData interface{}) {
	if imageData == nil {
		return
	}

	// Type assertion to get the actual image
	var img image.Image
	switch data := imageData.(type) {
	case *pipeline.ImageData:
		if data.Image != nil {
			img = data.Image
		}
	case image.Image:
		img = data
	default:
		return
	}

	if img == nil {
		return
	}

	// Update the canvas image using thread-safe Do
	fyne.Do(func() {
		id.originalImage.Image = img
		id.originalImage.Refresh()
	})
}

func (id *ImageDisplay) SetPreviewImage(imageData interface{}) {
	if imageData == nil {
		return
	}

	// Type assertion to get the actual image
	var img image.Image
	switch data := imageData.(type) {
	case *pipeline.ImageData:
		if data.Image != nil {
			img = data.Image
		}
	case image.Image:
		img = data
	default:
		return
	}

	if img == nil {
		return
	}

	// Update the canvas image using thread-safe Do
	fyne.Do(func() {
		id.previewImage.Image = img
		id.previewImage.Refresh()
	})
}
