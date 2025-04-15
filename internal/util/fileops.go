// BYZRA ⸻ internal/util/fileops.go
// file operation utilities

package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copies a file with integrity verification
func SafeCopy(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// copy contents
	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// sync to ensure writes are flushed
	if err = dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// verify integrity
	if err = verifyFileIntegrity(src, dst); err != nil {
		return err
	}

	return nil
}

// create a backup
func CreateBackup(path string) (string, error) {
	backupPath := path + ".bak"

	if _, err := os.Stat(backupPath); err == nil {
		return backupPath, nil
	}

	err := SafeCopy(path, backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// restore from backup
func RestoreBackup(backupPath string) error {
	if !strings.HasSuffix(backupPath, ".bak") {
		return fmt.Errorf("invalid backup path: %s", backupPath)
	}

	originalPath := strings.TrimSuffix(backupPath, ".bak")

	err := SafeCopy(backupPath, originalPath)
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// creates the output path with volena suffix
func GenerateOutputPath(path string) string {
	ext := filepath.Ext(path)
	basePath := strings.TrimSuffix(path, ext)
	return basePath + ".volena" + ext
}

func ValidatePath(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path validation failed: %w", err)
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, expected a file: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("file is not readable: %w", err)
	}
	file.Close()

	file, err = os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("file is not writable: %w", err)
	}
	file.Close()

	return nil
}

// temporary file for processing
func CreateTempFile(prefix string) (*os.File, error) {
	tempDir := os.TempDir()
	return os.CreateTemp(tempDir, prefix)
}

// deletes a file safely
func RemoveFile(path string) error {
	return os.Remove(path)
}

// checks if two files have the same content using SHA-256
func verifyFileIntegrity(file1, file2 string) error {
	hash1, err := calculateSHA256(file1)
	if err != nil {
		return err
	}

	hash2, err := calculateSHA256(file2)
	if err != nil {
		return err
	}

	if hash1 != hash2 {
		return fmt.Errorf("integrity verification failed: file checksums don't match")
	}

	return nil
}

// computes the SHA-256 hash of a file
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate file hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// file info, following symlinks
func GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// directory entries
func ListDirectory(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

// metadata keys match, ignoring case and common variations
func KeysMatch(key1, key2 string) bool {
	// direct match
	if strings.EqualFold(key1, key2) {
		return true
	}

	// common variations (Author/Creator/Artist)
	authorVariations := map[string]bool{
		"author": true, "creator": true, "artist": true, "by": true,
	}

	key1Lower := strings.ToLower(key1)
	key2Lower := strings.ToLower(key2)

	if authorVariations[key1Lower] && authorVariations[key2Lower] {
		return true
	}

	// location variations
	locationVariations := map[string]bool{
		"location": true, "place": true, "where": true,
	}

	if locationVariations[key1Lower] && locationVariations[key2Lower] {
		return true
	}

	// date variations
	dateVariations := map[string]bool{
		"date": true, "created": true, "createdate": true, "when": true,
	}

	if dateVariations[key1Lower] && dateVariations[key2Lower] {
		return true
	}

	return false
}
