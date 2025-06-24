package components

import "image"

type ImageType int

const (
	ImageTypeOriginal ImageType = iota
	ImageTypePreview
)

type ImageDisplayUpdate struct {
	Type  ImageType
	Image image.Image
}

type ParameterPanelUpdate struct {
	Algorithm  string
	Parameters map[string]interface{}
}

type StatusUpdate struct {
	Status string
}

type ProgressUpdate struct {
	Progress float64
}

type MetricsUpdate struct {
	PSNR float64
	SSIM float64
}