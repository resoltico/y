package conversion

import (
	"fmt"
	"math"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

// MatProperties contains information about Mat characteristics
type MatProperties struct {
	Rows     int
	Cols     int
	Channels int
	Type     gocv.MatType
	DataType string
	Empty    bool
}

// GetMatProperties returns detailed information about a Mat
func GetMatProperties(mat *safe.Mat) MatProperties {
	if mat == nil {
		return MatProperties{Empty: true}
	}

	return MatProperties{
		Rows:     mat.Rows(),
		Cols:     mat.Cols(),
		Channels: mat.Channels(),
		Type:     mat.Type(),
		DataType: getDataTypeName(mat.Type()),
		Empty:    mat.Empty(),
	}
}

// CloneMat creates a deep copy of the source Mat
func CloneMat(src *safe.Mat) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "Mat cloning"); err != nil {
		return nil, err
	}

	return src.Clone()
}

// ConvertMatType converts Mat to a different data type
func ConvertMatType(src *safe.Mat, targetType gocv.MatType) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "Mat type conversion"); err != nil {
		return nil, err
	}

	if src.Type() == targetType {
		return src.Clone()
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), targetType)
	if err != nil {
		return nil, fmt.Errorf("destination Mat creation failed: %w", err)
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	// Use appropriate conversion factor based on data types
	scale, offset := getConversionParameters(src.Type(), targetType)
	
	srcMat.ConvertTo(&dstMat, int(targetType), scale, offset)

	return dst, nil
}

// ResizeMat resizes Mat to new dimensions using specified interpolation
func ResizeMat(src *safe.Mat, newWidth, newHeight int, interpolation gocv.InterpolationFlags) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "Mat resizing"); err != nil {
		return nil, err
	}

	if newWidth <= 0 || newHeight <= 0 {
		return nil, fmt.Errorf("invalid dimensions: %dx%d", newWidth, newHeight)
	}

	dst, err := safe.NewMat(newHeight, newWidth, src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	gocv.Resize(srcMat, &dstMat, gocv.Point{X: newWidth, Y: newHeight}, 0, 0, interpolation)

	return dst, nil
}

// CropMat extracts a rectangular region from the Mat
func CropMat(src *safe.Mat, x, y, width, height int) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "Mat cropping"); err != nil {
		return nil, err
	}

	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid crop parameters: x=%d, y=%d, w=%d, h=%d", x, y, width, height)
	}

	if x+width > src.Cols() || y+height > src.Rows() {
		return nil, fmt.Errorf("crop region exceeds Mat bounds: Mat=%dx%d, crop=%d,%d to %d,%d",
			src.Cols(), src.Rows(), x, y, x+width, y+height)
	}

	dst, err := safe.NewMat(height, width, src.Type())
	if err != nil {
		return nil, err
	}

	// Copy pixel data from source region to destination
	for dstY := 0; dstY < height; dstY++ {
		for dstX := 0; dstX < width; dstX++ {
			srcX := x + dstX
			srcY := y + dstY

			switch src.Channels() {
			case 1:
				val, err := src.GetUCharAt(srcY, srcX)
				if err != nil {
					dst.Close()
					return nil, fmt.Errorf("pixel access failed at (%d,%d): %w", srcX, srcY, err)
				}
				if err := dst.SetUCharAt(dstY, dstX, val); err != nil {
					dst.Close()
					return nil, fmt.Errorf("pixel setting failed at (%d,%d): %w", dstX, dstY, err)
				}
			case 3:
				for ch := 0; ch < 3; ch++ {
					val, err := src.GetUCharAt3(srcY, srcX, ch)
					if err != nil {
						dst.Close()
						return nil, fmt.Errorf("channel %d access failed at (%d,%d): %w", ch, srcX, srcY, err)
					}
					if err := dst.SetUCharAt3(dstY, dstX, ch, val); err != nil {
						dst.Close()
						return nil, fmt.Errorf("channel %d setting failed at (%d,%d): %w", ch, dstX, dstY, err)
					}
				}
			default:
				dst.Close()
				return nil, fmt.Errorf("unsupported channel count for cropping: %d", src.Channels())
			}
		}
	}

	return dst, nil
}

// NormalizeMat normalizes pixel values to 0-255 range
func NormalizeMat(src *safe.Mat) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "Mat normalization"); err != nil {
		return nil, err
	}

	// Find min and max values
	minVal, maxVal := findMinMaxValues(src)
	
	if maxVal == minVal {
		// All pixels have the same value - return a copy
		return src.Clone()
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	scale := 255.0 / (maxVal - minVal)
	
	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			switch src.Channels() {
			case 1:
				val, _ := src.GetUCharAt(y, x)
				normalized := uint8((float64(val) - minVal) * scale)
				dst.SetUCharAt(y, x, normalized)
			case 3:
				for ch := 0; ch < 3; ch++ {
					val, _ := src.GetUCharAt3(y, x, ch)
					normalized := uint8((float64(val) - minVal) * scale)
					dst.SetUCharAt3(y, x, ch, normalized)
				}
			}
		}
	}

	return dst, nil
}

// CopyMat creates a copy of source Mat into destination Mat
func CopyMat(src, dst *safe.Mat) error {
	if err := safe.ValidateMatForOperation(src, "source Mat copy"); err != nil {
		return err
	}
	if err := safe.ValidateMatForOperation(dst, "destination Mat copy"); err != nil {
		return err
	}

	if src.Rows() != dst.Rows() || src.Cols() != dst.Cols() || src.Type() != dst.Type() {
		return fmt.Errorf("Mat dimensions mismatch: src=%dx%d type=%d, dst=%dx%d type=%d",
			src.Cols(), src.Rows(), int(src.Type()),
			dst.Cols(), dst.Rows(), int(dst.Type()))
	}

	return src.CopyTo(dst)
}

// FillMat fills Mat with specified value
func FillMat(mat *safe.Mat, value uint8) error {
	if err := safe.ValidateMatForOperation(mat, "Mat filling"); err != nil {
		return err
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			switch channels {
			case 1:
				if err := mat.SetUCharAt(y, x, value); err != nil {
					return fmt.Errorf("pixel setting failed at (%d,%d): %w", x, y, err)
				}
			case 3:
				for ch := 0; ch < 3; ch++ {
					if err := mat.SetUCharAt3(y, x, ch, value); err != nil {
						return fmt.Errorf("channel %d setting failed at (%d,%d): %w", ch, x, y, err)
					}
				}
			default:
				return fmt.Errorf("unsupported channel count: %d", channels)
			}
		}
	}

	return nil
}

// Helper functions

// getDataTypeName returns human-readable name for MatType
func getDataTypeName(matType gocv.MatType) string {
	switch matType {
	case gocv.MatTypeCV8UC1:
		return "8-bit unsigned single channel"
	case gocv.MatTypeCV8UC3:
		return "8-bit unsigned 3-channel"
	case gocv.MatTypeCV8UC4:
		return "8-bit unsigned 4-channel"
	case gocv.MatTypeCV16UC1:
		return "16-bit unsigned single channel"
	case gocv.MatTypeCV16UC3:
		return "16-bit unsigned 3-channel"
	case gocv.MatTypeCV32FC1:
		return "32-bit float single channel"
	case gocv.MatTypeCV32FC3:
		return "32-bit float 3-channel"
	case gocv.MatTypeCV64FC1:
		return "64-bit float single channel"
	case gocv.MatTypeCV64FC3:
		return "64-bit float 3-channel"
	default:
		return fmt.Sprintf("unknown type %d", int(matType))
	}
}

// getConversionParameters returns scale and offset for type conversion
func getConversionParameters(srcType, dstType gocv.MatType) (scale, offset float64) {
	scale = 1.0
	offset = 0.0

	// Handle common conversion cases
	switch {
	case isFloatType(srcType) && isIntType(dstType):
		scale = 255.0 // Float [0,1] to int [0,255]
	case isIntType(srcType) && isFloatType(dstType):
		scale = 1.0 / 255.0 // Int [0,255] to float [0,1]
	case is16BitType(srcType) && is8BitType(dstType):
		scale = 1.0 / 256.0 // 16-bit to 8-bit
	case is8BitType(srcType) && is16BitType(dstType):
		scale = 256.0 // 8-bit to 16-bit
	}

	return scale, offset
}

// findMinMaxValues finds minimum and maximum pixel values in Mat
func findMinMaxValues(mat *safe.Mat) (minVal, maxVal float64) {
	minVal = math.Inf(1)
	maxVal = math.Inf(-1)

	rows := mat.Rows()
	cols := mat.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val, err := mat.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			floatVal := float64(val)
			if floatVal < minVal {
				minVal = floatVal
			}
			if floatVal > maxVal {
				maxVal = floatVal
			}
		}
	}

	if math.IsInf(minVal, 1) {
		minVal = 0
	}
	if math.IsInf(maxVal, -1) {
		maxVal = 255
	}

	return minVal, maxVal
}

// Type checking helpers
func isFloatType(matType gocv.MatType) bool {
	return matType == gocv.MatTypeCV32FC1 || matType == gocv.MatTypeCV32FC3 ||
		matType == gocv.MatTypeCV64FC1 || matType == gocv.MatTypeCV64FC3
}

func isIntType(matType gocv.MatType) bool {
	return matType == gocv.MatTypeCV8UC1 || matType == gocv.MatTypeCV8UC3 ||
		matType == gocv.MatTypeCV8UC4 || matType == gocv.MatTypeCV16UC1 || matType == gocv.MatTypeCV16UC3
}

func is8BitType(matType gocv.MatType) bool {
	return matType == gocv.MatTypeCV8UC1 || matType == gocv.MatTypeCV8UC3 || matType == gocv.MatTypeCV8UC4
}

func is16BitType(matType gocv.MatType) bool {
	return matType == gocv.MatTypeCV16UC1 || matType == gocv.MatTypeCV16UC3
}