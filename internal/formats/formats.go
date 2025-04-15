// BYZRA ⸻ internal/formats/formats.go
// format handler interfaces and common functionality

package formats

import (
	"fmt"
	"slices"
	"strings"
)

// defines operations for format-specific metadata handling
type FormatHandler interface {
	// extract all metadata
	ExtractMetadata(path string) (map[string]any, error)

	// remove all metadata
	WipeMetadata(path string) error

	// add profile metadata
	InjectMetadata(path string, profile map[string]string) error

	// verify file integrity after ops
	VerifyIntegrity(path string) bool
}

// appropriate handler for a file format
func GetHandler(format string) (FormatHandler, error) {
	switch format {
	case "image":
		return &ImageHandler{}, nil
	case "audio":
		return &AudioHandler{}, nil
	case "video":
		return &VideoHandler{}, nil
	case "text":
		return &TextHandler{}, nil
	default:
		return nil, fmt.Errorf("no handler for format: %s", format)
	}
}

// all supported extensions by format
var (
	ImageExtensions = []string{"jpg", "jpeg", "png", "gif", "tiff", "svg"}
	AudioExtensions = []string{"mp3", "flac", "opus", "ogg"}
	VideoExtensions = []string{"mp4", "avi"}
	TextExtensions  = []string{"txt", "md", "html"}
)

// list of all supported file extensions
func SupportedFormats() []string {
	allFormats := []string{}
	allFormats = append(allFormats, ImageExtensions...)
	allFormats = append(allFormats, AudioExtensions...)
	allFormats = append(allFormats, VideoExtensions...)
	allFormats = append(allFormats, TextExtensions...)
	return allFormats
}

// checks if a file extension is supported
func IsSupported(extension string) bool {
	// remove leading dot if present
	if len(extension) > 0 && extension[0] == '.' {
		extension = extension[1:]
	}

	extension = strings.ToLower(extension)

	return slices.Contains(SupportedFormats(), extension)
}

// format category for a given extension
func GetFormatType(extension string) (string, error) {
	// remove leading dot if present
	if len(extension) > 0 && extension[0] == '.' {
		extension = extension[1:]
	}

	extension = strings.ToLower(extension)

	if slices.Contains(ImageExtensions, extension) {
		return "image", nil
	}

	if slices.Contains(AudioExtensions, extension) {
		return "audio", nil
	}

	if slices.Contains(VideoExtensions, extension) {
		return "video", nil
	}

	if slices.Contains(TextExtensions, extension) {
		return "text", nil
	}

	return "", fmt.Errorf("unsupported extension: %s", extension)
}
