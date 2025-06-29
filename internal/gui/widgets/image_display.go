package widgets

import (
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	ImageAreaWidth  = 600
	ImageAreaHeight = 450
)

type ImageDisplay struct {
	container     *fyne.Container
	originalImage *canvas.Image
	previewImage  *canvas.Image
	splitView     *container.Split

	// Placeholder images for empty states
	originalPlaceholder *canvas.Image
	previewPlaceholder  *canvas.Image
}

func NewImageDisplay() *ImageDisplay {
	display := &ImageDisplay{}
	display.createComponents()
	display.setupLayout()
	return display
}

func (id *ImageDisplay) createComponents() {
	// Create placeholder images
	id.originalPlaceholder = id.createPlaceholderImage("Load an image to begin")
	id.previewPlaceholder = id.createPlaceholderImage("Processed result will appear here")

	// Create image canvases with modern styling
	id.originalImage = canvas.NewImageFromImage(nil)
	id.originalImage.FillMode = canvas.ImageFillContain
	id.originalImage.ScaleMode = canvas.ImageScaleSmooth
	id.originalImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))

	id.previewImage = canvas.NewImageFromImage(nil)
	id.previewImage.FillMode = canvas.ImageFillContain
	id.previewImage.ScaleMode = canvas.ImageScaleSmooth
	id.previewImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))

	// Start with placeholders
	id.originalImage.Image = id.originalPlaceholder.Image
	id.previewImage.Image = id.previewPlaceholder.Image
}

func (id *ImageDisplay) createPlaceholderImage(text string) *canvas.Image {
	// Create a simple placeholder image with text
	img := image.NewRGBA(image.Rect(0, 0, ImageAreaWidth, ImageAreaHeight))

	// Fill with light gray background
	lightGray := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	for y := 0; y < ImageAreaHeight; y++ {
		for x := 0; x < ImageAreaWidth; x++ {
			img.Set(x, y, lightGray)
		}
	}

	// Add a border
	borderColor := color.RGBA{R: 200, G: 200, B: 200, A: 255}
	for x := 0; x < ImageAreaWidth; x++ {
		img.Set(x, 0, borderColor)
		img.Set(x, ImageAreaHeight-1, borderColor)
	}
	for y := 0; y < ImageAreaHeight; y++ {
		img.Set(0, y, borderColor)
		img.Set(ImageAreaWidth-1, y, borderColor)
	}

	canvasImg := canvas.NewImageFromImage(img)
	canvasImg.FillMode = canvas.ImageFillContain
	canvasImg.ScaleMode = canvas.ImageScaleSmooth

	return canvasImg
}

func (id *ImageDisplay) setupLayout() {
	// Create containers with headers and modern styling
	originalContainer := container.NewBorder(
		container.NewHBox(
			widget.NewIcon(nil), // Placeholder for icon
			widget.NewRichTextFromMarkdown("**Original Image**"),
		),
		nil, nil, nil,
		container.NewStack(
			id.createImageBackground(),
			id.originalImage,
		),
	)

	previewContainer := container.NewBorder(
		container.NewHBox(
			widget.NewIcon(nil), // Placeholder for icon
			widget.NewRichTextFromMarkdown("**Processed Result**"),
		),
		nil, nil, nil,
		container.NewStack(
			id.createImageBackground(),
			id.previewImage,
		),
	)

	// Create split view with responsive design
	id.splitView = container.NewHSplit(originalContainer, previewContainer)
	id.splitView.SetOffset(0.5) // Equal split initially

	id.container = id.splitView
}

func (id *ImageDisplay) createImageBackground() *canvas.Rectangle {
	// Create subtle background for image areas
	bg := canvas.NewRectangle(color.RGBA{R: 252, G: 252, B: 252, A: 255})
	return bg
}

func (id *ImageDisplay) GetContainer() fyne.CanvasObject {
	return id.container
}

func (id *ImageDisplay) SetOriginalImage(img image.Image) {
	// Use fyne.Do for thread-safe UI updates in Fyne v2.6+
	fyne.Do(func() {
		if img != nil {
			id.originalImage.Image = img
		} else {
			id.originalImage.Image = id.originalPlaceholder.Image
		}
		id.originalImage.Refresh()
		id.container.Refresh()
	})
}

func (id *ImageDisplay) SetPreviewImage(img image.Image) {
	fyne.Do(func() {
		if img != nil {
			id.previewImage.Image = img
		} else {
			id.previewImage.Image = id.previewPlaceholder.Image
		}
		id.previewImage.Refresh()
		id.container.Refresh()
	})
}

func (id *ImageDisplay) GetSplitView() *container.Split {
	return id.splitView
}

// Utility methods for enhanced functionality
func (id *ImageDisplay) SetSplitRatio(ratio float64) {
	fyne.Do(func() {
		if ratio >= 0.1 && ratio <= 0.9 {
			id.splitView.SetOffset(ratio)
		}
	})
}

func (id *ImageDisplay) GetOriginalImageSize() (int, int) {
	if id.originalImage.Image == nil {
		return 0, 0
	}
	bounds := id.originalImage.Image.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func (id *ImageDisplay) GetPreviewImageSize() (int, int) {
	if id.previewImage.Image == nil {
		return 0, 0
	}
	bounds := id.previewImage.Image.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func (id *ImageDisplay) ClearImages() {
	fyne.Do(func() {
		id.SetOriginalImage(nil)
		id.SetPreviewImage(nil)
	})
}

func (id *ImageDisplay) HasOriginalImage() bool {
	return id.originalImage.Image != nil && id.originalImage.Image != id.originalPlaceholder.Image
}

func (id *ImageDisplay) HasPreviewImage() bool {
	return id.previewImage.Image != nil && id.previewImage.Image != id.previewPlaceholder.Image
}
