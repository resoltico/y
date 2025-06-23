package debug

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
)

type FormatDetection struct {
	URI               fyne.URI
	URIScheme         string
	URIPath           string
	URIExtension      string
	URIMimeType       string
	StandardLibFormat string
	OpenCVSupported   bool
	FinalFormat       string
	DataSize          int
	FirstBytes        []byte // First 16 bytes for format signature detection
}

func (dm *Manager) LogFormatDetection(detection *FormatDetection) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	var firstBytesHex string
	if len(detection.FirstBytes) > 0 {
		hexBytes := make([]string, len(detection.FirstBytes))
		for i, b := range detection.FirstBytes {
			hexBytes[i] = fmt.Sprintf("%02X", b)
		}
		firstBytesHex = strings.Join(hexBytes, " ")
	}

	report := fmt.Sprintf(`Format Detection Analysis:
- URI: %s
- URI Scheme: %s  
- URI Path: %s
- URI Extension: %s
- URI MimeType: %s
- Standard Library Format: %s
- OpenCV Supported: %t
- Final Format: %s
- Data Size: %d bytes
- First 16 bytes (hex): %s
- Format Signature Analysis: %s`,
		detection.URI.String(),
		detection.URIScheme,
		detection.URIPath,
		detection.URIExtension,
		detection.URIMimeType,
		detection.StandardLibFormat,
		detection.OpenCVSupported,
		detection.FinalFormat,
		detection.DataSize,
		firstBytesHex,
		dm.analyzeFormatSignature(detection.FirstBytes))

	LogInfo("FormatDebug", report)
}

func (dm *Manager) analyzeFormatSignature(data []byte) string {
	if len(data) < 4 {
		return "insufficient data for signature analysis"
	}

	// Check common format signatures
	signatures := map[string]string{
		"89504E47": "PNG signature detected",
		"FFD8FF":   "JPEG signature detected",
		"49492A00": "TIFF little-endian signature detected",
		"4D4D002A": "TIFF big-endian signature detected",
		"424D":     "BMP signature detected",
		"47494638": "GIF signature detected",
		"52494646": "RIFF container (possibly WebP) detected",
	}

	// Convert first few bytes to hex for comparison
	if len(data) >= 8 {
		hex8 := fmt.Sprintf("%02X%02X%02X%02X%02X%02X%02X%02X",
			data[0], data[1], data[2], data[3], data[4], data[5], data[6], data[7])

		for sig, desc := range signatures {
			if strings.HasPrefix(hex8, sig) {
				return desc
			}
		}
	}

	if len(data) >= 4 {
		hex4 := fmt.Sprintf("%02X%02X%02X%02X", data[0], data[1], data[2], data[3])

		for sig, desc := range signatures {
			if strings.HasPrefix(hex4, sig) {
				return desc
			}
		}
	}

	if len(data) >= 3 {
		hex3 := fmt.Sprintf("%02X%02X%02X", data[0], data[1], data[2])

		for sig, desc := range signatures {
			if strings.HasPrefix(hex3, sig) {
				return desc
			}
		}
	}

	if len(data) >= 2 {
		hex2 := fmt.Sprintf("%02X%02X", data[0], data[1])

		for sig, desc := range signatures {
			if strings.HasPrefix(hex2, sig) {
				return desc
			}
		}
	}

	return "unknown format signature"
}

func (dm *Manager) LogExtensionMimeTypeMismatch(uri fyne.URI, expectedFromExt, detectedMime string) {
	warning := fmt.Sprintf("Extension/MimeType mismatch - URI: %s, Expected from extension: %s, Detected MimeType: %s",
		uri.String(), expectedFromExt, detectedMime)
	LogWarning("FormatDebug", warning)
}

func (dm *Manager) LogStandardLibDecodingResult(format string, success bool, errorMsg string) {
	status := "success"
	if !success {
		status = fmt.Sprintf("failed: %s", errorMsg)
	}

	LogInfo("FormatDebug", fmt.Sprintf("Standard library decoding - Format: %s, Status: %s", format, status))
}

func (dm *Manager) LogOpenCVDecodingResult(success bool, matChannels int, errorMsg string) {
	status := "success"
	if !success {
		status = fmt.Sprintf("failed: %s", errorMsg)
	}

	LogInfo("FormatDebug", fmt.Sprintf("OpenCV IMDecode - Status: %s, Channels: %d", status, matChannels))
}
