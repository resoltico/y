package pipeline

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"fyne.io/fyne/v2"
)

type imageSaver struct {
	debugManager DebugManager
}

func (s *imageSaver) SaveToWriter(writer io.Writer, imageData *ImageData, format string) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	img, ok := imageData.Image.(image.Image)
	if !ok {
		return fmt.Errorf("image data is not a valid image")
	}

	saveFormat := format
	if saveFormat == "" {
		if uriWriter, ok := writer.(fyne.URIWriteCloser); ok {
			ext := strings.ToLower(uriWriter.URI().Extension())
			switch ext {
			case ".jpg", ".jpeg":
				saveFormat = "jpeg"
			case ".png":
				saveFormat = "png"
			default:
				saveFormat = imageData.Format
			}
		} else {
			saveFormat = imageData.Format
		}
	}

	if saveFormat == "" {
		saveFormat = "png"
	}

	s.debugManager.LogInfo("ImageSaver", 
		fmt.Sprintf("Saving image as %s format", saveFormat))

	switch saveFormat {
	case "jpeg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 95})
	case "png":
		return png.Encode(writer, img)
	case "tiff", "bmp":
		s.debugManager.LogWarning("ImageSaver", 
			fmt.Sprintf("%s encoding not supported, saving as PNG", strings.ToUpper(saveFormat)))
		return png.Encode(writer, img)
	default:
		return png.Encode(writer, img)
	}
}

func (s *imageSaver) SaveToPath(path string, imageData *ImageData) error {
	return fmt.Errorf("file path saving not implemented")
}
