package otsu

import (
	"gocv.io/x/gocv"
)

type TriclassSegmenter struct {
	params        map[string]interface{}
	memoryManager MemoryManagerInterface
}

func NewTriclassSegmenter(params map[string]interface{}, memoryManager MemoryManagerInterface) *TriclassSegmenter {
	return &TriclassSegmenter{
		params:        params,
		memoryManager: memoryManager,
	}
}

func (segmenter *TriclassSegmenter) SegmentRegion(region *gocv.Mat, threshold float64) (gocv.Mat, gocv.Mat, gocv.Mat) {
	rows := region.Rows()
	cols := region.Cols()

	foreground := segmenter.memoryManager.GetMat(rows, cols, gocv.MatTypeCV8UC1)
	background := segmenter.memoryManager.GetMat(rows, cols, gocv.MatTypeCV8UC1)
	tbd := segmenter.memoryManager.GetMat(rows, cols, gocv.MatTypeCV8UC1)

	// Initialize all masks to 0
	foreground.SetTo(gocv.NewScalar(0, 0, 0, 0))
	background.SetTo(gocv.NewScalar(0, 0, 0, 0))
	tbd.SetTo(gocv.NewScalar(0, 0, 0, 0))

	gapFactor := segmenter.getFloatParam("lower_upper_gap_factor")

	// Create adaptive thresholds
	lowerThreshold := threshold * (1.0 - gapFactor)
	upperThreshold := threshold * (1.0 + gapFactor)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue := float64(region.GetUCharAt(y, x))

			// Only process active pixels
			if pixelValue > 0 {
				if pixelValue > upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if pixelValue < lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd
}

func (segmenter *TriclassSegmenter) getFloatParam(name string) float64 {
	if value, ok := segmenter.params[name].(float64); ok {
		return value
	}
	return 0.0
}
