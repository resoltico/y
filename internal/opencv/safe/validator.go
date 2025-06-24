package safe

import (
	"fmt"

	"gocv.io/x/gocv"
)

func ValidateMatForOperation(mat *Mat, operation string) error {
	if mat == nil {
		return fmt.Errorf("Mat is nil for operation: %s", operation)
	}
	
	if !mat.IsValid() {
		return fmt.Errorf("Mat is invalid for operation: %s", operation)
	}
	
	if mat.Empty() {
		return fmt.Errorf("Mat is empty for operation: %s", operation)
	}
	
	if mat.Rows() <= 0 || mat.Cols() <= 0 {
		return fmt.Errorf("Mat has invalid dimensions %dx%d for operation: %s", 
			mat.Cols(), mat.Rows(), operation)
	}
	
	return nil
}

func ValidateColorConversion(src *Mat, code gocv.ColorConversionCode) error {
	if err := ValidateMatForOperation(src, "CvtColor"); err != nil {
		return err
	}
	
	channels := src.Channels()
	
	switch code {
	case gocv.ColorBGRToGray, gocv.ColorRGBToGray:
		if channels != 3 {
			return fmt.Errorf("BGR/RGB to Gray conversion requires 3 channels, got %d", channels)
		}
	case gocv.ColorGrayToBGR, gocv.ColorGrayToRGB:
		if channels != 1 {
			return fmt.Errorf("Gray to BGR/RGB conversion requires 1 channel, got %d", channels)
		}
	case gocv.ColorBGRToRGB, gocv.ColorRGBToBGR:
		if channels != 3 {
			return fmt.Errorf("BGR/RGB conversion requires 3 channels, got %d", channels)
		}
	case gocv.ColorBGRToBGRA, gocv.ColorRGBToRGBA:
		if channels != 3 {
			return fmt.Errorf("BGR/RGB to BGRA/RGBA conversion requires 3 channels, got %d", channels)
		}
	case gocv.ColorBGRAToBGR, gocv.ColorRGBATorgb:
		if channels != 4 {
			return fmt.Errorf("BGRA/RGBA to BGR/RGB conversion requires 4 channels, got %d", channels)
		}
	}
	
	return nil
}

func ValidateDimensions(width, height int, operation string) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid dimensions %dx%d for operation: %s", width, height, operation)
	}
	
	if width > 32768 || height > 32768 {
		return fmt.Errorf("dimensions %dx%d exceed maximum size for operation: %s", width, height, operation)
	}
	
	return nil
}

func ValidateMatType(matType gocv.MatType, operation string) error {
	switch matType {
	case gocv.MatTypeCV8UC1, gocv.MatTypeCV8UC3, gocv.MatTypeCV8UC4:
		return nil
	case gocv.MatTypeCV16UC1, gocv.MatTypeCV16UC3, gocv.MatTypeCV16UC4:
		return nil
	case gocv.MatTypeCV32FC1, gocv.MatTypeCV32FC3, gocv.MatTypeCV32FC4:
		return nil
	default:
		return fmt.Errorf("unsupported MatType %d for operation: %s", int(matType), operation)
	}
}

func ValidateCoordinates(row, col, rows, cols int, operation string) error {
	if row < 0 || row >= rows {
		return fmt.Errorf("row %d out of bounds [0, %d) for operation: %s", row, rows, operation)
	}
	
	if col < 0 || col >= cols {
		return fmt.Errorf("col %d out of bounds [0, %d) for operation: %s", col, cols, operation)
	}
	
	return nil
}

func ValidateChannel(channel, channels int, operation string) error {
	if channel < 0 || channel >= channels {
		return fmt.Errorf("channel %d out of bounds [0, %d) for operation: %s", channel, channels, operation)
	}
	
	return nil
}