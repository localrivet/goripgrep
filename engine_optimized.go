package goripgrep

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"strings"
)

// FastEngine provides a highly optimized search engine focused on speed
type FastEngine struct {
	pattern   string
	regex     *regexp.Regexp
	isLiteral bool
	literal   string
}

// NewFastEngine creates a new fast search engine
func NewFastEngine(pattern string, ignoreCase bool) (*FastEngine, error) {
	engine := &FastEngine{
		pattern: pattern,
	}

	// Simple literal pattern detection
	engine.isLiteral = isLiteralPattern(pattern)

	if engine.isLiteral {
		if ignoreCase {
			engine.literal = strings.ToLower(pattern)
		} else {
			engine.literal = pattern
		}
	} else {
		// Compile regex once
		regexPattern := pattern
		if ignoreCase {
			regexPattern = "(?i)" + pattern
		}

		var err error
		engine.regex, err = regexp.Compile(regexPattern)
		if err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// FastSearch performs optimized search on a file with minimal overhead
func (e *FastEngine) FastSearch(ctx context.Context, filePath string, ignoreCase bool) ([]Match, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []Match
	scanner := bufio.NewScanner(file)

	// Use a larger buffer for better I/O performance
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 1

	if e.isLiteral {
		// Optimized literal search
		searchTerm := e.literal

		for scanner.Scan() {
			// Check context cancellation occasionally
			if lineNum%1000 == 0 {
				select {
				case <-ctx.Done():
					return results, ctx.Err()
				default:
				}
			}

			line := scanner.Text()

			// Fast literal search
			var searchLine string
			if ignoreCase {
				searchLine = strings.ToLower(line)
			} else {
				searchLine = line
			}

			if idx := strings.Index(searchLine, searchTerm); idx != -1 {
				results = append(results, Match{
					File:    filePath,
					Line:    lineNum,
					Column:  idx + 1,
					Content: line,
				})
			}

			lineNum++
		}
	} else {
		// Optimized regex search
		for scanner.Scan() {
			// Check context cancellation occasionally
			if lineNum%1000 == 0 {
				select {
				case <-ctx.Done():
					return results, ctx.Err()
				default:
				}
			}

			line := scanner.Text()

			if matches := e.regex.FindAllStringIndex(line, -1); matches != nil {
				for _, match := range matches {
					results = append(results, Match{
						File:    filePath,
						Line:    lineNum,
						Column:  match[0] + 1,
						Content: line,
					})
				}
			}

			lineNum++
		}
	}

	return results, scanner.Err()
}

// QuickFind provides a minimal, fast search function for testing performance
func QuickFind(pattern, path string, ignoreCase bool) ([]Match, error) {
	engine, err := NewFastEngine(pattern, ignoreCase)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	return engine.FastSearch(ctx, path, ignoreCase)
}
