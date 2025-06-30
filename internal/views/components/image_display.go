package components

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

// ImageDisplay handles the display of original and processed images
type ImageDisplay struct {
	container      *fyne.Container
	originalImage  *canvas.Image
	processedImage *canvas.Image
	splitView      *container.Split
	
	// Placeholder images
	originalPlaceholder  *canvas.Image
	processedPlaceholder *canvas.Image
	
	// State
	hasOriginal  bool
	hasProcessed bool
}

// NewImageDisplay creates a new image display component
func NewImageDisplay() *ImageDisplay {
	display := &ImageDisplay{}
	display.createComponents()
	display.setupLayout()
	return display
}

// createComponents initializes the image display components
func (id *ImageDisplay) createComponents() {
	// Create placeholder images
	id.originalPlaceholder = id.createPlaceholderImage("Load an image to begin")
	id.processedPlaceholder = id.createPlaceholderImage("Processed result will appear here")
	
	// Create image canvases with modern Fyne v2.6+ settings
	id.originalImage = canvas.NewImageFromImage(id.originalPlaceholder.Image)
	id.originalImage.FillMode = canvas.ImageFillContain
	id.originalImage.ScaleMode = canvas.ImageScaleSmooth
	id.originalImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))
	
	id.processedImage = canvas.NewImageFromImage(id.processedPlaceholder.Image)
	id.processedImage.FillMode = canvas.ImageFillContain
	id.processedImage.ScaleMode = canvas.ImageScaleSmooth
	id.processedImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))
}

// createPlaceholderImage creates a placeholder image with text
func (id *ImageDisplay) createPlaceholderImage(text string) *canvas.Image {
	// Create simple placeholder with border
	img := image.NewRGBA(image.Rect(0, 0, ImageAreaWidth, ImageAreaHeight))
	
	// Fill with light gray background
	lightGray := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	for y := 0; y < ImageAreaHeight; y++ {
		for x := 0; x < ImageAreaWidth; x++ {
			img.Set(x, y, lightGray)
		}
	}
	
	// Add border
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

// setupLayout creates the split view layout
func (id *ImageDisplay) setupLayout() {
	// Create containers with headers
	originalContainer := container.NewBorder(
		container.NewHBox(
			widget.NewRichTextFromMarkdown("**Original Image**"),
		),
		nil, nil, nil,
		container.NewStack(
			id.createImageBackground(),
			id.originalImage,
		),
	)
	
	processedContainer := container.NewBorder(
		container.NewHBox(
			widget.NewRichTextFromMarkdown("**Processed Result**"),
		),
		nil, nil, nil,
		container.NewStack(
			id.createImageBackground(),
			id.processedImage,
		),
	)
	
	// Create split view
	id.splitView = container.NewHSplit(originalContainer, processedContainer)
	id.splitView.SetOffset(0.5) // Equal split
	
	id.container = id.splitView
}

// createImageBackground creates background for image areas
func (id *ImageDisplay) createImageBackground() *canvas.Rectangle {
	bg := canvas.NewRectangle(color.RGBA{R: 252, G: 252, B: 252, A: 255})
	return bg
}

// SetOriginalImage updates the original image display
func (id *ImageDisplay) SetOriginalImage(img image.Image) {
	fyne.Do(func() {
		if img != nil {
			id.originalImage.Image = img
			id.hasOriginal = true
		} else {
			id.originalImage.Image = id.originalPlaceholder.Image
			id.hasOriginal = false
		}
		id.originalImage.Refresh()
		id.container.Refresh()
	})
}

// SetProcessedImage updates the processed image display
func (id *ImageDisplay) SetProcessedImage(img image.Image) {
	fyne.Do(func() {
		if img != nil {
			id.processedImage.Image = img
			id.hasProcessed = true
		} else {
			id.processedImage.Image = id.processedPlaceholder.Image
			id.hasProcessed = false
		}
		id.processedImage.Refresh()
		id.container.Refresh()
	})
}

// HasOriginalImage returns true if original image is loaded
func (id *ImageDisplay) HasOriginalImage() bool {
	return id.hasOriginal
}

// HasProcessedImage returns true if processed image is available
func (id *ImageDisplay) HasProcessedImage() bool {
	return id.hasProcessed
}

// ClearImages clears both images
func (id *ImageDisplay) ClearImages() {
	fyne.Do(func() {
		id.SetOriginalImage(nil)
		id.SetProcessedImage(nil)
	})
}

// SetSplitRatio adjusts the split ratio between images
func (id *ImageDisplay) SetSplitRatio(ratio float64) {
	fyne.Do(func() {
		if ratio >= 0.1 && ratio <= 0.9 {
			id.splitView.SetOffset(ratio)
		}
	})
}

// GetOriginalImageSize returns the dimensions of the original image
func (id *ImageDisplay) GetOriginalImageSize() (int, int) {
	if !id.hasOriginal || id.originalImage.Image == nil {
		return 0, 0
	}
	bounds := id.originalImage.Image.Bounds()
	return bounds.Dx(), bounds.Dy()
}

// GetProcessedImageSize returns the dimensions of the processed image
func (id *ImageDisplay) GetProcessedImageSize() (int, int) {
	if !id.hasProcessed || id.processedImage.Image == nil {
		return 0, 0
	}
	bounds := id.processedImage.Image.Bounds()
	return bounds.Dx(), bounds.Dy()
}

// GetContainer returns the main container
func (id *ImageDisplay) GetContainer() *fyne.Container {
	return id.container
}

// GetSplitView returns the split view container
func (id *ImageDisplay) GetSplitView() *container.Split {
	return id.splitView
}

// Refresh refreshes the image display
func (id *ImageDisplay) Refresh() {
	fyne.Do(func() {
		id.container.Refresh()
	})
}