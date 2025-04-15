// BYZRA ⸻ internal/wipe/wipe.go
// main wipe orchestration

package wipe

import (
	"fmt"
	"path/filepath"
	"strings"

	"caligra/internal/analyse"
	"caligra/internal/formats"
	"caligra/internal/util"
)

// behavior of the wipe operation
type WipeOptions struct {
	// inject profile metadata after wiping?
	InjectProfile bool

	// custom profile to inject (nil for default)
	CustomProfile map[string]string

	// create a clean copy instead of modifying the original?
	CreateCopy bool

	// keep backup files?
	KeepBackup bool

	// securely overwrite original before deletion?
	SecureDelete bool
}

// default wiping behavior
func DefaultWipeOptions() *WipeOptions {
	return &WipeOptions{
		InjectProfile: true,
		CustomProfile: nil,
		CreateCopy:    true,
		KeepBackup:    true,
		SecureDelete:  false,
	}
}

// results of the wipe operation
type WipeResult struct {
	Success       bool
	OriginalPath  string
	OutputPath    string
	BackupPath    string
	SensitiveData []string
	WipeErrors    []string
	Verification  *VerificationResult
	Injection     *ProfileInjectionResult
}

// removes metadata from a file and optionally injects a profile
func WipeFile(path string, options *WipeOptions) (*WipeResult, error) {
	if options == nil {
		options = DefaultWipeOptions()
	}

	result := &WipeResult{
		OriginalPath: path,
		WipeErrors:   []string{},
	}

	if err := util.ValidatePath(path); err != nil {
		return result, fmt.Errorf("invalid input file: %w", err)
	}

	// analyze to get metadata before wiping
	report, err := analyse.Analyze(path)
	if err != nil {
		return result, fmt.Errorf("failed to analyze file: %w", err)
	}

	result.SensitiveData = report.SensitiveFields

	handler, err := formats.GetHandler(report.FileType.Format)
	if err != nil {
		return result, fmt.Errorf("no handler for format %s: %w", report.FileType.Format, err)
	}

	outputPath := path
	if options.CreateCopy {
		// output path with .volena extension
		outputPath = util.GenerateOutputPath(path)
		result.OutputPath = outputPath

		// copy file
		if err := util.SafeCopy(path, outputPath); err != nil {
			return result, fmt.Errorf("failed to create output file: %w", err)
		}
	} else {
		// backup of original first
		backupPath, err := util.CreateBackup(path)
		if err != nil {
			return result, fmt.Errorf("failed to create backup: %w", err)
		}
		result.BackupPath = backupPath
	}

	// Use the output path for all operations from here
	workingPath := outputPath

	// wipe metadata
	util.SpinWhile(fmt.Sprintf("[~] Wiping metadata from %s", filepath.Base(workingPath)), func() (string, error) {
		if err := handler.WipeMetadata(workingPath); err != nil {
			result.WipeErrors = append(result.WipeErrors, fmt.Sprintf("[X] Metadata wipe failed: %s", err))
			return "", err
		}
		return "Metadata removed", nil
	})

	// profile injection if requested
	if options.InjectProfile && len(result.WipeErrors) == 0 {
		injResult, err := InjectProfile(workingPath, options.CustomProfile)
		if err != nil {
			result.WipeErrors = append(result.WipeErrors, fmt.Sprintf("[X] Profile injection failed: %s", err))
		}
		result.Injection = injResult
	}

	verifyResult, err := VerifyFile(workingPath, options.CustomProfile)
	if err != nil {
		result.WipeErrors = append(result.WipeErrors, fmt.Sprintf("[X] Verification failed: %s", err))
	}
	result.Verification = verifyResult

	// clean up based on options
	if !options.CreateCopy && !options.KeepBackup && result.BackupPath != "" && len(result.WipeErrors) == 0 {
		if options.SecureDelete {
			_ = util.SecureOverwriteFile(result.BackupPath)
		} else {
			_ = util.RemoveFile(result.BackupPath)
		}
		result.BackupPath = ""
	}

	// set success based on errors and verification
	result.Success = len(result.WipeErrors) == 0 &&
		(result.Verification == nil || result.Verification.Success)

	return result, nil
}

// user-friendly report of the wipe operation
func FormatWipeResult(result *WipeResult) string {
	var sb strings.Builder

	if len(result.SensitiveData) > 0 {
		fmt.Fprintf(&sb, "%s", util.SUB.Render(fmt.Sprintf(
			"Found %d sensitive metadata fields\n", len(result.SensitiveData))))
	} else {
		fmt.Fprintf(&sb, "%s", util.SUB.Render("✓ No sensitive metadata detected\n"))
	}

	if result.Success {
		fmt.Fprintf(&sb, "%s", util.NSH.Render("[✓] File successfully processed\n"))

		if result.OutputPath != "" && result.OutputPath != result.OriginalPath {
			fmt.Fprintf(&sb, "%s", util.SUB.Render(fmt.Sprintf(
				"Output saved to: %s\n", result.OutputPath)))
		}

		if result.BackupPath != "" {
			fmt.Fprintf(&sb, "%s", util.SUB.Render(fmt.Sprintf(
				"Backup created at: %s\n", result.BackupPath)))
		}
	} else {
		fmt.Fprintf(&sb, "%s", util.LBL.Render("[!] Processing completed with issues:\n"))

		for _, err := range result.WipeErrors {
			fmt.Fprintf(&sb, "%s", util.SUB.Render(fmt.Sprintf("  • %s\n", err)))
		}

		if result.BackupPath != "" {
			fmt.Fprintf(&sb, "%s", util.NSH.Render(fmt.Sprintf(
				"Original preserved at: %s\n", result.BackupPath)))
		}
	}

	// verification details
	if result.Verification != nil && !result.Verification.Success {
		fmt.Fprintf(&sb, "%s", FormatVerificationResult(result.Verification))
	}

	// injection details
	if result.Injection != nil && !result.Injection.Success {
		fmt.Fprintf(&sb, "%s", FormatInjectionResult(result.Injection))
	}

	return sb.String()
}
