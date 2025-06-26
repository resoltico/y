package triclass

import (
	"fmt"
	"image"

	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

func (p *Processor) performIterativeTriclass(working *safe.Mat, params map[string]interface{}) (*safe.Mat, error) {
	maxIterations := p.getIntParam(params, "max_iterations")
	convergenceEpsilon := p.getFloatParam(params, "convergence_epsilon")
	minTBDFraction := p.getFloatParam(params, "minimum_tbd_fraction")

	result, err := safe.NewMat(working.Rows(), working.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, fmt.Errorf("failed to create result Mat: %w", err)
	}

	currentRegion, err := working.Clone()
	if err != nil {
		result.Close()
		return nil, fmt.Errorf("failed to clone working Mat: %w", err)
	}
	defer currentRegion.Close()

	var previousThreshold float64 = -1

	for iteration := 0; iteration < maxIterations; iteration++ {
		nonZeroPixels := p.countNonZeroPixels(currentRegion)
		if nonZeroPixels == 0 {
			break
		}

		threshold := p.calculateThresholdForRegion(currentRegion, params)

		convergence := abs(threshold - previousThreshold)
		if previousThreshold >= 0 && convergence < convergenceEpsilon {
			break
		}
		previousThreshold = threshold

		foregroundMask, backgroundMask, tbdMask, err := p.segmentRegion(currentRegion, threshold, params)
		if err != nil {
			return nil, fmt.Errorf("segmentation failed at iteration %d: %w", iteration, err)
		}

		tbdCount := p.countNonZeroPixels(tbdMask)
		totalPixels := currentRegion.Rows() * currentRegion.Cols()
		tbdFraction := float64(tbdCount) / float64(totalPixels)

		p.updateResult(result, foregroundMask)

		foregroundMask.Close()
		backgroundMask.Close()

		if tbdFraction < minTBDFraction {
			tbdMask.Close()
			break
		}

		newRegion, err := p.extractTBDRegion(working, tbdMask)
		tbdMask.Close()
		if err != nil {
			return nil, fmt.Errorf("TBD region extraction failed: %w", err)
		}

		currentRegion.Close()
		currentRegion = newRegion
	}

	return result, nil
}

func (p *Processor) countNonZeroPixels(mat *safe.Mat) int {
	rows := mat.Rows()
	cols := mat.Cols()
	count := 0

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := mat.GetUCharAt(y, x); err == nil && value > 0 {
				count++
			}
		}
	}

	return count
}

func (p *Processor) calculateThresholdForRegion(region *safe.Mat, params map[string]interface{}) float64 {
	method := p.getStringParam(params, "initial_threshold_method")
	histBins := p.getIntParam(params, "histogram_bins")

	histogram := p.calculateHistogram(region, histBins)

	switch method {
	case "mean":
		return p.calculateMeanThreshold(histogram, histBins)
	case "median":
		return p.calculateMedianThreshold(histogram, histBins)
	default: // "otsu"
		return p.calculateOtsuThreshold(histogram, histBins)
	}
}

func (p *Processor) calculateHistogram(src *safe.Mat, histBins int) []int {
	histogram := make([]int, histBins)
	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if pixelValue, err := src.GetUCharAt(y, x); err == nil && pixelValue > 0 {
				bin := int(float64(pixelValue) * float64(histBins-1) / 255.0)
				if bin < 0 {
					bin = 0
				} else if bin >= histBins {
					bin = histBins - 1
				}
				histogram[bin]++
			}
		}
	}

	return histogram
}

func (p *Processor) calculateOtsuThreshold(histogram []int, histBins int) float64 {
	total := 0
	for i := 0; i < histBins; i++ {
		total += histogram[i]
	}

	if total == 0 {
		return 127.5
	}

	sum := 0.0
	for i := 0; i < histBins; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	bestThreshold := 127.5

	for t := 0; t < histBins; t++ {
		wB += histogram[t]
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += float64(t) * float64(histogram[t])

		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)

		varBetween := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if varBetween > maxVariance {
			maxVariance = varBetween
			bestThreshold = float64(t) * 255.0 / float64(histBins-1)
		}
	}

	return bestThreshold
}

func (p *Processor) calculateMeanThreshold(histogram []int, histBins int) float64 {
	totalPixels := 0
	weightedSum := 0.0

	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
		weightedSum += float64(i) * float64(histogram[i])
	}

	if totalPixels == 0 {
		return 127.5
	}

	meanBin := weightedSum / float64(totalPixels)
	return meanBin * 255.0 / float64(histBins-1)
}

func (p *Processor) calculateMedianThreshold(histogram []int, histBins int) float64 {
	totalPixels := 0
	for i := 0; i < histBins; i++ {
		totalPixels += histogram[i]
	}

	if totalPixels == 0 {
		return 127.5
	}

	halfPixels := totalPixels / 2
	cumSum := 0

	for i := 0; i < histBins; i++ {
		cumSum += histogram[i]
		if cumSum >= halfPixels {
			return float64(i) * 255.0 / float64(histBins-1)
		}
	}

	return 127.5
}

func (p *Processor) segmentRegion(region *safe.Mat, threshold float64, params map[string]interface{}) (*safe.Mat, *safe.Mat, *safe.Mat, error) {
	rows := region.Rows()
	cols := region.Cols()

	foreground, err := safe.NewMat(rows, cols, gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, nil, nil, err
	}

	background, err := safe.NewMat(rows, cols, gocv.MatTypeCV8UC1)
	if err != nil {
		foreground.Close()
		return nil, nil, nil, err
	}

	tbd, err := safe.NewMat(rows, cols, gocv.MatTypeCV8UC1)
	if err != nil {
		foreground.Close()
		background.Close()
		return nil, nil, nil, err
	}

	gapFactor := p.getFloatParam(params, "lower_upper_gap_factor")
	lowerThreshold := threshold * (1.0 - gapFactor)
	upperThreshold := threshold * (1.0 + gapFactor)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelValue, err := region.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			if pixelValue > 0 {
				if float64(pixelValue) > upperThreshold {
					foreground.SetUCharAt(y, x, 255)
				} else if float64(pixelValue) < lowerThreshold {
					background.SetUCharAt(y, x, 255)
				} else {
					tbd.SetUCharAt(y, x, 255)
				}
			}
		}
	}

	return foreground, background, tbd, nil
}

func (p *Processor) updateResult(result, foregroundMask *safe.Mat) {
	rows := result.Rows()
	cols := result.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if value, err := foregroundMask.GetUCharAt(y, x); err == nil && value > 0 {
				result.SetUCharAt(y, x, 255)
			}
		}
	}
}

func (p *Processor) extractTBDRegion(original, tbdMask *safe.Mat) (*safe.Mat, error) {
	result, err := safe.NewMat(original.Rows(), original.Cols(), gocv.MatTypeCV8UC1)
	if err != nil {
		return nil, err
	}

	rows := original.Rows()
	cols := original.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if tbdValue, err := tbdMask.GetUCharAt(y, x); err == nil && tbdValue > 0 {
				if origValue, err := original.GetUCharAt(y, x); err == nil {
					result.SetUCharAt(y, x, origValue)
				}
			}
		}
	}

	return result, nil
}

func (p *Processor) applyPreprocessing(src *safe.Mat) (*safe.Mat, error) {
	clahe := gocv.NewCLAHE()
	defer clahe.Close()

	dst, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	dstMat := dst.GetMat()

	clahe.Apply(srcMat, &dstMat)

	denoised, err := safe.NewMat(dst.Rows(), dst.Cols(), dst.Type())
	if err != nil {
		dst.Close()
		return nil, err
	}

	denoisedMat := denoised.GetMat()
	gocv.FastNlMeansDenoising(dstMat, &denoisedMat)

	dst.Close()
	return denoised, nil
}

func (p *Processor) applyCleanup(src *safe.Mat) (*safe.Mat, error) {
	kernel := gocv.GetStructuringElement(gocv.MorphEllipse, image.Point{X: 3, Y: 3})
	defer kernel.Close()

	opened, err := safe.NewMat(src.Rows(), src.Cols(), src.Type())
	if err != nil {
		return nil, err
	}

	srcMat := src.GetMat()
	openedMat := opened.GetMat()

	gocv.MorphologyEx(srcMat, &openedMat, gocv.MorphOpen, kernel)

	closed, err := safe.NewMat(opened.Rows(), opened.Cols(), opened.Type())
	if err != nil {
		opened.Close()
		return nil, err
	}

	closedMat := closed.GetMat()
	gocv.MorphologyEx(openedMat, &closedMat, gocv.MorphClose, kernel)

	opened.Close()

	medianFiltered, err := safe.NewMat(closed.Rows(), closed.Cols(), closed.Type())
	if err != nil {
		closed.Close()
		return nil, err
	}

	medianMat := medianFiltered.GetMat()
	gocv.MedianBlur(closedMat, &medianMat, 3)

	closed.Close()
	return medianFiltered, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
