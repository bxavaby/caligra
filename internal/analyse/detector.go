// BYZRA ⸻ internal/analyse/detector.go
// file type detection system

package analyse

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FileType struct {
	Format    string // "image", "audio", "video", "text"
	Extension string // "jpg", "mp3", etc
	MimeType  string // "image/jpeg", etc
}

func DetectFile(path string) (FileType, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" && ext[0] == '.' {
		ext = ext[1:]
	}

	// 1st magic numbers
	ft, err := detectByMagicNumbers(path)
	if err == nil && ft.Format != "" {
		return ft, nil
	}

	// fallback to extension
	ft = detectByExtension(ext)
	if ft.Format != "" {
		return ft, nil
	}

	return FileType{}, fmt.Errorf("unknown file type for %s", path)
}

// examines file headers to determine type
func detectByMagicNumbers(path string) (FileType, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileType{}, err
	}
	defer file.Close()

	// read first 12 bytes for signature detection
	// many formats need 8+ bytes for accurate detection
	buffer := make([]byte, 12)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return FileType{}, err
	}

	// JPEG: FF D8 FF
	if bytes.HasPrefix(buffer, []byte{0xFF, 0xD8, 0xFF}) {
		return FileType{Format: "image", Extension: "jpg", MimeType: "image/jpeg"}, nil
	}

	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if bytes.HasPrefix(buffer, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return FileType{Format: "image", Extension: "png", MimeType: "image/png"}, nil
	}

	// GIF: 47 49 46 38 (GIF8)
	if bytes.HasPrefix(buffer, []byte{0x47, 0x49, 0x46, 0x38}) {
		return FileType{Format: "image", Extension: "gif", MimeType: "image/gif"}, nil
	}

	// TIFF: 49 49 2A 00 or 4D 4D 00 2A (II* or MM*)
	if bytes.HasPrefix(buffer, []byte{0x49, 0x49, 0x2A, 0x00}) ||
		bytes.HasPrefix(buffer, []byte{0x4D, 0x4D, 0x00, 0x2A}) {
		return FileType{Format: "image", Extension: "tiff", MimeType: "image/tiff"}, nil
	}

	// SVG: Usually starts with XML declaration or <svg
	// for this, we need to check more bytes, reopen and check for text patterns
	if isSVG(path) {
		return FileType{Format: "image", Extension: "svg", MimeType: "image/svg+xml"}, nil
	}

	// MP3: ID3 or FFFB or FFF3 or FFF2
	if bytes.HasPrefix(buffer, []byte{0x49, 0x44, 0x33}) || // ID3
		bytes.HasPrefix(buffer, []byte{0xFF, 0xFB}) || // MPEG ADTS, layer III
		bytes.HasPrefix(buffer, []byte{0xFF, 0xF3}) || // MPEG ADTS, layer III
		bytes.HasPrefix(buffer, []byte{0xFF, 0xF2}) { // MPEG ADTS, layer III
		return FileType{Format: "audio", Extension: "mp3", MimeType: "audio/mpeg"}, nil
	}

	// FLAC: 66 4C 61 43 (fLaC)
	if bytes.HasPrefix(buffer, []byte{0x66, 0x4C, 0x61, 0x43}) {
		return FileType{Format: "audio", Extension: "flac", MimeType: "audio/flac"}, nil
	}

	// OGG (covers both Ogg and Opus): 4F 67 67 53 (OggS)
	if bytes.HasPrefix(buffer, []byte{0x4F, 0x67, 0x67, 0x53}) {
		// Further inspection could distinguish between Ogg and Opus
		return FileType{Format: "audio", Extension: "ogg", MimeType: "audio/ogg"}, nil
	}

	// MP4: varies but often starts with ftyp at position 4
	if bytes.Equal(buffer[4:8], []byte{0x66, 0x74, 0x79, 0x70}) {
		return FileType{Format: "video", Extension: "mp4", MimeType: "video/mp4"}, nil
	}

	// AVI: 52 49 46 46 ...  41 56 49 (RIFF...AVI)
	if bytes.HasPrefix(buffer, []byte{0x52, 0x49, 0x46, 0x46}) {
		// check for AVI marker
		file.Seek(8, 0)
		aviMarker := make([]byte, 4)
		file.Read(aviMarker)
		if bytes.Equal(aviMarker, []byte{0x41, 0x56, 0x49, 0x20}) {
			return FileType{Format: "video", Extension: "avi", MimeType: "video/x-msvideo"}, nil
		}
	}

	// Plaintext detection requires different approach
	if isTextFile(path) {
		// determine if it's HTML, Markdown, or plain text
		textType, err := determineTextType(path)
		if err == nil {
			return textType, nil
		}
		return FileType{Format: "text", Extension: "txt", MimeType: "text/plain"}, nil
	}

	return FileType{}, nil
}

// isSVG checks if file is likely an SVG
func isSVG(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// read first 1KB for SVG markers
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}
	buffer = buffer[:n]

	// SVG usually starts with XML declaration or directly with <svg
	content := string(buffer)
	return strings.Contains(strings.ToLower(content), "<svg") ||
		(strings.Contains(content, "<?xml") && strings.Contains(strings.ToLower(content), "<svg"))
}

// checks if a file is likely a text file
func isTextFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// read a sample to check for binary content
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}

	// check if there are any null bytes or too many non-printable characters
	nullCount := 0
	controlCount := 0
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			nullCount++
		} else if buffer[i] < 32 &&
			buffer[i] != '\n' &&
			buffer[i] != '\r' &&
			buffer[i] != '\t' {
			controlCount++
		}
	}

	// heuristic: if more than 5% are null or control chars, likely binary
	threshold := n / 20
	return nullCount < threshold && controlCount < threshold
}

// checks if text file is HTML, Markdown or plain
func determineTextType(path string) (FileType, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return FileType{}, err
	}

	// convert to string and lowercase for easier pattern matching
	text := strings.ToLower(string(content))

	// check for HTML
	if strings.Contains(text, "<!doctype html>") ||
		strings.Contains(text, "<html") ||
		(strings.Contains(text, "<head") && strings.Contains(text, "<body")) {
		return FileType{Format: "text", Extension: "html", MimeType: "text/html"}, nil
	}

	// check for Markdown (more challenging as it's less standardized)
	// look for common markdown patterns
	mdPatterns := []string{
		"# ", "## ", "### ", "```", "*****", "-----",
		"- [ ]", "- [x]", "[](", "![](", "|---|", "```code",
	}

	mdCount := 0
	for _, pattern := range mdPatterns {
		if strings.Contains(text, pattern) {
			mdCount++
		}
	}

	// if we found several markdown patterns, it's likely markdown
	if mdCount >= 3 {
		return FileType{Format: "text", Extension: "md", MimeType: "text/markdown"}, nil
	}

	// default to plain text
	return FileType{Format: "text", Extension: "txt", MimeType: "text/plain"}, nil
}

// maps file extensions to types (fallback method)
func detectByExtension(ext string) FileType {
	// image
	switch ext {
	case "jpg", "jpeg":
		return FileType{Format: "image", Extension: ext, MimeType: "image/jpeg"}
	case "png":
		return FileType{Format: "image", Extension: ext, MimeType: "image/png"}
	case "gif":
		return FileType{Format: "image", Extension: ext, MimeType: "image/gif"}
	case "tiff":
		return FileType{Format: "image", Extension: ext, MimeType: "image/tiff"}
	case "svg":
		return FileType{Format: "image", Extension: ext, MimeType: "image/svg+xml"}

	// audio
	case "mp3":
		return FileType{Format: "audio", Extension: ext, MimeType: "audio/mpeg"}
	case "flac":
		return FileType{Format: "audio", Extension: ext, MimeType: "audio/flac"}
	case "opus":
		return FileType{Format: "audio", Extension: ext, MimeType: "audio/opus"}
	case "ogg":
		return FileType{Format: "audio", Extension: ext, MimeType: "audio/ogg"}

	// video
	case "mp4":
		return FileType{Format: "video", Extension: ext, MimeType: "video/mp4"}
	case "avi":
		return FileType{Format: "video", Extension: ext, MimeType: "video/x-msvideo"}

	// text
	case "txt":
		return FileType{Format: "text", Extension: ext, MimeType: "text/plain"}
	case "md":
		return FileType{Format: "text", Extension: ext, MimeType: "text/markdown"}
	case "html", "htm":
		return FileType{Format: "text", Extension: ext, MimeType: "text/html"}
	}

	return FileType{} // unknown
}
