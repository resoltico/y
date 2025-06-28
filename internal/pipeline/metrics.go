package pipeline

import (
	"fmt"
	"math"

	"otsu-obliterator/internal/opencv/safe"
)

// SegmentationMetrics contains quality evaluation metrics for thresholding
type SegmentationMetrics struct {
	IoU                    float64 // Intersection over Union
	DiceCoefficient        float64 // Dice Similarity Coefficient
	MisclassificationError float64 // Misclassification Error Rate
	RegionUniformity       float64 // Intra-region uniformity measure
	BoundaryAccuracy       float64 // Boundary preservation accuracy
	HausdorffDistance      float64 // Maximum boundary discrepancy
}

// CalculateSegmentationMetrics computes task-specific thresholding quality metrics
func CalculateSegmentationMetrics(original, segmented *ImageData, groundTruth *ImageData) (*SegmentationMetrics, error) {
	if original == nil || segmented == nil {
		return nil, fmt.Errorf("original and segmented images cannot be nil")
	}

	// Validate dimensions
	if original.Width != segmented.Width || original.Height != segmented.Height {
		return nil, fmt.Errorf("image dimensions must match: original %dx%d, segmented %dx%d",
			original.Width, original.Height, segmented.Width, segmented.Height)
	}

	metrics := &SegmentationMetrics{}

	// Calculate binary mask metrics
	if err := calculateBinaryMaskMetrics(original.Mat, segmented.Mat, metrics); err != nil {
		return nil, fmt.Errorf("failed to calculate binary mask metrics: %w", err)
	}

	// Calculate region uniformity
	if err := calculateRegionUniformity(original.Mat, segmented.Mat, metrics); err != nil {
		return nil, fmt.Errorf("failed to calculate region uniformity: %w", err)
	}

	// Calculate boundary accuracy
	if err := calculateBoundaryAccuracy(original.Mat, segmented.Mat, metrics); err != nil {
		return nil, fmt.Errorf("failed to calculate boundary accuracy: %w", err)
	}

	// Calculate Hausdorff distance if ground truth is available
	if groundTruth != nil && groundTruth.Width == segmented.Width && groundTruth.Height == segmented.Height {
		if err := calculateHausdorffDistance(groundTruth.Mat, segmented.Mat, metrics); err != nil {
			return nil, fmt.Errorf("failed to calculate Hausdorff distance: %w", err)
		}
	}

	return metrics, nil
}

// calculateBinaryMaskMetrics computes IoU, Dice coefficient, and misclassification error
func calculateBinaryMaskMetrics(original, segmented *safe.Mat, metrics *SegmentationMetrics) error {
	if err := safe.ValidateMatForOperation(original, "binary metrics calculation"); err != nil {
		return err
	}
	if err := safe.ValidateMatForOperation(segmented, "binary metrics calculation"); err != nil {
		return err
	}

	rows := original.Rows()
	cols := original.Cols()

	// Generate ground truth using adaptive thresholding on original
	groundTruth, err := generateAdaptiveGroundTruth(original)
	if err != nil {
		return fmt.Errorf("failed to generate ground truth: %w", err)
	}
	defer groundTruth.Close()

	var truePositive, falsePositive, falseNegative, trueNegative int

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			groundTruthVal, err := groundTruth.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			segmentedVal, err := segmented.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			// Convert to binary (0 or 1)
			gtBinary := 0
			if groundTruthVal > 127 {
				gtBinary = 1
			}

			segBinary := 0
			if segmentedVal > 127 {
				segBinary = 1
			}

			// Count classification results
			if gtBinary == 1 && segBinary == 1 {
				truePositive++
			} else if gtBinary == 0 && segBinary == 1 {
				falsePositive++
			} else if gtBinary == 1 && segBinary == 0 {
				falseNegative++
			} else {
				trueNegative++
			}
		}
	}

	totalPixels := truePositive + falsePositive + falseNegative + trueNegative

	// Calculate IoU (Intersection over Union)
	intersection := float64(truePositive)
	union := float64(truePositive + falsePositive + falseNegative)
	if union > 0 {
		metrics.IoU = intersection / union
	} else {
		metrics.IoU = 1.0 // Perfect match when both are empty
	}

	// Calculate Dice Coefficient
	if truePositive+falsePositive+falseNegative > 0 {
		metrics.DiceCoefficient = (2.0 * intersection) / float64(2*truePositive+falsePositive+falseNegative)
	} else {
		metrics.DiceCoefficient = 1.0
	}

	// Calculate Misclassification Error Rate
	if totalPixels > 0 {
		metrics.MisclassificationError = float64(falsePositive+falseNegative) / float64(totalPixels)
	}

	return nil
}

// generateAdaptiveGroundTruth creates reference segmentation using multiple methods
func generateAdaptiveGroundTruth(original *safe.Mat) (*safe.Mat, error) {
	rows := original.Rows()
	cols := original.Cols()

	// Apply Otsu thresholding as baseline
	otsuThreshold := calculateOtsuThreshold(original)

	groundTruth, err := safe.NewMat(rows, cols, original.Type())
	if err != nil {
		return nil, err
	}

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelVal, err := original.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			var result uint8
			if float64(pixelVal) > otsuThreshold {
				result = 255
			} else {
				result = 0
			}

			groundTruth.SetUCharAt(y, x, result)
		}
	}

	return groundTruth, nil
}

// calculateOtsuThreshold implements standard Otsu thresholding for ground truth generation
func calculateOtsuThreshold(src *safe.Mat) float64 {
	// Build histogram
	histogram := make([]int, 256)
	rows := src.Rows()
	cols := src.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixelVal, err := src.GetUCharAt(y, x)
			if err == nil {
				histogram[pixelVal]++
			}
		}
	}

	// Calculate Otsu threshold
	total := rows * cols
	sum := 0.0
	for i := 0; i < 256; i++ {
		sum += float64(i) * float64(histogram[i])
	}

	sumB := 0.0
	wB := 0
	maxVariance := 0.0
	threshold := 0.0

	for i := 0; i < 256; i++ {
		wB += histogram[i]
		if wB == 0 {
			continue
		}

		wF := total - wB
		if wF == 0 {
			break
		}

		sumB += float64(i) * float64(histogram[i])

		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)

		varBetween := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)

		if varBetween > maxVariance {
			maxVariance = varBetween
			threshold = float64(i)
		}
	}

	return threshold
}

// calculateRegionUniformity measures intra-region homogeneity
func calculateRegionUniformity(original, segmented *safe.Mat, metrics *SegmentationMetrics) error {
	rows := original.Rows()
	cols := original.Cols()

	var foregroundSum, backgroundSum float64
	var foregroundSumSq, backgroundSumSq float64
	var foregroundCount, backgroundCount int

	// Calculate statistics for foreground and background regions
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			originalVal, err := original.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			segmentedVal, err := segmented.GetUCharAt(y, x)
			if err != nil {
				continue
			}

			pixelFloat := float64(originalVal)

			if segmentedVal > 127 { // Foreground
				foregroundSum += pixelFloat
				foregroundSumSq += pixelFloat * pixelFloat
				foregroundCount++
			} else { // Background
				backgroundSum += pixelFloat
				backgroundSumSq += pixelFloat * pixelFloat
				backgroundCount++
			}
		}
	}

	// Calculate variances
	var foregroundVar, backgroundVar float64

	if foregroundCount > 1 {
		foregroundMean := foregroundSum / float64(foregroundCount)
		foregroundVar = (foregroundSumSq - foregroundSum*foregroundMean) / float64(foregroundCount-1)
	}

	if backgroundCount > 1 {
		backgroundMean := backgroundSum / float64(backgroundCount)
		backgroundVar = (backgroundSumSq - backgroundSum*backgroundMean) / float64(backgroundCount-1)
	}

	// Calculate weighted uniformity score (lower variance = higher uniformity)
	totalPixels := float64(foregroundCount + backgroundCount)
	if totalPixels > 0 {
		weightedVariance := (float64(foregroundCount)*foregroundVar + float64(backgroundCount)*backgroundVar) / totalPixels
		// Convert to uniformity score (0-1, higher is better)
		metrics.RegionUniformity = 1.0 / (1.0 + weightedVariance/255.0)
	}

	return nil
}

// calculateBoundaryAccuracy measures preservation of object boundaries
func calculateBoundaryAccuracy(original, segmented *safe.Mat, metrics *SegmentationMetrics) error {
	rows := original.Rows()
	cols := original.Cols()

	// Calculate gradients in original image to identify edges
	gradientMagnitude := make([][]float64, rows)
	for i := range gradientMagnitude {
		gradientMagnitude[i] = make([]float64, cols)
	}

	// Sobel gradient calculation
	for y := 1; y < rows-1; y++ {
		for x := 1; x < cols-1; x++ {
			// Get neighborhood values
			vals := make([]float64, 9)
			idx := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					val, _ := original.GetUCharAt(y+dy, x+dx)
					vals[idx] = float64(val)
					idx++
				}
			}

			// Sobel operators
			gx := -vals[0] - 2*vals[3] - vals[6] + vals[2] + 2*vals[5] + vals[8]
			gy := -vals[0] - 2*vals[1] - vals[2] + vals[6] + 2*vals[7] + vals[8]

			gradientMagnitude[y][x] = math.Sqrt(gx*gx + gy*gy)
		}
	}

	// Find edges in segmented image
	var edgePixels, accurateEdgePixels int
	edgeThreshold := 30.0 // Gradient magnitude threshold for edge detection

	for y := 1; y < rows-1; y++ {
		for x := 1; x < cols-1; x++ {
			// Check if this is an edge pixel in the original image
			if gradientMagnitude[y][x] > edgeThreshold {
				edgePixels++

				// Check if segmentation preserves this edge
				center, _ := segmented.GetUCharAt(y, x)
				hasEdge := false

				// Check 8-neighborhood for segmentation boundary
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dy == 0 && dx == 0 {
							continue
						}
						neighbor, _ := segmented.GetUCharAt(y+dy, x+dx)
						if (center > 127) != (neighbor > 127) {
							hasEdge = true
							break
						}
					}
					if hasEdge {
						break
					}
				}

				if hasEdge {
					accurateEdgePixels++
				}
			}
		}
	}

	// Calculate boundary accuracy
	if edgePixels > 0 {
		metrics.BoundaryAccuracy = float64(accurateEdgePixels) / float64(edgePixels)
	} else {
		metrics.BoundaryAccuracy = 1.0 // No edges to preserve
	}

	return nil
}

// calculateHausdorffDistance computes maximum boundary discrepancy
func calculateHausdorffDistance(groundTruth, segmented *safe.Mat, metrics *SegmentationMetrics) error {
	// Extract boundary points from both images
	gtBoundary := extractBoundaryPoints(groundTruth)
	segBoundary := extractBoundaryPoints(segmented)

	if len(gtBoundary) == 0 || len(segBoundary) == 0 {
		metrics.HausdorffDistance = 0.0
		return nil
	}

	// Calculate directed Hausdorff distances
	h1 := calculateDirectedHausdorff(gtBoundary, segBoundary)
	h2 := calculateDirectedHausdorff(segBoundary, gtBoundary)

	// Hausdorff distance is the maximum of both directions
	metrics.HausdorffDistance = math.Max(h1, h2)

	return nil
}

// Point represents a 2D point
type Point struct {
	X, Y int
}

// extractBoundaryPoints finds boundary pixels in a binary image
func extractBoundaryPoints(binary *safe.Mat) []Point {
	var boundary []Point
	rows := binary.Rows()
	cols := binary.Cols()

	for y := 1; y < rows-1; y++ {
		for x := 1; x < cols-1; x++ {
			center, _ := binary.GetUCharAt(y, x)
			if center > 127 { // Foreground pixel
				// Check if it's on boundary (has background neighbor)
				isBoundary := false
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if dy == 0 && dx == 0 {
							continue
						}
						neighbor, _ := binary.GetUCharAt(y+dy, x+dx)
						if neighbor <= 127 { // Background neighbor found
							isBoundary = true
							break
						}
					}
					if isBoundary {
						break
					}
				}

				if isBoundary {
					boundary = append(boundary, Point{X: x, Y: y})
				}
			}
		}
	}

	return boundary
}

// calculateDirectedHausdorff computes directed Hausdorff distance
func calculateDirectedHausdorff(set1, set2 []Point) float64 {
	maxDist := 0.0

	for _, p1 := range set1 {
		minDist := math.Inf(1)

		for _, p2 := range set2 {
			dx := float64(p1.X - p2.X)
			dy := float64(p1.Y - p2.Y)
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist < minDist {
				minDist = dist
			}
		}

		if minDist > maxDist {
			maxDist = minDist
		}
	}

	return maxDist
}

// GetMetricsDescription returns human-readable descriptions of the metrics
func (m *SegmentationMetrics) GetMetricsDescription() map[string]string {
	return map[string]string{
		"IoU":                    fmt.Sprintf("Intersection over Union: %.4f (higher is better, 1.0 = perfect)", m.IoU),
		"DiceCoefficient":        fmt.Sprintf("Dice Similarity: %.4f (higher is better, 1.0 = perfect)", m.DiceCoefficient),
		"MisclassificationError": fmt.Sprintf("Misclassification Error: %.4f (lower is better, 0.0 = perfect)", m.MisclassificationError),
		"RegionUniformity":       fmt.Sprintf("Region Uniformity: %.4f (higher is better, 1.0 = perfect)", m.RegionUniformity),
		"BoundaryAccuracy":       fmt.Sprintf("Boundary Accuracy: %.4f (higher is better, 1.0 = perfect)", m.BoundaryAccuracy),
		"HausdorffDistance":      fmt.Sprintf("Hausdorff Distance: %.2f pixels (lower is better, 0.0 = perfect)", m.HausdorffDistance),
	}
}