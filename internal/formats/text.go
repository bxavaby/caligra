// BYZRA ⸻ internal/formats/text.go
// text format handlers

package formats

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// implements FormatHandler for text files
type TextHandler struct{}

// extracts metadata from text files
func (h *TextHandler) ExtractMetadata(path string) (map[string]any, error) {
	// for text files, search for patterns that might indicate metadata
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	metadata := make(map[string]any)

	// HTML metadata in meta tags
	if strings.HasSuffix(strings.ToLower(path), ".html") ||
		strings.HasSuffix(strings.ToLower(path), ".htm") {
		extractHTMLMetadata(string(content), metadata)
	}

	// Markdown front matter
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		extractMarkdownFrontMatter(string(content), metadata)
	}

	// common headers in all text files
	extractCommonTextMetadata(string(content), metadata)

	return metadata, nil
}

// removes metadata from text files
func (h *TextHandler) WipeMetadata(path string) error {
	// read content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read text file: %w", err)
	}

	var newContent string

	// process based on file type
	if strings.HasSuffix(strings.ToLower(path), ".html") ||
		strings.HasSuffix(strings.ToLower(path), ".htm") {
		newContent = removeHTMLMetadata(string(content))
	} else if strings.HasSuffix(strings.ToLower(path), ".md") {
		newContent = removeMarkdownFrontMatter(string(content))
	} else {
		// for general text, remove any lines that look like metadata
		newContent = removeCommonTextMetadata(string(content))
	}

	// write back to the file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write cleaned text file: %w", err)
	}

	return nil
}

// adds profile metadata to text files
func (h *TextHandler) InjectMetadata(path string, profile map[string]string) error {
	// read the content
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read text file: %w", err)
	}

	var newContent string

	// process based on file type
	if strings.HasSuffix(strings.ToLower(path), ".html") ||
		strings.HasSuffix(strings.ToLower(path), ".htm") {
		newContent = injectHTMLMetadata(string(content), profile)
	} else if strings.HasSuffix(strings.ToLower(path), ".md") {
		newContent = injectMarkdownFrontMatter(string(content), profile)
	} else {
		// for general text, add metadata as comments at the top
		newContent = injectTextFileComments(string(content), profile)
	}

	// write back to the file
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write text file with metadata: %w", err)
	}

	return nil
}

// for text files just checks if the file is readable
func (h *TextHandler) VerifyIntegrity(path string) bool {
	_, err := os.ReadFile(path)
	return err == nil
}

// helper functions for extracting metadata

func extractHTMLMetadata(content string, metadata map[string]any) {
	// extract meta tags
	metaRegex := regexp.MustCompile(`<meta\s+(?:name|property)=["']([^"']+)["']\s+content=["']([^"']+)["']`)
	matches := metaRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) == 3 {
			metadata[match[1]] = match[2]
		}
	}

	// extract title
	titleRegex := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	if match := titleRegex.FindStringSubmatch(content); len(match) == 2 {
		metadata["title"] = match[1]
	}
}

func extractMarkdownFrontMatter(content string, metadata map[string]any) {
	// look for YAML front matter between --- markers
	frontMatterRegex := regexp.MustCompile(`(?s)^---\s*(.*?)\s*---`)
	if match := frontMatterRegex.FindStringSubmatch(content); len(match) == 2 {
		frontMatter := match[1]

		// extract key-value pairs
		lineRegex := regexp.MustCompile(`(?m)^([^:]+):\s*(.*)$`)
		matches := lineRegex.FindAllStringSubmatch(frontMatter, -1)

		for _, kv := range matches {
			if len(kv) == 3 {
				metadata[strings.TrimSpace(kv[1])] = strings.TrimSpace(kv[2])
			}
		}
	}
}

func extractCommonTextMetadata(content string, metadata map[string]any) {
	// look for common patterns like "Author: Name" or "Date: 2023-01-01"
	patterns := []string{
		`Author:\s*([^\r\n]+)`,
		`Date:\s*([^\r\n]+)`,
		`Created:\s*([^\r\n]+)`,
		`Version:\s*([^\r\n]+)`,
		`Copyright:\s*([^\r\n]+)`,
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		if match := regex.FindStringSubmatch(content); len(match) == 2 {
			key := strings.ToLower(strings.Split(pattern, ":")[0])
			metadata[key] = strings.TrimSpace(match[1])
		}
	}
}

// helper functions for removing metadata

func removeHTMLMetadata(content string) string {
	// remove meta tags
	content = regexp.MustCompile(`<meta\s+(?:name|property)=["'][^"']+["']\s+content=["'][^"']+["'][^>]*>`).
		ReplaceAllString(content, "")

	// remove title content but keep tag structure
	content = regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).
		ReplaceAllString(content, "<title></title>")

	return content
}

func removeMarkdownFrontMatter(content string) string {
	// remove YAML front matter between --- markers
	return regexp.MustCompile(`(?s)^---\s*.*?\s*---`).
		ReplaceAllString(content, "")
}

func removeCommonTextMetadata(content string) string {
	// remove common metadata patterns
	patterns := []string{
		`(?m)^Author:\s*[^\r\n]+$`,
		`(?m)^Date:\s*[^\r\n]+$`,
		`(?m)^Created:\s*[^\r\n]+$`,
		`(?m)^Version:\s*[^\r\n]+$`,
		`(?m)^Copyright:\s*[^\r\n]+$`,
	}

	for _, pattern := range patterns {
		content = regexp.MustCompile(pattern).ReplaceAllString(content, "")
	}

	return content
}

// helper functions for injecting metadata

func injectHTMLMetadata(content string, profile map[string]string) string {
	// prepare meta tags
	metaTags := ""
	for key, value := range profile {
		metaTags += fmt.Sprintf(`<meta name="%s" content="%s">`, key, value)
	}

	// find head tag to insert meta tags
	headRegex := regexp.MustCompile(`<head[^>]*>`)
	if headRegex.MatchString(content) {
		return headRegex.ReplaceAllString(content, `$0`+metaTags)
	}

	// if no head tag, add one
	if strings.Contains(strings.ToLower(content), "<html") {
		htmlRegex := regexp.MustCompile(`<html[^>]*>`)
		return htmlRegex.ReplaceAllString(content, `$0<head>`+metaTags+`</head>`)
	}

	// last resort, add at the beginning
	return `<head>` + metaTags + `</head>` + content
}

func injectMarkdownFrontMatter(content string, profile map[string]string) string {
	// remove existing front matter if present
	content = removeMarkdownFrontMatter(content)

	// create new front matter
	frontMatter := "---\n"
	for key, value := range profile {
		frontMatter += fmt.Sprintf("%s: %s\n", key, value)
	}
	frontMatter += "---\n\n"

	return frontMatter + content
}

func injectTextFileComments(content string, profile map[string]string) string {
	// add metadata as comments at the top
	header := "# File Metadata\n"
	for key, value := range profile {
		header += fmt.Sprintf("# %s: %s\n", key, value)
	}
	header += "\n"

	return header + content
}
