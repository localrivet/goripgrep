package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/localrivet/goripgrep"
)

func main() {
	pattern := `\w+Sushi`
	testFile := "src/cmd/go/internal/modfetch/zip_sum_test/testdata/zip_sums.csv"

	fmt.Println("=== GoRipGrep Performance Debug ===")

	// Test 1: Pure Go regex performance
	testPureRegex(pattern, testFile)

	// Test 2: Pattern classification
	testPatternClassification(pattern)

	// Test 3: Your search engine with minimal config
	testMinimalSearch(pattern, testFile)

	// Test 4: Your search engine with default config
	testDefaultSearch(pattern, testFile)

	// Test 5: Test just the file walking
	testFileWalking(".")
}

func testPureRegex(pattern, testFile string) {
	fmt.Println("\n--- Test 1: Pure Go Regex ---")
	start := time.Now()

	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("Regex compile error: %v\n", err)
		return
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		fmt.Printf("File read error: %v\n", err)
		return
	}

	matches := re.FindAll(content, -1)
	duration := time.Since(start)

	fmt.Printf("Pure regex: %d matches in %v\n", len(matches), duration)
	fmt.Printf("File size: %d bytes\n", len(content))
}

func testPatternClassification(pattern string) {
	fmt.Println("\n--- Test 2: Pattern Classification ---")
	start := time.Now()

	// Note: These functions need to be exported from goripgrep package
	// For now, we'll test what we can
	fmt.Printf("Pattern: %s\n", pattern)

	// Check if pattern contains regex metacharacters manually
	hasRegexChars := false
	metaChars := []string{".", "*", "+", "?", "^", "$", "|", "(", ")", "[", "]", "{", "}", "\\"}
	for _, meta := range metaChars {
		if contains(pattern, meta) {
			hasRegexChars = true
			break
		}
	}

	fmt.Printf("Contains regex metacharacters: %t\n", hasRegexChars)
	fmt.Printf("Should be treated as regex pattern: %t\n", hasRegexChars)

	fmt.Printf("Pattern classification took: %v\n", time.Since(start))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func testMinimalSearch(pattern, testFile string) {
	fmt.Println("\n--- Test 3: Minimal Search ---")
	start := time.Now()

	// Use minimal options
	results, err := goripgrep.Find(pattern, testFile,
		goripgrep.WithWorkers(1),             // Single worker
		goripgrep.WithContextLines(0),        // No context
		goripgrep.WithOptimization(false),    // No optimization
		goripgrep.WithGitignore(false),       // No gitignore
		goripgrep.WithStreamingSearch(false), // No streaming
	)

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		return
	}

	fmt.Printf("Minimal search: %d matches in %v\n", len(results.Matches), duration)
	fmt.Printf("Files scanned: %d\n", results.Stats.FilesScanned)
	fmt.Printf("Bytes scanned: %d\n", results.Stats.BytesScanned)
}

func testDefaultSearch(pattern, testFile string) {
	fmt.Println("\n--- Test 4: Default Search ---")
	start := time.Now()

	// Use default options (as your CLI would)
	results, err := goripgrep.Find(pattern, testFile)

	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		return
	}

	fmt.Printf("Default search: %d matches in %v\n", len(results.Matches), duration)
	fmt.Printf("Files scanned: %d\n", results.Stats.FilesScanned)
	fmt.Printf("Bytes scanned: %d\n", results.Stats.BytesScanned)
}

func testFileWalking(searchPath string) {
	fmt.Println("\n--- Test 5: File Walking Performance ---")
	start := time.Now()

	// Test just the directory walking part
	config := goripgrep.SearchConfig{
		SearchPath:   searchPath,
		MaxWorkers:   1,
		MaxResults:   1000,
		UseGitignore: true,
		Recursive:    true,
	}

	_ = goripgrep.NewSearchEngine(config)

	// This would normally search, but we're just testing the setup
	fmt.Printf("Engine setup took: %v\n", time.Since(start))

	// Count files that would be processed
	fileCount := 0
	// Note: This is a simplified test - in a real implementation you'd need
	// to expose the file walking logic separately to test it in isolation

	fmt.Printf("Would process approximately %d files\n", fileCount)
}
