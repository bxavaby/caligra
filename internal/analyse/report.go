// BYZRA ⸻ internal/analyse/report.go
// format analysis reports

package analyse

import (
	"fmt"
	"sort"
	"strings"

	"caligra/internal/util"
)

// result of file metadata analysis
type AnalysisReport struct {
	Path            string
	FileType        FileType
	Metadata        map[string]any
	SensitiveFields []string
}

func GenerateReport(report *AnalysisReport) string {
	var sb strings.Builder

	// info header
	sb.WriteString(util.NSH.Render(fmt.Sprintf("File: %s\n", report.Path)))
	sb.WriteString(util.NSH.Render(fmt.Sprintf("Type: %s (%s)\n\n", report.FileType.Format, report.FileType.MimeType)))

	// no metadata
	if len(report.Metadata) == 0 {
		sb.WriteString(util.LBL.Render("✓ No metadata detected\n"))
		return sb.String()
	}

	sb.WriteString(util.LBL.Render("Detected Metadata:\n"))

	// sorted keys for consistent output
	keys := make([]string, 0, len(report.Metadata))
	for k := range report.Metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// process metadata fields
	sensitiveCount := 0
	for _, key := range keys {
		value := report.Metadata[key]

		// skip internal fields
		if strings.HasPrefix(key, "_") || strings.HasPrefix(key, "File") {
			continue
		}

		// format value
		valueStr := formatValue(value)
		if valueStr == "" {
			continue
		}

		// is field sensitive
		isSensitive := isSensitiveField(key, report.SensitiveFields)
		if isSensitive {
			sensitiveCount++
			sb.WriteString(fmt.Sprintf(" %s %s: %s\n",
				util.ORN.Render("!"),
				util.NSH.Render(key),
				util.NSH.Render(valueStr)))
		} else {
			sb.WriteString(fmt.Sprintf(" %s %s: %s\n",
				util.ORN.Render("•"),
				util.NSH.Render(key),
				valueStr))
		}
	}

	// summary and recommendation
	sb.WriteString("\n")
	if sensitiveCount > 0 {
		sb.WriteString(util.BRH.Render(fmt.Sprintf(
			"[!] Found %d potentially sensitive metadata fields.\n", sensitiveCount)))
		sb.WriteString(util.BRH.Render("[!] Consider using 'caligra wipe' to remove metadata.\n"))
	} else {
		sb.WriteString(util.LBL.Render("✓ No sensitive metadata detected\n"))
	}

	return sb.String()
}

// creates a machine-readable report
func GenerateSimplifiedReport(report *AnalysisReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("file: %s\n", report.Path))
	sb.WriteString(fmt.Sprintf("format: %s\n", report.FileType.Format))
	sb.WriteString(fmt.Sprintf("mimetype: %s\n", report.FileType.MimeType))

	sensitiveCount := 0
	for k, v := range report.Metadata {
		if strings.HasPrefix(k, "_") || strings.HasPrefix(k, "File") {
			continue
		}

		valueStr := formatValue(v)
		if valueStr == "" {
			continue
		}

		isSensitive := isSensitiveField(k, report.SensitiveFields)
		if isSensitive {
			sensitiveCount++
			sb.WriteString(fmt.Sprintf("sensitive:%s: %s\n", k, valueStr))
		} else {
			sb.WriteString(fmt.Sprintf("metadata:%s: %s\n", k, valueStr))
		}
	}

	sb.WriteString(fmt.Sprintf("sensitive_count: %d\n", sensitiveCount))

	return sb.String()
}

// converts a metadata value to string representation
func formatValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		if v == "" {
			return ""
		}
		return v
	case []any:
		if len(v) == 0 {
			return ""
		}
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if str := formatValue(item); str != "" {
				parts = append(parts, str)
			}
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		if len(v) == 0 {
			return ""
		}
		parts := make([]string, 0, len(v))
		for k, val := range v {
			if str := formatValue(val); str != "" {
				parts = append(parts, fmt.Sprintf("%s:%s", k, str))
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
}

// field is in the sensitive list checker
func isSensitiveField(field string, sensitiveFields []string) bool {
	lowerField := strings.ToLower(field)

	for _, sensitive := range sensitiveFields {
		if strings.ToLower(sensitive) == lowerField {
			return true
		}
		if strings.Contains(lowerField, strings.ToLower(sensitive)) {
			return true
		}
	}

	// + common sensitive fields
	commonSensitive := []string{
		"gps", "location", "author", "creator", "owner", "copyright",
		"email", "serial", "device", "username", "computer", "date",
	}

	for _, term := range commonSensitive {
		if strings.Contains(lowerField, term) {
			return true
		}
	}

	return false
}
