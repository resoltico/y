package layout

import (
	"fyne.io/fyne/v2"
)

// FixedColumnLayout maintains consistent column widths regardless of content changes
type FixedColumnLayout struct {
	columnWidths []float32
	padding      float32
}

func NewFixedColumnLayout(columnWidths []float32, padding float32) *FixedColumnLayout {
	return &FixedColumnLayout{
		columnWidths: columnWidths,
		padding:      padding,
	}
}

func (fcl *FixedColumnLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) == 0 {
		return
	}

	x := float32(0)
	for i, obj := range objects {
		if i >= len(fcl.columnWidths) {
			break
		}

		width := fcl.columnWidths[i]
		obj.Resize(fyne.NewSize(width-fcl.padding, containerSize.Height))
		obj.Move(fyne.NewPos(x, 0))
		x += width
	}
}

func (fcl *FixedColumnLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	totalWidth := float32(0)
	maxHeight := float32(0)

	for i, width := range fcl.columnWidths {
		totalWidth += width
		
		if i < len(objects) {
			objMin := objects[i].MinSize()
			if objMin.Height > maxHeight {
				maxHeight = objMin.Height
			}
		}
	}

	return fyne.NewSize(totalWidth, maxHeight)
}

// StableParameterLayout prevents UI shifts when parameter content changes
type StableParameterLayout struct {
	sectionsHeight map[string]float32
	sectionWidth   float32
	padding        float32
}

func NewStableParameterLayout(sectionWidth float32, padding float32) *StableParameterLayout {
	return &StableParameterLayout{
		sectionsHeight: make(map[string]float32),
		sectionWidth:   sectionWidth,
		padding:        padding,
	}
}

func (spl *StableParameterLayout) RegisterSection(name string, height float32) {
	spl.sectionsHeight[name] = height
}

func (spl *StableParameterLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	if len(objects) == 0 {
		return
	}

	y := float32(0)
	for i, obj := range objects {
		sectionName := spl.getSectionName(i)
		height := spl.sectionsHeight[sectionName]
		
		if height == 0 {
			height = obj.MinSize().Height
		}

		obj.Resize(fyne.NewSize(spl.sectionWidth-spl.padding, height))
		obj.Move(fyne.NewPos(0, y))
		y += height + spl.padding
	}
}

func (spl *StableParameterLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	totalHeight := float32(0)
	
	for i := range objects {
		sectionName := spl.getSectionName(i)
		height := spl.sectionsHeight[sectionName]
		
		if height == 0 && i < len(objects) {
			height = objects[i].MinSize().Height
		}
		
		totalHeight += height + spl.padding
	}

	return fyne.NewSize(spl.sectionWidth, totalHeight)
}

func (spl *StableParameterLayout) getSectionName(index int) string {
	switch index {
	case 0:
		return "quality"
	case 1:
		return "parameters"
	default:
		return "unknown"
	}
}
