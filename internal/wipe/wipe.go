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

func DefaultWipeOptions() *WipeOptions {
	return &WipeOptions{
		InjectProfile: true,
		CustomProfile: nil,
		CreateCopy:    true,
		KeepBackup:    true,
		SecureDelete:  false,
	}
}

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

	// get metadata before wiping
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
		// output with .volena ext
		outputPath = util.GenerateOutputPath(path)
		result.OutputPath = outputPath

		// copy
		if err := util.SafeCopy(path, outputPath); err != nil {
			return result, fmt.Errorf("failed to create output file: %w", err)
		}
	} else {
		// backup original
		backupPath, err := util.CreateBackup(path)
		if err != nil {
			return result, fmt.Errorf("failed to create backup: %w", err)
		}
		result.BackupPath = backupPath
	}

	workingPath := outputPath

	// wipe metadata
	util.SpinWhile(fmt.Sprintf("[~] Wiping metadata from %s", filepath.Base(workingPath)), func() (string, error) {
		if err := handler.WipeMetadata(workingPath); err != nil {
			result.WipeErrors = append(result.WipeErrors, fmt.Sprintf("[X] Metadata wipe failed: %s", err))
			return "", err
		}
		return "Metadata removed", nil
	})

	// profile injection
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

	// option-based clean up
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

// report of the wipe operation
func FormatWipeResult(result *WipeResult) string {
	var sb strings.Builder

	if len(result.SensitiveData) > 0 {
		message := fmt.Sprintf("[!] Found %d sensitive metadata fields", len(result.SensitiveData))
		sb.WriteString(util.BRH.Render(message))
		sb.WriteString("\n")
	} else {
		message := "[i] No sensitive metadata detected"
		sb.WriteString(util.SEC.Render(message))
		sb.WriteString("\n")
	}

	if result.Success {
		sb.WriteString(util.SEC.Render("✓ File successfully processed"))
		sb.WriteString("\n")

		if result.OutputPath != "" && result.OutputPath != result.OriginalPath {
			message := fmt.Sprintf("[i] Output saved to: %s", result.OutputPath)
			sb.WriteString(util.NSH.Render(message))
			sb.WriteString("\n")
		}

		if result.BackupPath != "" {
			message := fmt.Sprintf("[i] Backup created at: %s", result.BackupPath)
			sb.WriteString(util.NSH.Render(message))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(util.BRH.Render("[!] Processing completed with issues..."))
		sb.WriteString("\n")

		for _, err := range result.WipeErrors {
			message := fmt.Sprintf("  • %s", err)
			sb.WriteString(util.NSH.Render(message))
			sb.WriteString("\n")
		}

		if result.BackupPath != "" {
			message := fmt.Sprintf("[i] Original preserved at: %s", result.BackupPath)
			sb.WriteString(util.SEC.Render(message))
			sb.WriteString("\n")
		}
	}

	if (result.Verification != nil && !result.Verification.Success) ||
		(result.Injection != nil && !result.Injection.Success) {
		sb.WriteString("\n")
	}

	// verification details
	if result.Verification != nil && !result.Verification.Success {
		sb.WriteString(FormatVerificationResult(result.Verification))
	}

	// injection details
	if result.Injection != nil && !result.Injection.Success {
		sb.WriteString(FormatInjectionResult(result.Injection))
	}

	return sb.String()
}
