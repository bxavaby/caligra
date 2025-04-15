// BYZRA ⸻ internal/wipe/verify.go
// integrity verification for processed files

package wipe

import (
	"fmt"
	"os"

	"caligra/internal/analyse"
	"caligra/internal/formats"
	"caligra/internal/util"
)

// results of a file verification
type VerificationResult struct {
	Success          bool
	FileIntact       bool
	MetadataRemoved  bool
	ProfileInjected  bool
	RemainingFields  []string
	MissingFields    []string
	ValidationErrors []string
}

// checks if a file is intact and properly sanitized
func VerifyFile(path string, expectedProfile map[string]string) (*VerificationResult, error) {
	result := &VerificationResult{
		ValidationErrors: []string{},
	}

	if _, err := os.Stat(path); err != nil {
		return result, fmt.Errorf("file not found: %w", err)
	}

	fileType, err := analyse.DetectFile(path)
	if err != nil {
		return result, fmt.Errorf("file type detection failed: %w", err)
	}

	handler, err := formats.GetHandler(fileType.Format)
	if err != nil {
		return result, fmt.Errorf("no handler for format %s: %w", fileType.Format, err)
	}

	result.FileIntact = handler.VerifyIntegrity(path)
	if !result.FileIntact {
		result.ValidationErrors = append(result.ValidationErrors, "File integrity check failed")
		return result, nil
	}

	report, err := analyse.Analyze(path)
	if err != nil {
		return result, fmt.Errorf("failed to verify metadata: %w", err)
	}

	result.RemainingFields = report.SensitiveFields
	result.MetadataRemoved = len(result.RemainingFields) == 0

	if !result.MetadataRemoved {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Found %d sensitive fields that should have been removed",
				len(result.RemainingFields)))
	}

	if expectedProfile != nil {
		result.MissingFields = verifyProfileFields(report.Metadata, expectedProfile)
		result.ProfileInjected = len(result.MissingFields) == 0

		if !result.ProfileInjected {
			result.ValidationErrors = append(result.ValidationErrors,
				fmt.Sprintf("Profile injection incomplete (%d fields missing)",
					len(result.MissingFields)))
		}
	} else {
		result.ProfileInjected = true
	}

	// overall success
	result.Success = result.FileIntact && result.MetadataRemoved && result.ProfileInjected

	return result, nil
}

// all profile fields were injected properly
func verifyProfileFields(metadata map[string]any, profile map[string]string) []string {
	var missing []string

	// for each profile field, check if it exists in the metadata
	for key, expectedValue := range profile {
		// skip empty fields
		if expectedValue == "" {
			continue
		}

		found := false
		for metaKey, metaValue := range metadata {
			metaValueStr, ok := metaValue.(string)
			if ok && util.KeysMatch(metaKey, key) && metaValueStr == expectedValue {
				found = true
				break
			}
		}

		if !found {
			missing = append(missing, key)
		}
	}

	return missing
}

// user-friendly report of the verification
func FormatVerificationResult(result *VerificationResult) string {
	if result.Success {
		return util.NSH.Render("✓ File successfully processed and verified")
	}

	var message string

	if !result.FileIntact {
		message += util.LBL.Render("[!] File integrity check failed. File may be corrupted.\n")
	}

	if !result.MetadataRemoved {
		message += util.LBL.Render(fmt.Sprintf(
			"[!] Found %d remaining sensitive fields that were not removed.\n",
			len(result.RemainingFields)))

		for _, field := range result.RemainingFields {
			message += util.SUB.Render(fmt.Sprintf("  • %s\n", field))
		}
	}

	if !result.ProfileInjected {
		message += util.LBL.Render(fmt.Sprintf(
			"[!] Profile injection incomplete (%d fields missing).\n",
			len(result.MissingFields)))

		for _, field := range result.MissingFields {
			message += util.SUB.Render(fmt.Sprintf("  • %s\n", field))
		}
	}

	return message
}
