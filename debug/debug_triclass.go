package debug

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

// Global debug toggle for triclass algorithm (set from main package)
var EnableTriclassDebug = false

type TriclassDebugInfo struct {
	InputMatDimensions   string
	InputMatChannels     int
	InputMatType         gocv.MatType
	OutputMatDimensions  string
	OutputMatChannels    int
	OutputMatType        gocv.MatType
	IterationCount       int
	FinalThreshold       float64
	TotalPixels          int
	ForegroundPixels     int
	BackgroundPixels     int
	TBDPixels            int
	ProcessingSteps      []string
	IterationThresholds  []float64
	IterationConvergence []float64
}

func (dm *Manager) LogTriclassStart(inputMat gocv.Mat, params map[string]interface{}) {
	if !EnableTriclassDebug {
		return
	}

	LogInfo("TriclassDebug", fmt.Sprintf("Starting Iterative Triclass - Input: %dx%d, Channels: %d, Type: %d, Params: %+v",
		inputMat.Cols(), inputMat.Rows(), inputMat.Channels(), int(inputMat.Type()), params))
}

func (dm *Manager) LogTriclassIteration(iteration int, threshold float64, convergence float64,
	foregroundCount, backgroundCount, tbdCount int) {
	if !EnableTriclassDebug {
		return
	}

	LogInfo("TriclassDebug", fmt.Sprintf("Iteration %d - Threshold: %.2f, Convergence: %.4f, FG: %d, BG: %d, TBD: %d",
		iteration, threshold, convergence, foregroundCount, backgroundCount, tbdCount))
}

func (dm *Manager) LogTriclassResult(info *TriclassDebugInfo) {
	if !EnableTriclassDebug {
		return
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	report := fmt.Sprintf(`Iterative Triclass Debug Report:
- Input Mat: %s, Channels: %d, Type: %d
- Output Mat: %s, Channels: %d, Type: %d
- Iterations: %d
- Final Threshold: %.2f
- Total Pixels: %d
- Foreground Pixels: %d (%.2f%%)
- Background Pixels: %d (%.2f%%)
- TBD Pixels: %d (%.2f%%)
- Processing Steps: %v
- Iteration Thresholds: %v
- Iteration Convergence: %v`,
		info.InputMatDimensions, info.InputMatChannels, int(info.InputMatType),
		info.OutputMatDimensions, info.OutputMatChannels, int(info.OutputMatType),
		info.IterationCount, info.FinalThreshold, info.TotalPixels,
		info.ForegroundPixels, float64(info.ForegroundPixels)/float64(info.TotalPixels)*100,
		info.BackgroundPixels, float64(info.BackgroundPixels)/float64(info.TotalPixels)*100,
		info.TBDPixels, float64(info.TBDPixels)/float64(info.TotalPixels)*100,
		info.ProcessingSteps, info.IterationThresholds, info.IterationConvergence)

	LogInfo("TriclassDebug", report)
}

func (dm *Manager) LogMatPixelSample(matName string, mat gocv.Mat, sampleSize int) {
	if !EnableTriclassDebug {
		return
	}

	if mat.Empty() {
		LogWarning("TriclassDebug", fmt.Sprintf("%s Mat is empty", matName))
		return
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	LogInfo("TriclassDebug", fmt.Sprintf("%s Mat Info - Size: %dx%d, Channels: %d, Type: %d",
		matName, cols, rows, channels, int(mat.Type())))

	// Sample pixels from different regions
	samples := []string{}
	stepY := rows / sampleSize
	stepX := cols / sampleSize

	if stepY == 0 {
		stepY = 1
	}
	if stepX == 0 {
		stepX = 1
	}

	for y := 0; y < rows && len(samples) < sampleSize*sampleSize; y += stepY {
		for x := 0; x < cols && len(samples) < sampleSize*sampleSize; x += stepX {
			var pixelValue string

			if channels == 1 {
				value := mat.GetUCharAt(y, x)
				pixelValue = fmt.Sprintf("(%d,%d)=%d", x, y, value)
			} else if channels == 3 {
				b := mat.GetUCharAt3(y, x, 0)
				g := mat.GetUCharAt3(y, x, 1)
				r := mat.GetUCharAt3(y, x, 2)
				pixelValue = fmt.Sprintf("(%d,%d)=(%d,%d,%d)", x, y, r, g, b)
			} else {
				pixelValue = fmt.Sprintf("(%d,%d)=unsupported", x, y)
			}

			samples = append(samples, pixelValue)
		}
	}

	LogInfo("TriclassDebug", fmt.Sprintf("%s Pixel Samples: %v", matName, samples))
}

func (dm *Manager) LogMatStatistics(matName string, mat gocv.Mat) {
	if !EnableTriclassDebug {
		return
	}

	if mat.Empty() {
		LogWarning("TriclassDebug", fmt.Sprintf("%s Mat is empty", matName))
		return
	}

	// Calculate basic statistics
	rows := mat.Rows()
	cols := mat.Cols()
	totalPixels := rows * cols

	if mat.Channels() == 1 {
		// For grayscale, count different values
		histogram := make(map[uint8]int)
		var minVal, maxVal uint8 = 255, 0
		var sum uint64 = 0

		for y := 0; y < rows; y++ {
			for x := 0; x < cols; x++ {
				value := mat.GetUCharAt(y, x)
				histogram[value]++
				sum += uint64(value)
				if value < minVal {
					minVal = value
				}
				if value > maxVal {
					maxVal = value
				}
			}
		}

		avg := float64(sum) / float64(totalPixels)
		uniqueValues := len(histogram)

		LogInfo("TriclassDebug", fmt.Sprintf("%s Statistics - Min: %d, Max: %d, Avg: %.2f, Unique Values: %d, Total Pixels: %d",
			matName, minVal, maxVal, avg, uniqueValues, totalPixels))

		// Log histogram for binary images
		if uniqueValues <= 10 {
			LogInfo("TriclassDebug", fmt.Sprintf("%s Histogram: %v", matName, histogram))
		}
	}
}

func (dm *Manager) LogImageConversionDebug(fromMat gocv.Mat, toImage image.Image, conversionType string) {
	if !EnableTriclassDebug {
		return
	}

	matInfo := fmt.Sprintf("%dx%d, %d channels, type %d",
		fromMat.Cols(), fromMat.Rows(), fromMat.Channels(), int(fromMat.Type()))

	bounds := toImage.Bounds()
	imageInfo := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())

	LogInfo("TriclassDebug", fmt.Sprintf("Mat->Image Conversion (%s) - Mat: %s -> Image: %s",
		conversionType, matInfo, imageInfo))

	// Sample a few pixels to verify conversion
	dm.LogMatPixelSample("ConversionSource", fromMat, 3)
}
