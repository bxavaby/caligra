// BYZRA ⸻ internal/formats/audio.go
// audio format handlers

package formats

import (
	"fmt"
	"os/exec"
	"strings"

	"caligra/internal/util"
)

// implements FormatHandler for audio files
type AudioHandler struct{}

// extracts metadata from audio files
func (h *AudioHandler) ExtractMetadata(path string) (map[string]any, error) {
	data, err := util.ExifToolExtract(path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract audio metadata: %w", err)
	}

	// parse the JSON response into a map
	metadata, err := util.ParseExifToolOutput(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse audio metadata: %w", err)
	}

	return metadata, nil
}

// removes all metadata from audio files
func (h *AudioHandler) WipeMetadata(path string) error {
	err := util.ExifToolRemove(path)
	if err != nil {
		return fmt.Errorf("failed to wipe audio metadata: %w", err)
	}
	return nil
}

// adds profile metadata to audio files
func (h *AudioHandler) InjectMetadata(path string, profile map[string]string) error {
	for key, value := range profile {
		// map profile keys to audio metadata tags
		tag := mapProfileKeyToAudioTag(key)
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

// ensures the audio file is still valid
func (h *AudioHandler) VerifyIntegrity(path string) bool {
	// for audio, use ffmpeg to check validity
	cmd := exec.Command("ffmpeg", "-v", "error", "-i", path, "-f", "null", "-")
	err := cmd.Run()
	return err == nil
}

// maps profile keys to audio metadata tags
func mapProfileKeyToAudioTag(key string) string {
	switch strings.ToLower(key) {
	case "author":
		return "Artist"
	case "software":
		return "EncodedBy"
	case "created":
		return "Date"
	case "organization":
		return "Publisher"
	case "location":
		return "Composer" // repurposing this field
	case "comment":
		return "Comment"
	default:
		return ""
	}
}
