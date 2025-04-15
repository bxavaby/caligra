// BYZRA ⸻ internal/util/security.go
// security operations for file handling

package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// overwrites a file multiple times before deletion
// helps prevent data recovery
func SecureOverwriteFile(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file for secure overwrite: %w", err)
	}

	size := fileInfo.Size()

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open file for secure overwrite: %w", err)
	}
	defer file.Close()

	// multiple pass overwrite
	// pass 1: all zeros
	if err := overwriteWithPattern(file, size, 0x00); err != nil {
		return err
	}

	// pass 2: all ones
	if err := overwriteWithPattern(file, size, 0xFF); err != nil {
		return err
	}

	// pass 3: random data
	if err := overwriteWithRandom(file, size); err != nil {
		return err
	}

	// sync to ensure all writes are flushed
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync during secure overwrite: %w", err)
	}

	// close before deletion
	file.Close()

	// delete file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file after secure overwrite: %w", err)
	}

	return nil
}

// removes unsafe characters from a path
func SanitizePath(path string) string {
	// replace potentially dangerous sequences
	sanitized := filepath.Clean(path)

	// remove path traversal sequences that might remain
	sanitized = strings.ReplaceAll(sanitized, "..", "")

	return sanitized
}

// sets secure permissions on a file
// 0600 = owner can read and write, no one else has access
func EnsureSafePermissions(path string) error {
	return os.Chmod(path, 0600)
}

// verifies the current user owns the file
func CheckFileOwnership(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file for ownership check: %w", err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to get file stats")
	}

	// get current user ID
	currentUID := os.Getuid()

	// current user file owner check
	if int(stat.Uid) != currentUID {
		return fmt.Errorf("file is not owned by current user")
	}

	return nil
}

// removes potentially unsafe characters from a filename
func SanitizeFilename(filename string) string {
	// remove path elements
	filename = filepath.Base(filename)

	// replace unsafe characters
	unsafe := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|", ";", "&"}
	for _, char := range unsafe {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// special cases like hidden files
	if strings.HasPrefix(filename, ".") {
		filename = "_" + filename[1:]
	}

	return filename
}

// overwrites a file with a specific byte pattern
func overwriteWithPattern(file *os.File, size int64, pattern byte) error {
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %w", err)
	}

	// create a buffer of the pattern (1MB chunks for efficiency)
	const maxBufSize int64 = 1024 * 1024 // 1MB
	bufSize := min(size, maxBufSize)
	buf := make([]byte, bufSize)
	for i := range buf {
		buf[i] = pattern
	}

	// write the pattern repeatedly until the file is covered
	remaining := size
	for remaining > 0 {
		writeSize := min(remaining, bufSize)

		if _, err := file.Write(buf[:writeSize]); err != nil {
			return fmt.Errorf("failed to write pattern: %w", err)
		}

		remaining -= writeSize
	}

	return nil
}

// overwrites a file with random data
func overwriteWithRandom(file *os.File, size int64) error {
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning: %w", err)
	}

	// create a buffer for random data (1MB chunks for efficiency)
	const maxBufSize int64 = 1024 * 1024 // 1MB
	bufSize := min(size, maxBufSize)
	buf := make([]byte, bufSize)

	// write random data repeatedly until the file is covered
	remaining := size
	for remaining > 0 {
		writeSize := min(remaining, bufSize)

		// fill buffer with random data
		if _, err := io.ReadFull(rand.Reader, buf[:writeSize]); err != nil {
			return fmt.Errorf("failed to generate random data: %w", err)
		}

		if _, err := file.Write(buf[:writeSize]); err != nil {
			return fmt.Errorf("failed to write random data: %w", err)
		}

		remaining -= writeSize
	}

	return nil
}

// random identifier for metadata
func GenerateRandomID() string {
	// 8-byte random value
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		// fallback if random fails
		return fmt.Sprintf("caligra-%d", time.Now().Unix())
	}

	// convert 2 hex string
	return fmt.Sprintf("caligra-%x", b)
}
