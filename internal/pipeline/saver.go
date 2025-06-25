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
	logger        Logger
	timingTracker TimingTracker
}

func (s *imageSaver) SaveToWriter(writer io.Writer, imageData *ImageData, format string) error {
	if imageData == nil {
		return fmt.Errorf("no image data to save")
	}

	ctx := s.timingTracker.StartTiming("save_to_writer")
	defer s.timingTracker.EndTiming(ctx)

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
				// No extension provided, default to PNG
				saveFormat = "png"
			}
		} else {
			saveFormat = imageData.Format
		}
	}

	if saveFormat == "" {
		saveFormat = "png"
	}

	s.logger.Debug("ImageSaver", "saving image", map[string]interface{}{
		"format": saveFormat,
		"width":  imageData.Width,
		"height": imageData.Height,
	})

	var err error
	switch saveFormat {
	case "jpeg":
		err = jpeg.Encode(writer, img, &jpeg.Options{Quality: 95})
	case "png":
		err = png.Encode(writer, img)
	case "tiff", "bmp":
		s.logger.Warning("ImageSaver", "format not supported, using PNG", map[string]interface{}{
			"requested_format": strings.ToUpper(saveFormat),
		})
		err = png.Encode(writer, img)
	default:
		err = png.Encode(writer, img)
	}

	if err != nil {
		s.logger.Error("ImageSaver", err, map[string]interface{}{
			"format": saveFormat,
		})
		return err
	}

	s.logger.Info("ImageSaver", "image saved", map[string]interface{}{
		"format": saveFormat,
	})

	return nil
}

func (s *imageSaver) SaveToPath(path string, imageData *ImageData) error {
	return fmt.Errorf("file path saving not implemented")
}
