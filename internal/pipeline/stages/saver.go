package stages

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	"fyne.io/fyne/v2"
	"otsu-obliterator/internal/debug"
	"otsu-obliterator/internal/pipeline"
)

type Saver struct {
	debugManager *debug.Manager
}

func NewSaver(debugMgr *debug.Manager) *Saver {
	return &Saver{
		debugManager: debugMgr,
	}
}

func (s *Saver) SaveImage(writer fyne.URIWriteCloser, imageData *pipeline.ImageData) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	img, ok := imageData.Image.(image.Image)
	if !ok {
		return fmt.Errorf("image data is not a valid image")
	}

	uri := writer.URI()
	ext := strings.ToLower(uri.Extension())

	var saveFormat string
	switch ext {
	case ".jpg", ".jpeg":
		saveFormat = "jpeg"
	case ".png":
		saveFormat = "png"
	default:
		if imageData.Format != "" {
			saveFormat = imageData.Format
		} else {
			saveFormat = "png"
		}
	}

	s.debugManager.LogInfo("PipelineSaver", 
		fmt.Sprintf("Saving image as %s format", saveFormat))

	switch saveFormat {
	case "jpeg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: 95})
	case "png":
		return png.Encode(writer, img)
	case "tiff", "bmp":
		s.debugManager.LogWarning("PipelineSaver", 
			fmt.Sprintf("%s encoding not supported, saving as PNG", strings.ToUpper(saveFormat)))
		return png.Encode(writer, img)
	default:
		return png.Encode(writer, img)
	}
}