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

	// avoid flagging profile values
	profileValues := getProfileValues()

	for key, value := range metadata {
		if strings.HasPrefix(key, "_") {
			continue
		}

		if isProfileMetadata(key, fmt.Sprintf("%v", value), profileValues) {
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
	fieldMappings := map[string]string{
		"Artist":       "author",
		"Author":       "author",
		"Creator":      "author",
		"Software":     "software",
		"CreateDate":   "created",
		"DateCreated":  "created",
		"Copyright":    "organization",
		"Organization": "organization",
		"Location":     "location",
		"UserComment":  "comment",
		"Comment":      "comment",
	}

	if profileKey, ok := fieldMappings[key]; ok {
		if profileVal, exists := profileValues[profileKey]; exists && value == profileVal {
			return true
		}
	}

	return false
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
