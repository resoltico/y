package debug

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
)

type ImageDebugInfo struct {
	OriginalURI      fyne.URI
	ExtensionFromURI string
	DetectedFormat   string
	Width            int
	Height           int
	Channels         int
	DataSize         int
	LoadTime         time.Duration
	ProcessingSteps  []string
}

// Global debug toggles (set from main package)
var (
	EnableImageDebug       = true
	EnablePerformanceDebug = true
	EnableMemoryDebug      = true
)

func (dm *Manager) LogImageLoad(info *ImageDebugInfo) {
	if !EnableImageDebug {
		return
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	report := fmt.Sprintf(`Image Load Debug Report:
- Original URI: %s
- Extension from URI: %s
- Detected Format: %s
- Dimensions: %dx%d
- Channels: %d
- Data Size: %d bytes
- Load Time: %v
- Processing Steps: %v`,
		info.OriginalURI.String(),
		info.ExtensionFromURI,
		info.DetectedFormat,
		info.Width,
		info.Height,
		info.Channels,
		info.DataSize,
		info.LoadTime,
		info.ProcessingSteps)

	LogInfo("ImageDebug", report)
}

func (dm *Manager) LogImageFormatMismatch(uri fyne.URI, expectedExt, detectedFormat string) {
	if !EnableImageDebug {
		return
	}
	warning := fmt.Sprintf("Format mismatch detected - URI: %s, Expected: %s, Detected: %s",
		uri.String(), expectedExt, detectedFormat)
	LogWarning("ImageDebug", warning)
}

func (dm *Manager) LogImageProcessing(algorithm string, params map[string]interface{}, processingTime time.Duration) {
	if !EnableImageDebug {
		return
	}
	report := fmt.Sprintf("Image processing completed - Algorithm: %s, Time: %v, Params: %+v",
		algorithm, processingTime, params)
	LogInfo("ImageDebug", report)
}

func (dm *Manager) LogImageConversion(fromFormat, toFormat string, conversionTime time.Duration) {
	if !EnableImageDebug {
		return
	}
	LogInfo("ImageDebug",
		fmt.Sprintf("Image conversion: %s -> %s (Time: %v)", fromFormat, toFormat, conversionTime))
}

func (dm *Manager) LogImageMetrics(psnr, ssim float64, calculationTime time.Duration) {
	if !EnablePerformanceDebug {
		return
	}
	LogInfo("ImageDebug",
		fmt.Sprintf("Image metrics calculated - PSNR: %.2f dB, SSIM: %.4f (Time: %v)",
			psnr, ssim, calculationTime))
}
