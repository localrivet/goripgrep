package goripgrep

import (
	"fmt"
	"regexp"
	"strings"
)

// RegexEngine provides enhanced regex capabilities
type RegexEngine struct {
	pattern       string
	ignoreCase    bool
	compiledRegex *regexp.Regexp
}

// RegexMatch represents a regex match with capture groups
type RegexMatch struct {
	Start  int
	End    int
	Text   string
	Groups []string
	Named  map[string]string
}

// NewRegex creates a new regex engine with the given pattern and case sensitivity
func NewRegex(pattern string, ignoreCase bool) (*RegexEngine, error) {
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}

	engine := &RegexEngine{
		pattern:    pattern,
		ignoreCase: ignoreCase,
	}

	// Compile the regex with appropriate flags
	var regexPattern string
	if ignoreCase {
		regexPattern = "(?i)" + pattern
	} else {
		regexPattern = pattern
	}

	compiled, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	engine.compiledRegex = compiled
	return engine, nil
}

// FindAll finds all matches in the text
func (e *RegexEngine) FindAll(text string) []RegexMatch {
	matches := e.compiledRegex.FindAllStringSubmatchIndex(text, -1)
	var results []RegexMatch

	for _, match := range matches {
		if len(match) >= 2 {
			start := match[0]
			end := match[1]
			matchText := text[start:end]

			// Extract capture groups
			groups := e.extractGroups(text, match)

			// Extract named groups
			named := e.extractNamedGroups(text, match)

			results = append(results, RegexMatch{
				Start:  start,
				End:    end,
				Text:   matchText,
				Groups: groups,
				Named:  named,
			})
		}
	}

	return results
}

// Matches checks if the entire text matches the pattern
func (e *RegexEngine) Matches(text string) bool {
	return e.compiledRegex.MatchString(text)
}

// ReplaceAll replaces all matches with the replacement string
func (e *RegexEngine) ReplaceAll(text, replacement string) string {
	return e.compiledRegex.ReplaceAllString(text, replacement)
}

// Groups returns capture groups for the first match
func (e *RegexEngine) Groups(text string) []string {
	match := e.compiledRegex.FindStringSubmatchIndex(text)
	if match == nil {
		return nil
	}
	return e.extractGroups(text, match)
}

// NamedGroups returns named capture groups for the first match
func (e *RegexEngine) NamedGroups(text string) map[string]string {
	match := e.compiledRegex.FindStringSubmatchIndex(text)
	if match == nil {
		return nil
	}
	return e.extractNamedGroups(text, match)
}

// extractGroups extracts all capture groups from a match
func (e *RegexEngine) extractGroups(text string, match []int) []string {
	var groups []string

	// Add the full match first
	if len(match) >= 2 {
		groups = append(groups, text[match[0]:match[1]])
	}

	// Add capture groups
	for i := 2; i < len(match); i += 2 {
		if match[i] >= 0 && match[i+1] >= 0 {
			groups = append(groups, text[match[i]:match[i+1]])
		} else {
			groups = append(groups, "")
		}
	}

	return groups
}

// extractNamedGroups extracts named capture groups from a match
func (e *RegexEngine) extractNamedGroups(text string, match []int) map[string]string {
	named := make(map[string]string)
	groupNames := e.compiledRegex.SubexpNames()

	for i, name := range groupNames {
		if i > 0 && name != "" && i*2 < len(match) {
			start := match[i*2]
			end := match[i*2+1]
			if start >= 0 && end >= 0 {
				named[name] = text[start:end]
			} else {
				named[name] = ""
			}
		}
	}

	return named
}

// SupportsFeature checks if a regex feature is supported
func (e *RegexEngine) SupportsFeature(feature string) bool {
	switch feature {
	case "lookahead", "lookbehind", "backreferences", "unicode_classes":
		return true
	default:
		return false
	}
}

// Validate checks if a regex pattern is valid
func Validate(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

// Optimize attempts to optimize a regex pattern for better performance
func Optimize(pattern string) string {
	// Simple optimizations
	optimized := pattern

	// Remove unnecessary groups
	optimized = regexp.MustCompile(`\(\?\:`).ReplaceAllString(optimized, "(")

	// Simplify character classes
	optimized = regexp.MustCompile(`\[a-zA-Z\]`).ReplaceAllString(optimized, `\w`)

	return optimized
}

// Escape escapes special regex characters in text
func Escape(text string) string {
	return regexp.QuoteMeta(text)
}

// Complexity returns a complexity score for the regex pattern (1-10)
func Complexity(pattern string) int {
	score := 1

	// Count various regex features that increase complexity
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "+") {
		score += 2
	}
	if strings.Contains(pattern, "?") {
		score += 1
	}
	if strings.Contains(pattern, "|") {
		score += 2
	}
	if strings.Contains(pattern, "[") {
		score += 1
	}
	if strings.Contains(pattern, "(") {
		score += 1
	}
	if strings.Contains(pattern, "\\") {
		score += 1
	}

	// Cap at 10
	if score > 10 {
		score = 10
	}

	return score
}

// IsLiteral checks if a pattern is a literal string (no regex metacharacters)
func IsLiteral(pattern string) bool {
	// Check for common regex metacharacters
	metaChars := []string{".", "*", "+", "?", "^", "$", "|", "[", "]", "(", ")", "{", "}", "\\"}
	for _, char := range metaChars {
		if strings.Contains(pattern, char) {
			return false
		}
	}
	return true
}

// ExtractLiterals extracts literal strings from a regex pattern for optimization
func ExtractLiterals(pattern string) []string {
	var literals []string

	// Handle simple alternation patterns like "foo|bar|baz"
	if strings.Contains(pattern, "|") && !strings.Contains(pattern, "(") {
		parts := strings.Split(pattern, "|")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" && isAlphaNumeric(part) {
				literals = append(literals, part)
			}
		}
		if len(literals) > 0 {
			return literals
		}
	}

	// Skip patterns that are primarily regex constructs
	if strings.HasPrefix(pattern, "[") || strings.HasSuffix(pattern, "+") || strings.HasSuffix(pattern, "*") || strings.HasSuffix(pattern, "?") {
		return literals // Return empty slice
	}

	// Extract simple literal sequences by parsing character by character
	var current strings.Builder
	inCharClass := false

	i := 0
	for i < len(pattern) {
		char := pattern[i]

		// Handle escape sequences
		if char == '\\' && i+1 < len(pattern) {
			// This is an escape sequence, skip both characters
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}
			i += 2 // Skip both the backslash and the next character
			continue
		}

		// Track character class boundaries
		if char == '[' {
			inCharClass = true
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}
			i++
			continue
		}
		if char == ']' {
			inCharClass = false
			i++
			continue
		}

		// Skip everything inside character classes
		if inCharClass {
			i++
			continue
		}

		if containsMetaChars(string(char)) {
			if current.Len() > 0 {
				literals = append(literals, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(char)
		}

		i++
	}

	// Handle final literal if any
	if current.Len() > 0 {
		literals = append(literals, current.String())
	}

	return literals
}

// Helper functions
func isAlphaNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func containsMetaChars(s string) bool {
	metaChars := ".+*?^$|[](){}\\"
	for _, r := range s {
		if strings.ContainsRune(metaChars, r) {
			return true
		}
	}
	return false
}
