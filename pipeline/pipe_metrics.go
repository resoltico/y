package pipeline

import (
	"image"
	"math"
	"time"

	"gocv.io/x/gocv"
)

func (pipeline *ImagePipeline) CalculatePSNR(original, processed *ImageData) float64 {
	if original == nil || processed == nil {
		return 0.0
	}

	startTime := pipeline.debugManager.StartTiming("psnr_calculation")
	defer pipeline.debugManager.EndTiming("psnr_calculation", startTime)

	calcStartTime := time.Now()

	// Convert both images to grayscale for comparison
	origGray := gocv.NewMat()
	defer origGray.Close()
	procGray := gocv.NewMat()
	defer procGray.Close()

	if original.Mat.Channels() == 3 {
		gocv.CvtColor(original.Mat, &origGray, gocv.ColorBGRToGray)
	} else {
		original.Mat.CopyTo(&origGray)
	}

	if processed.Mat.Channels() == 3 {
		gocv.CvtColor(processed.Mat, &procGray, gocv.ColorBGRToGray)
	} else {
		processed.Mat.CopyTo(&procGray)
	}

	// Resize processed image to match original if necessary
	if origGray.Rows() != procGray.Rows() || origGray.Cols() != procGray.Cols() {
		resized := gocv.NewMat()
		defer resized.Close()
		gocv.Resize(procGray, &resized, image.Point{X: origGray.Cols(), Y: origGray.Rows()}, 0, 0, gocv.InterpolationLinear)
		resized.CopyTo(&procGray)
	}

	// Calculate MSE
	diff := gocv.NewMat()
	defer diff.Close()
	gocv.AbsDiff(origGray, procGray, &diff)

	diffSquared := gocv.NewMat()
	defer diffSquared.Close()
	gocv.Multiply(diff, diff, &diffSquared)

	// Calculate sum manually
	totalSum := 0.0
	totalPixels := float64(origGray.Rows() * origGray.Cols())
	for y := 0; y < diffSquared.Rows(); y++ {
		for x := 0; x < diffSquared.Cols(); x++ {
			totalSum += float64(diffSquared.GetUCharAt(y, x))
		}
	}
	mse := totalSum / totalPixels

	var psnr float64
	if mse == 0 {
		psnr = math.Inf(1) // Perfect match
	} else {
		maxPixelValue := 255.0
		psnr = 20.0 * math.Log10(maxPixelValue/math.Sqrt(mse))
	}

	// Log debug information
	calcTime := time.Since(calcStartTime)
	pipeline.debugManager.LogImageMetrics(psnr, 0.0, calcTime)

	return psnr
}

func (pipeline *ImagePipeline) CalculateSSIM(original, processed *ImageData) float64 {
	if original == nil || processed == nil {
		return 0.0
	}

	startTime := pipeline.debugManager.StartTiming("ssim_calculation")
	defer pipeline.debugManager.EndTiming("ssim_calculation", startTime)

	calcStartTime := time.Now()

	// Convert both images to grayscale for comparison
	origGray := gocv.NewMat()
	defer origGray.Close()
	procGray := gocv.NewMat()
	defer procGray.Close()

	if original.Mat.Channels() == 3 {
		gocv.CvtColor(original.Mat, &origGray, gocv.ColorBGRToGray)
	} else {
		original.Mat.CopyTo(&origGray)
	}

	if processed.Mat.Channels() == 3 {
		gocv.CvtColor(processed.Mat, &procGray, gocv.ColorBGRToGray)
	} else {
		processed.Mat.CopyTo(&procGray)
	}

	// Resize processed image to match original if necessary
	if origGray.Rows() != procGray.Rows() || origGray.Cols() != procGray.Cols() {
		resized := gocv.NewMat()
		defer resized.Close()
		gocv.Resize(procGray, &resized, image.Point{X: origGray.Cols(), Y: origGray.Rows()}, 0, 0, gocv.InterpolationLinear)
		resized.CopyTo(&procGray)
	}

	// Convert to float for calculation
	orig32 := gocv.NewMat()
	defer orig32.Close()
	proc32 := gocv.NewMat()
	defer proc32.Close()

	origGray.ConvertTo(&orig32, gocv.MatTypeCV32F)
	procGray.ConvertTo(&proc32, gocv.MatTypeCV32F)

	// Simplified SSIM calculation - return a basic correlation value
	rows := orig32.Rows()
	cols := orig32.Cols()

	var sum1, sum2, sum1Sq, sum2Sq, sum12 float64
	totalPixels := float64(rows * cols)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			val1 := float64(orig32.GetFloatAt(y, x))
			val2 := float64(proc32.GetFloatAt(y, x))

			sum1 += val1
			sum2 += val2
			sum1Sq += val1 * val1
			sum2Sq += val2 * val2
			sum12 += val1 * val2
		}
	}

	mean1 := sum1 / totalPixels
	mean2 := sum2 / totalPixels

	numerator := sum12 - totalPixels*mean1*mean2
	denominator1 := sum1Sq - totalPixels*mean1*mean1
	denominator2 := sum2Sq - totalPixels*mean2*mean2

	var ssim float64
	if denominator1 <= 0 || denominator2 <= 0 {
		ssim = 0.0
	} else {
		correlation := numerator / math.Sqrt(denominator1*denominator2)
		ssim = math.Abs(correlation)
	}

	// Log debug information
	calcTime := time.Since(calcStartTime)
	pipeline.debugManager.LogImageMetrics(0.0, ssim, calcTime)

	return ssim
}
