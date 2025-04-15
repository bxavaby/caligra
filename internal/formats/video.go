// BYZRA ⸻ internal/formats/video.go
// video format handlers

package formats

import (
	"fmt"
	"os/exec"
	"strings"

	"caligra/internal/util"
)

// implements FormatHandler for video files
type VideoHandler struct{}

// extracts metadata from video files
func (h *VideoHandler) ExtractMetadata(path string) (map[string]interface{}, error) {
	data, err := util.ExifToolExtract(path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract video metadata: %w", err)
	}

	// parse the JSON response into a map
	metadata, err := util.ParseExifToolOutput(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse video metadata: %w", err)
	}

	return metadata, nil
}

// removes all metadata from video files
func (h *VideoHandler) WipeMetadata(path string) error {
	err := util.ExifToolRemove(path)
	if err != nil {
		return fmt.Errorf("failed to wipe video metadata: %w", err)
	}
	return nil
}

// adds profile metadata to video files
func (h *VideoHandler) InjectMetadata(path string, profile map[string]string) error {
	for key, value := range profile {
		// map profile keys to video metadata tags
		tag := mapProfileKeyToVideoTag(key)
		if tag == "" {
			continue // Skip unmapped keys
		}

		cmd := exec.Command("exiftool", fmt.Sprintf("-%s=%s", tag, value), "-overwrite_original", path)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to inject %s metadata: %w", key, err)
		}
	}
	return nil
}

// ensures the video file is still valid
func (h *VideoHandler) VerifyIntegrity(path string) bool {
	// for video, use ffmpeg to check validity
	cmd := exec.Command("ffmpeg", "-v", "error", "-i", path, "-f", "null", "-")
	err := cmd.Run()
	return err == nil
}

// maps profile keys to video metadata tags
func mapProfileKeyToVideoTag(key string) string {
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
		return "Comment"
	default:
		return ""
	}
}
