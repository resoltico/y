package pipeline

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

func (pipeline *ImagePipeline) matToImage(mat gocv.Mat) (image.Image, error) {
	// Validate Mat before conversion
	if mat.Empty() {
		return nil, fmt.Errorf("Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()
	channels := mat.Channels()

	if rows <= 0 || cols <= 0 {
		return nil, fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	// Additional validation
	if mat.Type() < 0 {
		return nil, fmt.Errorf("Mat has invalid type: %d", mat.Type())
	}

	// Skip cloning and pixel access to avoid segfaults on corrupted Mats
	// Log basic info without accessing pixel data
	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Converting Mat to Image: %dx%d, %d channels, type %d",
		cols, rows, channels, mat.Type()))

	var resultImage image.Image
	var err error

	switch channels {
	case 1:
		// Grayscale
		gray := image.NewGray(image.Rect(0, 0, cols, rows))
		err = pipeline.copyMatToGray(mat, gray)
		resultImage = gray
	case 3:
		// BGR to RGB
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		err = pipeline.copyMatBGRToRGBA(mat, rgba)
		resultImage = rgba
	case 4:
		// BGRA to RGBA
		rgba := image.NewRGBA(image.Rect(0, 0, cols, rows))
		err = pipeline.copyMatBGRAToRGBA(mat, rgba)
		resultImage = rgba
	default:
		return nil, fmt.Errorf("unsupported number of channels: %d", channels)
	}

	if err != nil {
		return nil, err
	}

	// Debug Image after conversion - skip Mat debugging to avoid segfaults
	pipeline.debugManager.LogPixelAnalysis("MatToImageOutput", resultImage)
	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Conversion completed: %d-channel Mat to Image", channels))

	return resultImage, nil
}

func (pipeline *ImagePipeline) copyMatToGray(mat gocv.Mat, img *image.Gray) error {
	// Validate inputs
	if mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Channels() != 1 {
		return fmt.Errorf("expected 1-channel Mat, got %d channels", mat.Channels())
	}

	bounds := img.Bounds()
	if bounds.Dx() != cols || bounds.Dy() != rows {
		return fmt.Errorf("image size mismatch: Mat=%dx%d, Image=%dx%d", cols, rows, bounds.Dx(), bounds.Dy())
	}

	// Use recovery for any remaining memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat to Gray conversion: %v", r))
		}
	}()

	pipeline.debugManager.LogInfo("Conversion", fmt.Sprintf("Copying Mat to Gray: %dx%d", cols, rows))

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			value := mat.GetUCharAt(y, x)
			img.SetGray(x, y, color.Gray{Y: value})
		}
	}

	pipeline.debugManager.LogPixelAnalysis("GrayConversionResult", img)
	return nil
}

func (pipeline *ImagePipeline) copyMatBGRToRGBA(mat gocv.Mat, img *image.RGBA) error {
	// Validate inputs
	if mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Channels() != 3 {
		return fmt.Errorf("expected 3-channel Mat, got %d channels", mat.Channels())
	}

	bounds := img.Bounds()
	if bounds.Dx() != cols || bounds.Dy() != rows {
		return fmt.Errorf("image size mismatch: Mat=%dx%d, Image=%dx%d", cols, rows, bounds.Dx(), bounds.Dy())
	}

	// Use recovery for any remaining memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat BGR to RGBA conversion: %v", r))
		}
	}()

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
	return nil
}

func (pipeline *ImagePipeline) copyMatBGRAToRGBA(mat gocv.Mat, img *image.RGBA) error {
	// Validate inputs
	if mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	rows := mat.Rows()
	cols := mat.Cols()

	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("Mat has invalid dimensions: %dx%d", cols, rows)
	}

	if mat.Channels() != 4 {
		return fmt.Errorf("expected 4-channel Mat, got %d channels", mat.Channels())
	}

	bounds := img.Bounds()
	if bounds.Dx() != cols || bounds.Dy() != rows {
		return fmt.Errorf("image size mismatch: Mat=%dx%d, Image=%dx%d", cols, rows, bounds.Dx(), bounds.Dy())
	}

	// Use recovery for any remaining memory access issues
	defer func() {
		if r := recover(); r != nil {
			pipeline.debugManager.LogWarning("Conversion", fmt.Sprintf("Panic during Mat BGRA to RGBA conversion: %v", r))
		}
	}()

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
	return nil
}
