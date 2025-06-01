package goripgrep

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// SimplifiedFind performs a simple, optimized search for comparison
func SimplifiedFind(pattern, filePath string, useRegex bool) ([]Match, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []Match
	scanner := bufio.NewScanner(file)

	// Use large buffer for better performance
	buf := make([]byte, 0, 128*1024) // 128KB buffer
	scanner.Buffer(buf, 1024*1024)   // 1MB max line

	var compiled *regexp.Regexp
	if useRegex {
		compiled, err = regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
	}

	lineNum := 1
	for scanner.Scan() {
		line := scanner.Text()

		var found bool
		var position int

		if useRegex && compiled != nil {
			if match := compiled.FindStringIndex(line); match != nil {
				found = true
				position = match[0]
			}
		} else {
			if pos := strings.Index(line, pattern); pos != -1 {
				found = true
				position = pos
			}
		}

		if found {
			matches = append(matches, Match{
				File:    filePath,
				Line:    lineNum,
				Column:  position + 1,
				Content: line,
			})
		}

		lineNum++
	}

	return matches, scanner.Err()
}
