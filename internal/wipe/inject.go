// BYZRA ⸻ internal/wipe/inject.go
// profile metadata injection

package wipe

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"caligra/internal/analyse"
	"caligra/internal/config"
	"caligra/internal/formats"
	"caligra/internal/util"
)

// results of profile injection
type ProfileInjectionResult struct {
	Success      bool
	FieldsAdded  []string
	FieldsFailed []string
	Profile      map[string]string
}

// applies profile metadata 2 a file
func InjectProfile(path string, customProfile map[string]string) (*ProfileInjectionResult, error) {
	// Initialize result
	result := &ProfileInjectionResult{
		FieldsAdded:  []string{},
		FieldsFailed: []string{},
	}

	// load default profile if no custom provided
	var profile map[string]string
	var err error

	if customProfile != nil {
		profile = customProfile
	} else {
		// load profile from config
		profile, err = config.LoadProfile()
		if err != nil {
			// fall back to default
			profile = config.GetDefaultProfile()
		}
	}

	result.Profile = profile

	fileType, err := analyse.DetectFile(path)
	if err != nil {
		return result, fmt.Errorf("file type detection failed: %w", err)
	}

	handler, err := formats.GetHandler(fileType.Format)
	if err != nil {
		return result, fmt.Errorf("no handler for format %s: %w", fileType.Format, err)
	}

	profile = processDynamicFields(profile)

	err = handler.InjectMetadata(path, profile)
	if err != nil {
		return result, fmt.Errorf("metadata injection failed: %w", err)
	}

	// verify injection
	verifyResult, err := VerifyFile(path, profile)
	if err != nil {
		return result, fmt.Errorf("failed to verify injection: %w", err)
	}

	// populate result fields
	result.Success = verifyResult.ProfileInjected

	// determine which fields were added successfully
	for field := range profile {
		if slices.Contains(verifyResult.MissingFields, field) {
			result.FieldsFailed = append(result.FieldsFailed, field)
		} else {
			result.FieldsAdded = append(result.FieldsAdded, field)
		}
	}

	return result, nil
}

// dynamic values in the profile
func processDynamicFields(profile map[string]string) map[string]string {
	result := make(map[string]string, len(profile))

	for k, v := range profile {
		if v == "{{now}}" {
			// current date in ISO format
			result[k] = time.Now().Format("2006-01-02")
		} else if v == "{{random}}" {
			// random identifier
			result[k] = util.GenerateRandomID()
		} else {
			// use original value
			result[k] = v
		}
	}

	return result
}

// user-friendly report of the injection
func FormatInjectionResult(result *ProfileInjectionResult) string {
	var sb strings.Builder

	if result.Success {
		sb.WriteString(util.SEC.Render(fmt.Sprintf(
			"✓ Profile successfully injected (%d fields)", len(result.FieldsAdded))))
		sb.WriteString("\n")
		return sb.String()
	}

	if len(result.FieldsAdded) > 0 {
		message := fmt.Sprintf("✓ Successfully added %d profile fields:", len(result.FieldsAdded))
		sb.WriteString(util.LBL.Render(message))
		sb.WriteString("\n")

		for _, field := range result.FieldsAdded {
			value := result.Profile[field]
			sb.WriteString("  ")
			sb.WriteString(util.NSH.Render("• " + field + ": " + value))
			sb.WriteString("\n")
		}
	}

	if len(result.FieldsFailed) > 0 {
		message := fmt.Sprintf("! Failed to add %d profile fields:", len(result.FieldsFailed))
		sb.WriteString(util.LBL.Render(message))
		sb.WriteString("\n")

		for _, field := range result.FieldsFailed {
			value := result.Profile[field]
			sb.WriteString("  ")
			sb.WriteString(util.NSH.Render("• " + field + ": " + value))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
