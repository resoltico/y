package pipeline

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

func (pipeline *ImagePipeline) matToImage(mat gocv.Mat) (image.Image, error) {
	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	// Debug Mat before conversion
	pipeline.debugManager.LogMatPixelAnalysis("MatToImageInput", mat)

	var resultImage image.Image
	var err error

	switch channels {
	case 1:
		// Grayscale
		gray := image.NewGray(image.Rect(0, 0, cols, rows))
		pipeline.copyMatToGray(mat, gray)
		resultImage = gray
	case 3:
		// BGR to RGB
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		pipeline.copyMatBGRToRGBA(mat, rgba)
		resultImage = rgba
	case 4:
		// BGRA to RGBA
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		pipeline.copyMatBGRAToRGBA(mat, rgba)
		resultImage = rgba
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}

	// Debug Image after conversion
	pipeline.debugManager.LogPixelAnalysis("MatToImageOutput", resultImage)
	pipeline.debugManager.LogImageConversionDebug(mat, resultImage, fmt.Sprintf("%d-channel", channels))

	return resultImage, err
}

func (pipeline *ImagePipeline) copyMatToGray(mat gocv.Mat, img *image.Gray) {
	rows := mat.Rows()
	cols := mat.Cols()

	// Debug sample before copy
	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Copying Mat to Gray: %dx%d", cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value := mat.GetUCharAt(y, x)
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}

	// Debug sample after copy
	pipeline.debugManager.LogPixelAnalysis("GrayConversionResult", img)
}

func (pipeline *ImagePipeline) copyMatBGRToRGBA(mat gocv.Mat, img *image.RGBA) {
	rows := mat.Rows()
	cols := mat.Cols()

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Copying Mat BGR to RGBA: %dx%d", cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b := mat.GetUCharAt3(y, x, 0)
			g := mat.GetUCharAt3(y, x, 1)
			r := mat.GetUCharAt3(y, x, 2)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}

	pipeline.debugManager.LogPixelAnalysis("BGRToRGBAConversionResult", img)
}

func (pipeline *ImagePipeline) copyMatBGRAToRGBA(mat gocv.Mat, img *image.RGBA) {
	rows := mat.Rows()
	cols := mat.Cols()

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Copying Mat BGRA to RGBA: %dx%d", cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b := mat.GetUCharAt3(y, x, 0)
			g := mat.GetUCharAt3(y, x, 1)
			r := mat.GetUCharAt3(y, x, 2)
			a := mat.GetUCharAt3(y, x, 3)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	pipeline.debugManager.LogPixelAnalysis("BGRAToRGBAConversionResult", img)
}
