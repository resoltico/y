package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// HiddenSplit creates an HSplit container without visible divider
type HiddenSplit struct {
	widget.BaseWidget
	leading  fyne.CanvasObject
	trailing fyne.CanvasObject
	offset   float64
}

// NewHiddenSplit creates a new HSplit without visible divider
func NewHiddenSplit(leading, trailing fyne.CanvasObject) *HiddenSplit {
	split := &HiddenSplit{
		leading:  leading,
		trailing: trailing,
		offset:   0.5,
	}
	split.ExtendBaseWidget(split)
	return split
}

// SetOffset sets the split position (0.0 to 1.0)
func (h *HiddenSplit) SetOffset(offset float64) {
	h.offset = offset
	h.Refresh()
}

// GetOffset returns the current split position
func (h *HiddenSplit) GetOffset() float64 {
	return h.offset
}

// CreateRenderer creates the renderer for HiddenSplit
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
