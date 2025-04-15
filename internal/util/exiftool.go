// BYZRA ⸻ internal/util/exiftool.go
// exiftool wrapper for metadata operations

package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// runs exiftool to extract all metadata as JSON
func ExifToolExtract(path string) (string, error) {
	return SpinWhile("[~] Analyzing metadata", func() (string, error) {
		cmd := exec.Command("exiftool", "-json", path)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		return out.String(), err
	})
}

// runs exiftool to remove all metadata
func ExifToolRemove(path string) error {
	_, err := SpinWhile("[~] Removing metadata", func() (string, error) {
		cmd := exec.Command("exiftool", "-all=", "-overwrite_original", path)
		err := cmd.Run()
		return "", err
	})
	return err
}

// parses JSON output from exiftool into a map
func ParseExifToolOutput(output string) (map[string]any, error) {
	// trim whitespace
	output = strings.TrimSpace(output)

	// ExifTool outputs an array of objects, but we only care about the first one
	var results []map[string]any

	// parse JSON
	err := json.Unmarshal([]byte(output), &results)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ExifTool JSON: %w", err)
	}

	// ensure we have results
	if len(results) == 0 {
		return make(map[string]any), nil // return empty map, not an error
	}

	return results[0], nil
}

// returns names of potentially sensitive metadata fields
func GetSensitiveMetadataFields() []string {
	return []string{
		"GPSLatitude", "GPSLongitude", "GPSPosition", "Location",
		"Author", "Creator", "Artist", "Owner", "Copyright",
		"Email", "CameraSerialNumber", "SerialNumber", "DeviceID",
		"OriginalFilename", "FileName", "UserName", "HostComputer",
		"Make", "Model", "Software", "CreateDate", "ModifyDate",
	}
}

// returns true if the field might contain sensitive data
func IsSensitiveField(fieldName string) bool {
	fieldName = strings.ToLower(fieldName)
	sensitiveFields := GetSensitiveMetadataFields()

	for _, sensitive := range sensitiveFields {
		if strings.ToLower(sensitive) == fieldName {
			return true
		}
		// check for partial matches for compound fields
		if strings.Contains(fieldName, strings.ToLower(sensitive)) {
			return true
		}
	}

	return false
}
