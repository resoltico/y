package widgets

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
	splitView     *container.Split
}

func NewImageDisplay() *ImageDisplay {
	display := &ImageDisplay{}
	display.createComponents()
	display.setupLayout()
	return display
}

func (id *ImageDisplay) createComponents() {
	// Original image canvas
	id.originalImage = canvas.NewImageFromImage(nil)
	id.originalImage.FillMode = canvas.ImageFillContain
	id.originalImage.ScaleMode = canvas.ImageScaleSmooth
	id.originalImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))

	// Preview image canvas
	id.previewImage = canvas.NewImageFromImage(nil)
	id.previewImage.FillMode = canvas.ImageFillContain
	id.previewImage.ScaleMode = canvas.ImageScaleSmooth
	id.previewImage.SetMinSize(fyne.NewSize(ImageAreaWidth, ImageAreaHeight))
}

func (id *ImageDisplay) setupLayout() {
	// Original image container with label
	originalContainer := container.NewBorder(
		widget.NewRichTextFromMarkdown("**Original**"),
		nil, nil, nil,
		id.originalImage,
	)

	// Preview image container with label
	previewContainer := container.NewBorder(
		widget.NewRichTextFromMarkdown("**Preview**"),
		nil, nil, nil,
		id.previewImage,
	)

	// Create split layout using standard HSplit
	id.splitView = container.NewHSplit(originalContainer, previewContainer)
	id.splitView.SetOffset(0.5) // Equal split
	id.container = id.splitView
}

func (id *ImageDisplay) GetContainer() fyne.CanvasObject {
	return id.container
}

func (id *ImageDisplay) SetOriginalImage(img image.Image) {
	if img == nil {
		id.originalImage.Image = nil
	} else {
		id.originalImage.Image = img
	}
	id.originalImage.Refresh()
}

func (id *ImageDisplay) SetPreviewImage(img image.Image) {
	if img == nil {
		id.previewImage.Image = nil
	} else {
		id.previewImage.Image = img
	}
	id.previewImage.Refresh()
}

func (id *ImageDisplay) GetSplitView() *container.Split {
	return id.splitView
}

// HiddenSplit creates an HSplit container without visible divider
type HiddenSplit struct {
	widget.BaseWidget
	leading  fyne.CanvasObject
	trailing fyne.CanvasObject
	offset   float64
}

func NewHiddenSplit(leading, trailing fyne.CanvasObject) *HiddenSplit {
	split := &HiddenSplit{
		leading:  leading,
		trailing: trailing,
		offset:   0.5,
	}
	split.ExtendBaseWidget(split)
	return split
}

func (h *HiddenSplit) SetOffset(offset float64) {
	h.offset = offset
	h.Refresh()
}

func (h *HiddenSplit) GetOffset() float64 {
	return h.offset
}

func (h *HiddenSplit) CreateRenderer() fyne.WidgetRenderer {
	return &hiddenSplitRenderer{
		split:   h,
		objects: []fyne.CanvasObject{h.leading, h.trailing},
	}
}

type hiddenSplitRenderer struct {
	split   *HiddenSplit
	objects []fyne.CanvasObject
}

func (r *hiddenSplitRenderer) Layout(size fyne.Size) {
	leadingWidth := size.Width * float32(r.split.offset)
	trailingWidth := size.Width - leadingWidth

	r.split.leading.Resize(fyne.NewSize(leadingWidth, size.Height))
	r.split.leading.Move(fyne.NewPos(0, 0))

	r.split.trailing.Resize(fyne.NewSize(trailingWidth, size.Height))
	r.split.trailing.Move(fyne.NewPos(leadingWidth, 0))
}

func (r *hiddenSplitRenderer) MinSize() fyne.Size {
	leadingMin := r.split.leading.MinSize()
	trailingMin := r.split.trailing.MinSize()

	width := leadingMin.Width + trailingMin.Width
	height := fyne.Max(leadingMin.Height, trailingMin.Height)

	return fyne.NewSize(width, height)
}

func (r *hiddenSplitRenderer) Refresh() {
	r.Layout(r.split.Size())
	for _, obj := range r.objects {
		obj.Refresh()
	}
}

func (r *hiddenSplitRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *hiddenSplitRenderer) Destroy() {
	// Cleanup if needed
}
