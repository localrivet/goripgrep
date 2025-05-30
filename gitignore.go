package goripgrep

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GitignoreEngine provides gitignore pattern matching functionality
type GitignoreEngine struct {
	patterns []GitignorePattern
	basePath string
}

// GitignorePattern represents a single gitignore rule
type GitignorePattern struct {
	Pattern     string
	Regex       *regexp.Regexp
	Negation    bool
	Directory   bool
	Absolute    bool
	MatchPrefix bool
}

// NewGitignoreEngine creates a new gitignore engine
func NewGitignoreEngine(basePath string) *GitignoreEngine {
	engine := &GitignoreEngine{
		basePath: basePath,
	}

	// Load .gitignore files
	engine.loadGitignoreFiles()

	return engine
}

// loadGitignoreFiles loads all .gitignore files in the directory tree
func (g *GitignoreEngine) loadGitignoreFiles() {
	err := filepath.Walk(g.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		if info.IsDir() {
			return nil
		}

		if info.Name() == ".gitignore" {
			g.loadGitignoreFile(path)
		}

		return nil
	})

	// Silently continue on errors - no action needed
	_ = err
}

// loadGitignoreFile loads patterns from a specific .gitignore file
func (g *GitignoreEngine) loadGitignoreFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		pattern := g.parseGitignorePattern(line, filePath)
		if pattern != nil {
			g.patterns = append(g.patterns, *pattern)
		}
	}
}

// parseGitignorePattern parses a single gitignore pattern line
func (g *GitignoreEngine) parseGitignorePattern(line, gitignoreFile string) *GitignorePattern {
	pattern := &GitignorePattern{
		Pattern: line,
	}

	// Handle negation (!)
	if strings.HasPrefix(line, "!") {
		pattern.Negation = true
		line = line[1:]
	}

	// Handle directory patterns (trailing /)
	if strings.HasSuffix(line, "/") {
		pattern.Directory = true
		line = line[:len(line)-1]
	}

	// Handle absolute patterns (leading /)
	if strings.HasPrefix(line, "/") {
		pattern.Absolute = true
		line = line[1:]
	}

	// Convert gitignore pattern to regex
	regexPattern := g.gitignoreToRegex(line)

	var err error
	pattern.Regex, err = regexp.Compile(regexPattern)
	if err != nil {
		return nil
	}

	return pattern
}

// gitignoreToRegex converts a gitignore pattern to a regular expression
func (g *GitignoreEngine) gitignoreToRegex(pattern string) string {
	// Escape regex special characters except * and ?
	escaped := regexp.QuoteMeta(pattern)

	// Replace escaped wildcards with regex equivalents
	escaped = strings.ReplaceAll(escaped, "\\*\\*", "__DOUBLESTAR__")
	escaped = strings.ReplaceAll(escaped, "\\*", "[^/]*")
	escaped = strings.ReplaceAll(escaped, "__DOUBLESTAR__", ".*")
	escaped = strings.ReplaceAll(escaped, "\\?", "[^/]")

	// Handle character classes [abc]
	escaped = strings.ReplaceAll(escaped, "\\[", "[")
	escaped = strings.ReplaceAll(escaped, "\\]", "]")

	// Add anchors
	if strings.Contains(pattern, "/") {
		// Pattern contains slash, match from beginning
		escaped = "^" + escaped + "$"
	} else {
		// Pattern doesn't contain slash, can match anywhere in path
		escaped = "(^|/)" + escaped + "($|/)"
	}

	return escaped
}

// ShouldIgnore checks if a file should be ignored based on gitignore patterns
func (g *GitignoreEngine) ShouldIgnore(filePath string) bool {
	// Convert to relative path from base
	relPath, err := filepath.Rel(g.basePath, filePath)
	if err != nil {
		relPath = filePath
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	ignored := false

	// Apply patterns in order
	for _, pattern := range g.patterns {
		if g.matchesPattern(relPath, pattern) {
			if pattern.Negation {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

// matchesPattern checks if a path matches a gitignore pattern
func (g *GitignoreEngine) matchesPattern(path string, pattern GitignorePattern) bool {
	// Handle directory-only patterns
	if pattern.Directory {
		// Check if path is a directory or if any parent directory matches
		if !strings.HasSuffix(path, "/") {
			// For files, check if any parent directory matches
			parts := strings.Split(path, "/")
			for i := 0; i < len(parts)-1; i++ {
				dirPath := strings.Join(parts[:i+1], "/") + "/"
				if pattern.Regex.MatchString(dirPath) {
					return true
				}
			}
			return false
		}
	}

	// Handle absolute patterns
	if pattern.Absolute {
		return pattern.Regex.MatchString(path)
	}

	// Check if pattern matches the full path or any suffix
	if pattern.Regex.MatchString(path) {
		return true
	}

	// For non-absolute patterns, check all path components
	parts := strings.Split(path, "/")
	for i := 0; i < len(parts); i++ {
		subPath := strings.Join(parts[i:], "/")
		if pattern.Regex.MatchString(subPath) {
			return true
		}
	}

	return false
}

// GetIgnoredFiles returns a list of files that would be ignored
func (g *GitignoreEngine) GetIgnoredFiles(rootPath string) ([]string, error) {
	var ignoredFiles []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if g.ShouldIgnore(path) {
			ignoredFiles = append(ignoredFiles, path)

			// If it's a directory, skip walking into it
			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	return ignoredFiles, err
}

// AddPattern adds a custom gitignore pattern
func (g *GitignoreEngine) AddPattern(patternStr string) error {
	pattern := g.parseGitignorePattern(patternStr, "custom")
	if pattern == nil {
		return fmt.Errorf("invalid pattern: %s", patternStr)
	}

	g.patterns = append(g.patterns, *pattern)
	return nil
}

// RemovePattern removes patterns matching the given string
func (g *GitignoreEngine) RemovePattern(patternStr string) {
	var newPatterns []GitignorePattern

	for _, pattern := range g.patterns {
		if pattern.Pattern != patternStr {
			newPatterns = append(newPatterns, pattern)
		}
	}

	g.patterns = newPatterns
}

// ListPatterns returns all loaded gitignore patterns
func (g *GitignoreEngine) ListPatterns() []string {
	var patterns []string

	for _, pattern := range g.patterns {
		patternStr := pattern.Pattern
		if pattern.Negation {
			patternStr = "!" + patternStr
		}
		patterns = append(patterns, patternStr)
	}

	return patterns
}

// IsGitRepository checks if the base path is a git repository
func (g *GitignoreEngine) IsGitRepository() bool {
	gitDir := filepath.Join(g.basePath, ".git")
	if info, err := os.Stat(gitDir); err == nil {
		return info.IsDir()
	}
	return false
}

// GetGitignoreFiles returns paths to all .gitignore files found
func (g *GitignoreEngine) GetGitignoreFiles() []string {
	var gitignoreFiles []string

	err := filepath.Walk(g.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.Name() == ".gitignore" {
			gitignoreFiles = append(gitignoreFiles, path)
		}

		return nil
	})

	// Silently continue on errors - no action needed
	_ = err

	return gitignoreFiles
}

// ValidatePattern checks if a gitignore pattern is valid
func (g *GitignoreEngine) ValidatePattern(pattern string) error {
	testPattern := g.parseGitignorePattern(pattern, "test")
	if testPattern == nil {
		return fmt.Errorf("invalid gitignore pattern: %s", pattern)
	}
	return nil
}

// MatchesAnyPattern checks if a path matches any of the loaded patterns
func (g *GitignoreEngine) MatchesAnyPattern(path string) (bool, string) {
	relPath, err := filepath.Rel(g.basePath, path)
	if err != nil {
		relPath = path
	}

	relPath = filepath.ToSlash(relPath)

	for _, pattern := range g.patterns {
		if g.matchesPattern(relPath, pattern) {
			return true, pattern.Pattern
		}
	}

	return false, ""
}
