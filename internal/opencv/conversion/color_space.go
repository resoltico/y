package conversion

import (
	"fmt"
	"math"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

// ColorSpace represents different color space types
type ColorSpace int

const (
	ColorSpaceBGR ColorSpace = iota
	ColorSpaceRGB
	ColorSpaceHSV
	ColorSpaceLab
	ColorSpaceGray
	ColorSpaceYUV
)

// ConvertColorSpace converts Mat between different color spaces
func ConvertColorSpace(src *safe.Mat, targetSpace ColorSpace) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "color space conversion"); err != nil {
		return nil, err
	}

	currentSpace := determineColorSpace(src)
	if currentSpace == targetSpace {
		return src.Clone()
	}

	return performColorSpaceConversion(src, currentSpace, targetSpace)
}

// ConvertBGRToHSV converts BGR image to HSV color space
func ConvertBGRToHSV(src *safe.Mat) (*safe.Mat, error) {
	if err := validateBGRMat(src); err != nil {
		return nil, err
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.CvtColor(srcMat, &dstMat, gocv.ColorBGRToHSV)

	return dst, nil
}

// ConvertHSVToBGR converts HSV image to BGR color space
func ConvertHSVToBGR(src *safe.Mat) (*safe.Mat, error) {
	if err := validateHSVMat(src); err != nil {
		return nil, err
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.CvtColor(srcMat, &dstMat, gocv.ColorHSVToBGR)

	return dst, nil
}

// ConvertBGRToLab converts BGR image to Lab color space
func ConvertBGRToLab(src *safe.Mat) (*safe.Mat, error) {
	if err := validateBGRMat(src); err != nil {
		return nil, err
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.CvtColor(srcMat, &dstMat, gocv.ColorBGRToLab)

	return dst, nil
}

// ConvertLabToBGR converts Lab image to BGR color space
func ConvertLabToBGR(src *safe.Mat) (*safe.Mat, error) {
	if err := validateLabMat(src); err != nil {
		return nil, err
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.CvtColor(srcMat, &dstMat, gocv.ColorLabToBGR)

	return dst, nil
}

// ConvertBGRToYUV converts BGR image to YUV color space
func ConvertBGRToYUV(src *safe.Mat) (*safe.Mat, error) {
	if err := validateBGRMat(src); err != nil {
		return nil, err
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.CvtColor(srcMat, &dstMat, gocv.ColorBGRToYUV)

	return dst, nil
}

// ConvertYUVToBGR converts YUV image to BGR color space
func ConvertYUVToBGR(src *safe.Mat) (*safe.Mat, error) {
	if err := validateYUVMat(src); err != nil {
		return nil, err
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()
	gocv.CvtColor(srcMat, &dstMat, gocv.ColorYUVToBGR)

	return dst, nil
}

// ExtractLuminanceChannel extracts luminance from color image using different methods
func ExtractLuminanceChannel(src *safe.Mat, method LuminanceMethod) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "luminance extraction"); err != nil {
		return nil, err
	}

	if src.Channels() == 1 {
		return src.Clone()
	}

	switch method {
	case LuminanceOpenCV:
		return ConvertToGrayscale(src)
	case LuminanceNTSC:
		return extractLuminanceNTSC(src)
	case LuminanceRec709:
		return extractLuminanceRec709(src)
	case LuminanceAverage:
		return extractLuminanceAverage(src)
	default:
		return ConvertToGrayscale(src)
	}
}

// LuminanceMethod defines different luminance extraction methods
type LuminanceMethod int

const (
	LuminanceOpenCV LuminanceMethod = iota
	LuminanceNTSC
	LuminanceRec709
	LuminanceAverage
)

// performColorSpaceConversion handles conversion between color spaces
func performColorSpaceConversion(src *safe.Mat, currentSpace, targetSpace ColorSpace) (*safe.Mat, error) {
	// Direct conversion cases
	if currentSpace == ColorSpaceBGR && targetSpace == ColorSpaceGray {
		return ConvertToGrayscale(src)
	}
	if currentSpace == ColorSpaceBGR && targetSpace == ColorSpaceHSV {
		return ConvertBGRToHSV(src)
	}
	if currentSpace == ColorSpaceHSV && targetSpace == ColorSpaceBGR {
		return ConvertHSVToBGR(src)
	}
	if currentSpace == ColorSpaceBGR && targetSpace == ColorSpaceLab {
		return ConvertBGRToLab(src)
	}
	if currentSpace == ColorSpaceLab && targetSpace == ColorSpaceBGR {
		return ConvertLabToBGR(src)
	}
	if currentSpace == ColorSpaceBGR && targetSpace == ColorSpaceYUV {
		return ConvertBGRToYUV(src)
	}
	if currentSpace == ColorSpaceYUV && targetSpace == ColorSpaceBGR {
		return ConvertYUVToBGR(src)
	}

	// Multi-step conversions through BGR as intermediate
	if currentSpace != ColorSpaceBGR && targetSpace != ColorSpaceBGR {
		// Convert to BGR first
		bgrMat, err := performColorSpaceConversion(src, currentSpace, ColorSpaceBGR)
		if err != nil {
			return nil, fmt.Errorf("intermediate BGR conversion failed: %w", err)
		}
		defer bgrMat.Close()

		// Convert from BGR to target
		return performColorSpaceConversion(bgrMat, ColorSpaceBGR, targetSpace)
	}

	return nil, fmt.Errorf("unsupported color space conversion from %v to %v", currentSpace, targetSpace)
}

// determineColorSpace attempts to identify the color space of a Mat
func determineColorSpace(mat *safe.Mat) ColorSpace {
	channels := mat.Channels()
	
	switch channels {
	case 1:
		return ColorSpaceGray
	case 3:
		return ColorSpaceBGR // Default assumption for 3-channel
	case 4:
		return ColorSpaceBGR // Treat as BGR with alpha
	default:
		return ColorSpaceBGR // Fallback
	}
}

// extractLuminanceNTSC uses NTSC weights for luminance calculation
func extractLuminanceNTSC(src *safe.Mat) (*safe.Mat, error) {
	return extractLuminanceWeighted(src, 0.299, 0.587, 0.114)
}

// extractLuminanceRec709 uses Rec. 709 weights for luminance calculation
func extractLuminanceRec709(src *safe.Mat) (*safe.Mat, error) {
	return extractLuminanceWeighted(src, 0.2126, 0.7152, 0.0722)
}

// extractLuminanceAverage uses simple averaging for luminance
func extractLuminanceAverage(src *safe.Mat) (*safe.Mat, error) {
	return extractLuminanceWeighted(src, 1.0/3.0, 1.0/3.0, 1.0/3.0)
}

// extractLuminanceWeighted performs weighted luminance extraction
func extractLuminanceWeighted(src *safe.Mat, rWeight, gWeight, bWeight float64) (*safe.Mat, error) {
	if src.Channels() != 3 {
		return nil, fmt.Errorf("weighted luminance requires 3-channel input, got %d", src.Channels())
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b, err := src.GetUCharAt3(y, x, 0)
			if err != nil {
				dst.Close()
				return nil, fmt.Errorf("B channel access failed at (%d,%d): %w", x, y, err)
			}

			g, err := src.GetUCharAt3(y, x, 1)
			if err != nil {
				dst.Close()
				return nil, fmt.Errorf("G channel access failed at (%d,%d): %w", x, y, err)
			}

			r, err := src.GetUCharAt3(y, x, 2)
			if err != nil {
				dst.Close()
				return nil, fmt.Errorf("R channel access failed at (%d,%d): %w", x, y, err)
			}

			luminance := rWeight*float64(r) + gWeight*float64(g) + bWeight*float64(b)
			luminanceValue := uint8(math.Max(0, math.Min(255, luminance)))

			if err := dst.SetUCharAt(y, x, luminanceValue); err != nil {
				dst.Close()
				return nil, fmt.Errorf("luminance setting failed at (%d,%d): %w", x, y, err)
			}
		}
	}

	return dst, nil
}

// Validation functions for different color spaces
func validateBGRMat(mat *safe.Mat) error {
	if err := safe.ValidateMatForOperation(mat, "BGR validation"); err != nil {
		return err
	}
	if mat.Channels() != 3 {
		return fmt.Errorf("BGR Mat requires 3 channels, got %d", mat.Channels())
	}
	return nil
}

func validateHSVMat(mat *safe.Mat) error {
	if err := safe.ValidateMatForOperation(mat, "HSV validation"); err != nil {
		return err
	}
	if mat.Channels() != 3 {
		return fmt.Errorf("HSV Mat requires 3 channels, got %d", mat.Channels())
	}
	return nil
}

func validateLabMat(mat *safe.Mat) error {
	if err := safe.ValidateMatForOperation(mat, "Lab validation"); err != nil {
		return err
	}
	if mat.Channels() != 3 {
		return fmt.Errorf("Lab Mat requires 3 channels, got %d", mat.Channels())
	}
	return nil
}

func validateYUVMat(mat *safe.Mat) error {
	if err := safe.ValidateMatForOperation(mat, "YUV validation"); err != nil {
		return err
	}
	if mat.Channels() != 3 {
		return fmt.Errorf("YUV Mat requires 3 channels, got %d", mat.Channels())
	}
	return nil
}