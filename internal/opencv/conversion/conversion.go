package conversion

import (
	"fmt"
	"image"
	"image/color"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

// ConvertToGrayscale converts multi-channel images to single-channel grayscale
func ConvertToGrayscale(src *safe.Mat) (*safe.Mat, error) {
	if err := safe.ValidateMatForOperation(src, "grayscale conversion"); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if src.Channels() == 1 {
		return src.Clone()
	}

	dst, err := safe.NewMat(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("destination Mat creation failed: %w", err)
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	switch src.Channels() {
	case 3:
		gocv.CvtColor(srcMat, &dstMat, gocv.ColorBGRToGray)
	case 4:
		temp := gocv.NewMat()
		defer temp.Close()
		gocv.CvtColor(srcMat, &temp, gocv.ColorBGRAToBGR)
		gocv.CvtColor(temp, &dstMat, gocv.ColorBGRToGray)
	default:
		dst.Close()
		return nil, fmt.Errorf("unsupported channel count: %d", src.Channels())
	}

	return dst, nil
}

// MatToImage converts GoCV Mat to standard Go image
func MatToImage(src *safe.Mat) (image.Image, error) {
	if err := safe.ValidateMatForOperation(src, "Mat to image conversion"); err != nil {
		return nil, err
	}

	rows := src.Rows()
	cols := src.Cols()
	channels := src.Channels()

	switch channels {
	case 1:
		return matToGray(src, rows, cols)
	case 3:
		return matToBGRToRGBA(src, rows, cols)
	case 4:
		return matToBGRAToRGBA(src, rows, cols)
	default:
		return nil, fmt.Errorf("unsupported channel count: %d", channels)
	}
}

// ImageToMat converts standard Go image to GoCV Mat
func ImageToMat(img image.Image) (*safe.Mat, error) {
	if img == nil {
		return nil, fmt.Errorf("input image is nil")
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	switch typedImg := img.(type) {
	case *image.Gray:
		return grayImageToMat(typedImg, width, height)
	case *image.RGBA:
		return rgbaImageToMat(typedImg, width, height)
	case *image.NRGBA:
		return nrgbaImageToMat(typedImg, width, height)
	default:
		return convertGenericImageToMat(img, width, height)
	}
}

// matToGray converts single-channel Mat to grayscale image
func matToGray(src *safe.Mat, rows, cols int) (*image.Gray, error) {
	img := image.NewGray(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value, err := src.GetUCharAt(y, x)
			if err != nil {
				return nil, fmt.Errorf("pixel access failed at (%d,%d): %w", x, y, err)
			}
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}

	return img, nil
}

// matToBGRToRGBA converts BGR Mat to RGBA image
func matToBGRToRGBA(src *safe.Mat, rows, cols int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b, err := src.GetUCharAt3(y, x, 0)
			if err != nil {
				return nil, fmt.Errorf("B channel access failed at (%d,%d): %w", x, y, err)
			}

			g, err := src.GetUCharAt3(y, x, 1)
			if err != nil {
				return nil, fmt.Errorf("G channel access failed at (%d,%d): %w", x, y, err)
			}

			r, err := src.GetUCharAt3(y, x, 2)
			if err != nil {
				return nil, fmt.Errorf("R channel access failed at (%d,%d): %w", x, y, err)
			}

			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	return img, nil
}

// matToBGRAToRGBA converts BGRA Mat to RGBA image
func matToBGRAToRGBA(src *safe.Mat, rows, cols int) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b, err := src.GetUCharAt3(y, x, 0)
			if err != nil {
				return nil, fmt.Errorf("B channel access failed at (%d,%d): %w", x, y, err)
			}

			g, err := src.GetUCharAt3(y, x, 1)
			if err != nil {
				return nil, fmt.Errorf("G channel access failed at (%d,%d): %w", x, y, err)
			}

			r, err := src.GetUCharAt3(y, x, 2)
			if err != nil {
				return nil, fmt.Errorf("R channel access failed at (%d,%d): %w", x, y, err)
			}

			a, err := src.GetUCharAt3(y, x, 3)
			if err != nil {
				return nil, fmt.Errorf("A channel access failed at (%d,%d): %w", x, y, err)
			}

			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	return img, nil
}

// grayImageToMat converts grayscale image to single-channel Mat
func grayImageToMat(img *image.Gray, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.GrayAt(x+bounds.Min.X, y+bounds.Min.Y)
			if err := mat.SetUCharAt(y, x, pixel.Y); err != nil {
				mat.Close()
				return nil, fmt.Errorf("pixel setting failed at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}

// rgbaImageToMat converts RGBA image to BGR Mat
func rgbaImageToMat(img *image.RGBA, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.RGBAAt(x+bounds.Min.X, y+bounds.Min.Y)
			
			if err := mat.SetUCharAt3(y, x, 0, pixel.B); err != nil {
				mat.Close()
				return nil, fmt.Errorf("B channel setting failed at (%d,%d): %w", x, y, err)
			}
			if err := mat.SetUCharAt3(y, x, 1, pixel.G); err != nil {
				mat.Close()
				return nil, fmt.Errorf("G channel setting failed at (%d,%d): %w", x, y, err)
			}
			if err := mat.SetUCharAt3(y, x, 2, pixel.R); err != nil {
				mat.Close()
				return nil, fmt.Errorf("R channel setting failed at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}

// nrgbaImageToMat converts NRGBA image to BGR Mat
func nrgbaImageToMat(img *image.NRGBA, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := img.NRGBAAt(x+bounds.Min.X, y+bounds.Min.Y)
			
			if err := mat.SetUCharAt3(y, x, 0, pixel.B); err != nil {
				mat.Close()
				return nil, fmt.Errorf("B channel setting failed at (%d,%d): %w", x, y, err)
			}
			if err := mat.SetUCharAt3(y, x, 1, pixel.G); err != nil {
				mat.Close()
				return nil, fmt.Errorf("G channel setting failed at (%d,%d): %w", x, y, err)
			}
			if err := mat.SetUCharAt3(y, x, 2, pixel.R); err != nil {
				mat.Close()
				return nil, fmt.Errorf("R channel setting failed at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}

// convertGenericImageToMat converts any image type to BGR Mat using color model conversion
func convertGenericImageToMat(img image.Image, width, height int) (*safe.Mat, error) {
	mat, err := safe.NewMat(height, width, gocv.MatTypeCV8UC3)
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			
			// Convert from 16-bit to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			
			if err := mat.SetUCharAt3(y, x, 0, b8); err != nil {
				mat.Close()
				return nil, fmt.Errorf("B channel setting failed at (%d,%d): %w", x, y, err)
			}
			if err := mat.SetUCharAt3(y, x, 1, g8); err != nil {
				mat.Close()
				return nil, fmt.Errorf("G channel setting failed at (%d,%d): %w", x, y, err)
			}
			if err := mat.SetUCharAt3(y, x, 2, r8); err != nil {
				mat.Close()
				return nil, fmt.Errorf("R channel setting failed at (%d,%d): %w", x, y, err)
			}
		}
	}

	return mat, nil
}