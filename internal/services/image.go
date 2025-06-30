package services

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"time"

	"otsu-obliterator/internal/models"
	"otsu-obliterator/internal/opencv/conversion"
	"otsu-obliterator/internal/opencv/memory"
	"otsu-obliterator/internal/opencv/safe"

	"fyne.io/fyne/v2"
	"gocv.io/x/gocv"
)

// ImageService handles image loading, saving, and format conversions
type ImageService struct {
	memoryManager *memory.Manager
	repository    *models.ImageRepository
}

// NewImageService creates a new image service
func NewImageService(memMgr *memory.Manager, repo *models.ImageRepository) *ImageService {
	return &ImageService{
		memoryManager: memMgr,
		repository:    repo,
	}
}

// LoadImage loads an image from a URI reader
func (is *ImageService) LoadImage(ctx context.Context, reader fyne.URIReadCloser) (*models.ImageData, error) {
	defer reader.Close()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	startTime := time.Now()
	originalURI := reader.URI()
	
	// Read all data into buffer
	bufferedReader := bufio.NewReader(reader)
	data, err := io.ReadAll(bufferedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Determine format from URI extension
	uriExtension := strings.ToLower(filepath.Ext(originalURI.Path()))
	
	// Decode with standard library
	img, standardFormat, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create Mat from standard image
	mat, err := conversion.ImageToMat(img)
	if err != nil {
		return nil, fmt.Errorf("failed to convert image to Mat: %w", err)
	}

	// Determine final format
	actualFormat := is.determineFormat(uriExtension, standardFormat)
	bounds := img.Bounds()

	imageData := &models.ImageData{
		Image:       img,
		Mat:         mat,
		Width:       bounds.Dx(),
		Height:      bounds.Dy(),
		Channels:    mat.Channels(),
		Format:      actualFormat,
		OriginalURI: originalURI,
		LoadTime:    time.Now(),
		Metadata: models.ImageMetadata{
			FileSize:    int64(len(data)),
			ColorSpace:  is.determineColorSpace(mat),
			BitDepth:    8, // Standard for most images
			Compression: actualFormat,
			Software:    "Otsu Obliterator",
		},
	}

	// Store in repository
	is.repository.SetOriginalImage(imageData)

	loadDuration := time.Since(startTime)
	imageData.ProcessTime = loadDuration

	return imageData, nil
}

// SaveImage saves an image to a URI writer
func (is *ImageService) SaveImage(ctx context.Context, writer fyne.URIWriteCloser, imageData *models.ImageData, format string) error {
	defer writer.Close()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if imageData == nil || imageData.Image == nil {
		return fmt.Errorf("no image data to save")
	}

	// Determine save format
	saveFormat := format
	if saveFormat == "" {
		ext := strings.ToLower(writer.URI().Extension())
		saveFormat = is.determineFormat(ext, imageData.Format)
	}

	return is.saveToWriter(writer, imageData.Image, saveFormat)
}

// SaveImageToWriter saves an image to a generic writer
func (is *ImageService) SaveImageToWriter(writer io.Writer, imageData *models.ImageData, format string) error {
	if imageData == nil || imageData.Image == nil {
		return fmt.Errorf("no image data to save")
	}

	return is.saveToWriter(writer, imageData.Image, format)
}

// ConvertImageFormat converts an image to a different format
func (is *ImageService) ConvertImageFormat(ctx context.Context, imageData *models.ImageData, targetFormat string) (*models.ImageData, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if imageData == nil {
		return nil, fmt.Errorf("no image data provided")
	}

	// Create new image data with converted format
	convertedData := &models.ImageData{
		Image:       imageData.Image,
		Mat:         imageData.Mat, // Reuse Mat
		Width:       imageData.Width,
		Height:      imageData.Height,
		Channels:    imageData.Channels,
		Format:      targetFormat,
		OriginalURI: imageData.OriginalURI,
		LoadTime:    imageData.LoadTime,
		ProcessTime: imageData.ProcessTime,
		Metadata:    imageData.Metadata,
	}

	// Update metadata
	convertedData.Metadata.Compression = targetFormat

	return convertedData, nil
}

// ResizeImage resizes an image to new dimensions
func (is *ImageService) ResizeImage(ctx context.Context, imageData *models.ImageData, newWidth, newHeight int) (*models.ImageData, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if imageData == nil || imageData.Mat == nil {
		return nil, fmt.Errorf("no image data provided")
	}

	// Resize the Mat
	resizedMat, err := conversion.ResizeMat(imageData.Mat, newWidth, newHeight, gocv.InterpolationLinear)
	if err != nil {
		return nil, fmt.Errorf("failed to resize Mat: %w", err)
	}

	select {
	case <-ctx.Done():
		resizedMat.Close()
		return nil, ctx.Err()
	default:
	}

	// Convert Mat back to image
	resizedImage, err := conversion.MatToImage(resizedMat)
	if err != nil {
		resizedMat.Close()
		return nil, fmt.Errorf("failed to convert resized Mat to image: %w", err)
	}

	// Create new image data
	resizedData := &models.ImageData{
		Image:       resizedImage,
		Mat:         resizedMat,
		Width:       newWidth,
		Height:      newHeight,
		Channels:    resizedMat.Channels(),
		Format:      imageData.Format,
		OriginalURI: imageData.OriginalURI,
		LoadTime:    time.Now(),
		ProcessTime: 0,
		Metadata:    imageData.Metadata,
	}

	// Update metadata
	resizedData.Metadata.FileSize = int64(newWidth * newHeight * resizedData.Channels)

	return resizedData, nil
}

// CropImage crops an image to a rectangular region
func (is *ImageService) CropImage(ctx context.Context, imageData *models.ImageData, x, y, width, height int) (*models.ImageData, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if imageData == nil || imageData.Mat == nil {
		return nil, fmt.Errorf("no image data provided")
	}

	// Validate crop parameters
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid crop parameters")
	}

	if x+width > imageData.Width || y+height > imageData.Height {
		return nil, fmt.Errorf("crop region exceeds image bounds")
	}

	// Crop the Mat
	croppedMat, err := conversion.CropMat(imageData.Mat, x, y, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to crop Mat: %w", err)
	}

	select {
	case <-ctx.Done():
		croppedMat.Close()
		return nil, ctx.Err()
	default:
	}

	// Convert Mat back to image
	croppedImage, err := conversion.MatToImage(croppedMat)
	if err != nil {
		croppedMat.Close()
		return nil, fmt.Errorf("failed to convert cropped Mat to image: %w", err)
	}

	// Create new image data
	croppedData := &models.ImageData{
		Image:       croppedImage,
		Mat:         croppedMat,
		Width:       width,
		Height:      height,
		Channels:    croppedMat.Channels(),
		Format:      imageData.Format,
		OriginalURI: imageData.OriginalURI,
		LoadTime:    time.Now(),
		ProcessTime: 0,
		Metadata:    imageData.Metadata,
	}

	// Update metadata
	croppedData.Metadata.FileSize = int64(width * height * croppedData.Channels)

	return croppedData, nil
}

// GetImageInfo returns detailed information about an image
func (is *ImageService) GetImageInfo(imageData *models.ImageData) ImageInfo {
	if imageData == nil {
		return ImageInfo{}
	}

	info := ImageInfo{
		Width:     imageData.Width,
		Height:    imageData.Height,
		Channels:  imageData.Channels,
		Format:    imageData.Format,
		FileSize:  imageData.Metadata.FileSize,
		BitDepth:  imageData.Metadata.BitDepth,
		LoadTime:  imageData.LoadTime,
		HasMat:    imageData.Mat != nil,
	}

	if imageData.Mat != nil {
		properties := conversion.GetMatProperties(imageData.Mat)
		info.MatType = properties.DataType
		info.MatEmpty = properties.Empty
	}

	return info
}

// ImageInfo contains detailed information about an image
type ImageInfo struct {
	Width     int
	Height    int
	Channels  int
	Format    string
	FileSize  int64
	BitDepth  int
	LoadTime  time.Time
	HasMat    bool
	MatType   string
	MatEmpty  bool
}

// saveToWriter handles the actual saving to a writer
func (is *ImageService) saveToWriter(writer io.Writer, img image.Image, format string) error {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 95})
	case "png":
		return png.Encode(writer, img)
	default:
		// Default to PNG for unknown formats
		return png.Encode(writer, img)
	}
}

// determineFormat determines the appropriate format based on extension and detected format
func (is *ImageService) determineFormat(extension, detectedFormat string) string {
	switch extension {
	case ".jpg", ".jpeg":
		return "jpeg"
	case ".png":
		return "png"
	case ".bmp":
		return "bmp"
	case ".tiff", ".tif":
		return "tiff"
	default:
		if detectedFormat != "" {
			return detectedFormat
		}
		return "png" // Default format
	}
}

// determineColorSpace determines the color space of a Mat
func (is *ImageService) determineColorSpace(mat *safe.Mat) string {
	if mat == nil {
		return "unknown"
	}

	switch mat.Channels() {
	case 1:
		return "grayscale"
	case 3:
		return "BGR"
	case 4:
		return "BGRA"
	default:
		return "unknown"
	}
}

// ValidateImageFormat checks if a format is supported
func (is *ImageService) ValidateImageFormat(format string) bool {
	supportedFormats := map[string]bool{
		"jpeg": true,
		"jpg":  true,
		"png":  true,
		"bmp":  true,
		"tiff": true,
		"tif":  true,
	}

	return supportedFormats[strings.ToLower(format)]
}

// GetSupportedFormats returns list of supported image formats
func (is *ImageService) GetSupportedFormats() []string {
	return []string{"jpeg", "jpg", "png", "bmp", "tiff", "tif"}
}

// Cleanup releases resources
func (is *ImageService) Cleanup() {
	if is.repository != nil {
		is.repository.Shutdown()
	}
}