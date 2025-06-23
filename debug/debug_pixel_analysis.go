package debug

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

// Global debug toggle for pixel-level analysis
var EnablePixelAnalysisDebug = false

type PixelAnalysisInfo struct {
	ImageType       string
	Dimensions      string
	SampleLocations []string
	PixelValues     []string
	ColorModel      string
	IsAllBlack      bool
	IsAllWhite      bool
	HasMixedValues  bool
}

func (dm *Manager) LogPixelAnalysis(analysisType string, img image.Image) {
	if !EnablePixelAnalysisDebug {
		return
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	bounds := img.Bounds()
	info := &PixelAnalysisInfo{
		ImageType:       fmt.Sprintf("%T", img),
		Dimensions:      fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy()),
		ColorModel:      fmt.Sprintf("%T", img.ColorModel()),
		SampleLocations: []string{},
		PixelValues:     []string{},
	}

	// Sample pixels from different regions
	width := bounds.Dx()
	height := bounds.Dy()

	samplePoints := []image.Point{
		{0, 0},                  // Top-left
		{width / 2, 0},          // Top-center
		{width - 1, 0},          // Top-right
		{0, height / 2},         // Middle-left
		{width / 2, height / 2}, // Center
		{width - 1, height / 2}, // Middle-right
		{0, height - 1},         // Bottom-left
		{width / 2, height - 1}, // Bottom-center
		{width - 1, height - 1}, // Bottom-right
	}

	blackCount := 0
	whiteCount := 0
	totalSamples := 0

	for _, point := range samplePoints {
		if point.X < width && point.Y < height {
			pixelColor := img.At(point.X, point.Y)

			location := fmt.Sprintf("(%d,%d)", point.X, point.Y)
			info.SampleLocations = append(info.SampleLocations, location)

			var pixelDesc string

			switch c := pixelColor.(type) {
			case color.Gray:
				pixelDesc = fmt.Sprintf("Gray{%d}", c.Y)
				if c.Y == 0 {
					blackCount++
				} else if c.Y == 255 {
					whiteCount++
				}
			case color.RGBA:
				pixelDesc = fmt.Sprintf("RGBA{%d,%d,%d,%d}", c.R, c.G, c.B, c.A)
				if c.R == 0 && c.G == 0 && c.B == 0 {
					blackCount++
				} else if c.R == 255 && c.G == 255 && c.B == 255 {
					whiteCount++
				}
			case color.NRGBA:
				pixelDesc = fmt.Sprintf("NRGBA{%d,%d,%d,%d}", c.R, c.G, c.B, c.A)
				if c.R == 0 && c.G == 0 && c.B == 0 {
					blackCount++
				} else if c.R == 255 && c.G == 255 && c.B == 255 {
					whiteCount++
				}
			default:
				r, g, b, a := pixelColor.RGBA()
				pixelDesc = fmt.Sprintf("Generic{%d,%d,%d,%d}", r>>8, g>>8, b>>8, a>>8)
				if r == 0 && g == 0 && b == 0 {
					blackCount++
				} else if r == 0xFFFF && g == 0xFFFF && b == 0xFFFF {
					whiteCount++
				}
			}

			info.PixelValues = append(info.PixelValues, pixelDesc)
			totalSamples++
		}
	}

	info.IsAllBlack = blackCount == totalSamples
	info.IsAllWhite = whiteCount == totalSamples
	info.HasMixedValues = blackCount > 0 && blackCount < totalSamples

	report := fmt.Sprintf(`Pixel Analysis Report (%s):
- Image Type: %s
- Dimensions: %s
- Color Model: %s
- Sample Locations: %v
- Pixel Values: %v
- All Black: %t (%d/%d samples)
- All White: %t (%d/%d samples)
- Mixed Values: %t`,
		analysisType, info.ImageType, info.Dimensions, info.ColorModel,
		info.SampleLocations, info.PixelValues,
		info.IsAllBlack, blackCount, totalSamples,
		info.IsAllWhite, whiteCount, totalSamples,
		info.HasMixedValues)

	LogInfo("PixelAnalysis", report)

	if info.IsAllBlack {
		LogWarning("PixelAnalysis", fmt.Sprintf("%s: Image appears to be completely black!", analysisType))
	}
}

func (dm *Manager) LogMatPixelAnalysis(analysisType string, mat gocv.Mat) {
	if !EnablePixelAnalysisDebug {
		return
	}

	// Essential safety checks first
	if mat.Empty() {
		LogWarning("PixelAnalysis", fmt.Sprintf("%s: Mat is empty - skipping pixel analysis", analysisType))
		return
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	if rows <= 0 || cols <= 0 {
		LogWarning("PixelAnalysis", fmt.Sprintf("%s: Mat has invalid dimensions (%dx%d) - skipping pixel analysis",
			analysisType, cols, rows))
		return
	}

	// Additional validation - check Mat type
	matType := mat.Type()
	if matType < 0 {
		LogWarning("PixelAnalysis", fmt.Sprintf("%s: Mat has invalid type (%d) - skipping pixel analysis",
			analysisType, matType))
		return
	}

	// Use recovery for ANY Mat operation
	defer func() {
		if r := recover(); r != nil {
			LogWarning("PixelAnalysis", fmt.Sprintf("%s: Mat operation caused panic (recovered): %v", analysisType, r))
		}
	}()

	// Validate channels before any operations
	if channels != 1 && channels != 3 && channels != 4 {
		LogWarning("PixelAnalysis", fmt.Sprintf("%s: Unsupported channel count (%d) - skipping pixel analysis",
			analysisType, channels))
		return
	}

	// Log basic Mat info without pixel access to avoid corruption issues
	LogInfo("PixelAnalysis", fmt.Sprintf("Mat Info (%s): %dx%d, %d channels, type %d - basic validation passed",
		analysisType, cols, rows, channels, matType))

	// For safety, completely skip pixel sampling when Mat data might be corrupted
	// This prevents segfaults while maintaining debug info about Mat structure
	LogInfo("PixelAnalysis", fmt.Sprintf("%s: Skipping pixel sampling for memory safety", analysisType))
}
