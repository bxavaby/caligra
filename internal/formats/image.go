// BYZRA ⸻ internal/formats/image.go
// image format handler implementation

package formats

import (
	"fmt"
	"os/exec"
	"strings"

	"caligra/internal/util"
)

// implements FormatHandler for image files
type ImageHandler struct{}

// extracts metadata from image files
func (h *ImageHandler) ExtractMetadata(path string) (map[string]any, error) {
	data, err := util.ExifToolExtract(path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract image metadata: %w", err)
	}

	// parse the JSON response into a map
	metadata, err := util.ParseExifToolOutput(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image metadata: %w", err)
	}

	return metadata, nil
}

// removes all metadata from image files
func (h *ImageHandler) WipeMetadata(path string) error {
	err := util.ExifToolRemove(path)
	if err != nil {
		return fmt.Errorf("failed to wipe image metadata: %w", err)
	}
	return nil
}

// adds profile metadata to image files
func (h *ImageHandler) InjectMetadata(path string, profile map[string]string) error {
	for key, value := range profile {
		// map profile keys to ExifTool tags
		tag := mapProfileKeyToExifTag(key)
		if tag == "" {
			continue // skip unmapped keys
		}

		cmd := exec.Command("exiftool", fmt.Sprintf("-%s=%s", tag, value), "-overwrite_original", path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to inject %s metadata: %w", key, err)
		}
	}
	return nil
}

// ensures the image is still valid after modification
func (h *ImageHandler) VerifyIntegrity(path string) bool {
	// for images, use identify from ImageMagick
	cmd := exec.Command("identify", path)
	err := cmd.Run()
	return err == nil
}

// maps profile keys to ExifTool tag names
func mapProfileKeyToExifTag(key string) string {
	switch strings.ToLower(key) {
	case "author":
		return "Artist"
	case "software":
		return "Software"
	case "created":
		return "CreateDate"
	case "organization":
		return "Copyright"
	case "location":
		return "Location"
	case "comment":
		return "UserComment"
	default:
		return ""
	}
}
