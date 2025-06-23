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

	switch channels {
	case 1:
		// Grayscale
		gray := image.NewGray(image.Rect(0, 0, cols, rows))
		pipeline.copyMatToGray(mat, gray)
		return gray, nil
	case 3:
		// BGR to RGB
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		pipeline.copyMatBGRToRGBA(mat, rgba)
		return rgba, nil
	case 4:
		// BGRA to RGBA
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		pipeline.copyMatBGRAToRGBA(mat, rgba)
		return rgba, nil
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}
}

func (pipeline *ImagePipeline) copyMatToGray(mat gocv.Mat, img *image.Gray) {
	rows := mat.Rows()
	cols := mat.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value := mat.GetUCharAt(y, x)
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}
}

func (pipeline *ImagePipeline) copyMatBGRToRGBA(mat gocv.Mat, img *image.RGBA) {
	rows := mat.Rows()
	cols := mat.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b := mat.GetUCharAt3(y, x, 0)
			g := mat.GetUCharAt3(y, x, 1)
			r := mat.GetUCharAt3(y, x, 2)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func (pipeline *ImagePipeline) copyMatBGRAToRGBA(mat gocv.Mat, img *image.RGBA) {
	rows := mat.Rows()
	cols := mat.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			b := mat.GetUCharAt3(y, x, 0)
			g := mat.GetUCharAt3(y, x, 1)
			r := mat.GetUCharAt3(y, x, 2)
			a := mat.GetUCharAt3(y, x, 3)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
}
