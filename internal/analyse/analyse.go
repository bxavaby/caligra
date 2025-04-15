// BYZRA ⸻ internal/analyse/analyse.go
// core analysis logic

package analyse

import (
	"fmt"
	"path/filepath"
	"strings"

	"caligra/internal/config"
	"caligra/internal/formats"
	"caligra/internal/util"
)

// examines a file and returns metadata info
func Analyze(path string) (*AnalysisReport, error) {
	if err := util.ValidatePath(path); err != nil {
		return nil, fmt.Errorf("invalid file: %w", err)
	}

	fileType, err := DetectFile(path)
	if err != nil {
		return nil, fmt.Errorf("file type detection failed: %w", err)
	}

	// format support
	if !formats.IsSupported(fileType.Extension) {
		return nil, fmt.Errorf("unsupported file type: %s", fileType.Extension)
	}

	handler, err := formats.GetHandler(fileType.Format)
	if err != nil {
		return nil, fmt.Errorf("no handler for format %s: %w", fileType.Format, err)
	}

	metadata, err := handler.ExtractMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("metadata extraction failed: %w", err)
	}

	sensitiveFields := identifySensitiveFields(metadata)

	// generate report
	report := &AnalysisReport{
		Path:            path,
		FileType:        fileType,
		Metadata:        metadata,
		SensitiveFields: sensitiveFields,
	}

	return report, nil
}

// finds metadata fields that may contain sensitive information
func identifySensitiveFields(metadata map[string]any) []string {
	var sensitive []string
	profileValues := getProfileValues()

	fmt.Println("DEBUG: Profile values loaded:", profileValues)

	for key, value := range metadata {
		if strings.HasPrefix(key, "_") {
			continue
		}

		strValue := fmt.Sprintf("%v", value)

		if isProfileMetadata(key, strValue, profileValues) {
			fmt.Printf("DEBUG: Skipping profile field: %s = %s\n", key, strValue)
			continue
		}

		if util.IsSensitiveField(key) {
			sensitive = append(sensitive, key)
		}
	}

	return sensitive
}

func getProfileValues() map[string]string {
	profile, err := config.LoadProfile()
	if err != nil {
		// Fallback to default profile
		return config.GetDefaultProfile()
	}
	return profile
}

// does metadata match profile values?
func isProfileMetadata(key string, value string, profileValues map[string]string) bool {
	lowerKey := strings.ToLower(key)

	profileMappings := map[string]string{
		"artist":       "author",
		"author":       "author",
		"creator":      "author",
		"software":     "software",
		"createdate":   "created",
		"datecreated":  "created",
		"copyright":    "organization",
		"organization": "organization",
		"location":     "location",
		"usercomment":  "comment",
		"comment":      "comment",
	}

	profileKey, exists := profileMappings[lowerKey]
	if !exists {
		return false
	}

	profileValue, hasValue := profileValues[profileKey]
	if !hasValue {
		return false
	}

	if profileKey == "created" {
		normalizedValue := strings.ReplaceAll(value, ":", "-")
		normalizedProfile := strings.ReplaceAll(profileValue, ":", "-")

		return normalizedValue == normalizedProfile ||
			strings.HasPrefix(normalizedValue, normalizedProfile)
	}

	return strings.EqualFold(value, profileValue)
}

// analyzes multiple files and returns their reports
func AnalyzeFiles(paths []string) []*AnalysisReport {
	results := make([]*AnalysisReport, 0, len(paths))

	for _, path := range paths {
		info, err := util.GetFileInfo(path)
		if err != nil || info.IsDir() {
			continue
		}

		report, err := Analyze(path)
		if err != nil {
			// error report
			results = append(results, &AnalysisReport{
				Path: path,
				FileType: FileType{
					Format:    "error",
					Extension: filepath.Ext(path),
				},
				Metadata: map[string]any{
					"Error": err.Error(),
				},
			})
		} else {
			results = append(results, report)
		}
	}

	return results
}

// analyzes all supported files in a directory
// func AnalyzeDirectory(dirPath string) ([]*AnalysisReport, error) {
//	entries, err := util.ListDirectory(dirPath)
//	if err != nil {
//		return nil, fmt.Errorf("failed to list directory: %w", err)
//	}

//	var paths []string
//	for _, entry := range entries {
//		if !entry.IsDir() && formats.IsSupported(filepath.Ext(entry.Name())) {
//			paths = append(paths, filepath.Join(dirPath, entry.Name()))
//		}
//	}

//	return AnalyzeFiles(paths), nil
//}
