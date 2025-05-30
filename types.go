package goripgrep

import (
	"path/filepath"
	"strings"
)

// Match represents a single search result
type Match struct {
	File    string   // Path to the file containing the match
	Line    int      // Line number (1-indexed)
	Column  int      // Column number (1-indexed)
	Content string   // Content of the matching line
	Context []string // Context lines (if requested)
}

// SearchArgs represents arguments for search operations
type SearchArgs struct {
	Path          string
	Pattern       string
	FilePattern   *string
	IgnoreCase    *bool
	MaxResults    *int
	IncludeHidden *bool
	ContextLines  *int
	TimeoutMs     *int
}

// isLiteralPattern determines if a pattern is a literal string (no regex metacharacters)
func isLiteralPattern(pattern string) bool {
	// Check for common regex metacharacters
	metaChars := []string{".", "*", "+", "?", "^", "$", "|", "(", ")", "[", "]", "{", "}", "\\"}

	for _, meta := range metaChars {
		if strings.Contains(pattern, meta) {
			return false
		}
	}

	return true
}

// isBinaryFile checks if a file is likely binary based on its extension
func isBinaryFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))

	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".o": true, ".a": true, ".lib": true,
	}

	return binaryExts[ext]
}
