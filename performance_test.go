package goripgrep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// BenchmarkGoRipGrepVsStandardRegex compares our optimized search against Go's standard regex
func BenchmarkGoRipGrepVsStandardRegex(b *testing.B) {
	// Create test data
	testDir, testFiles := createBenchmarkData(b)
	defer os.RemoveAll(testDir)

	patterns := []string{
		"test",                   // Simple literal
		"func.*main",             // Simple regex
		"\\b\\w+@\\w+\\.\\w+\\b", // Email regex
		"TODO|FIXME|HACK",        // Alternation
		"(?i)error",              // Case insensitive
	}

	for _, pattern := range patterns {
		b.Run("Pattern_"+strings.ReplaceAll(pattern, "\\", "_"), func(b *testing.B) {
			b.Run("GoRipGrep_Optimized", func(b *testing.B) {
				benchmarkGoRipGrepOptimized(b, testDir, pattern)
			})

			b.Run("GoRipGrep_Simple", func(b *testing.B) {
				benchmarkGoRipGrepSimple(b, testDir, pattern)
			})

			b.Run("StandardRegex", func(b *testing.B) {
				benchmarkStandardRegex(b, testFiles, pattern)
			})
		})
	}
}

func benchmarkGoRipGrepOptimized(b *testing.B, testDir, pattern string) {
	config := SearchConfig{
		SearchPath:      testDir,
		MaxWorkers:      4,
		UseOptimization: true,
		UseGitignore:    false,
		IgnoreCase:      false,
		IncludeHidden:   true,
	}

	engine := NewSearchEngine(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := engine.Search(ctx, pattern)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
		if len(results.Matches) == 0 {
			b.Logf("No matches found for pattern: %s", pattern)
		}
	}
}

func benchmarkGoRipGrepSimple(b *testing.B, testDir, pattern string) {
	config := SearchConfig{
		SearchPath:      testDir,
		MaxWorkers:      4,
		UseOptimization: false, // Disable optimization
		UseGitignore:    false,
		IgnoreCase:      false,
		IncludeHidden:   true,
	}

	engine := NewSearchEngine(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := engine.Search(ctx, pattern)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
		if len(results.Matches) == 0 {
			b.Logf("No matches found for pattern: %s", pattern)
		}
	}
}

func benchmarkStandardRegex(b *testing.B, testFiles []string, pattern string) {
	// Compile the regex
	regex, err := regexp.Compile(pattern)
	if err != nil {
		b.Fatalf("Failed to compile regex: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var totalMatches int
		for _, filePath := range testFiles {
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			matches := regex.FindAll(content, -1)
			totalMatches += len(matches)
		}
		if totalMatches == 0 {
			b.Logf("No matches found for pattern: %s", pattern)
		}
	}
}

func createBenchmarkData(b *testing.B) (string, []string) {
	testDir, err := os.MkdirTemp("", "goripgrep_bench_*")
	if err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}

	// Create various types of files with different content
	files := map[string]string{
		"main.go": `package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	// TODO: Add error handling
	if err := doSomething(); err != nil {
		log.Fatal(err)
	}
}

func doSomething() error {
	// FIXME: This is a hack
	return nil
}

func processEmails() {
	emails := []string{
		"user@example.com",
		"admin@test.org",
		"support@company.net",
	}
	
	for _, email := range emails {
		fmt.Println("Processing:", email)
	}
}
`,
		"utils.go": `package main

import (
	"regexp"
	"strings"
)

func validateEmail(email string) bool {
	// Simple email validation
	pattern := ` + "`" + `\b\w+@\w+\.\w+\b` + "`" + `
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

func processText(text string) string {
	// Convert to lowercase for case-insensitive search
	return strings.ToLower(text)
}

func findPatterns(content string) []string {
	var results []string
	
	// Look for TODO comments
	if strings.Contains(content, "TODO") {
		results = append(results, "Found TODO")
	}
	
	// Look for FIXME comments  
	if strings.Contains(content, "FIXME") {
		results = append(results, "Found FIXME")
	}
	
	// Look for HACK comments
	if strings.Contains(content, "HACK") {
		results = append(results, "Found HACK")
	}
	
	return results
}

func testFunction() {
	// This is a test function with various patterns
	data := "test data with test patterns"
	if strings.Contains(data, "test") {
		fmt.Println("Found test pattern")
	}
}
`,
		"config.json": `{
	"name": "test-project",
	"version": "1.0.0",
	"description": "A test project for benchmarking",
	"author": "test@example.com",
	"keywords": ["test", "benchmark", "performance"],
	"scripts": {
		"test": "go test ./...",
		"build": "go build -o main ."
	},
	"dependencies": {
		"some-package": "^1.0.0"
	}
}`,
		"README.md": `# Test Project

This is a test project for benchmarking GoRipGrep performance.

## Features

- Fast text search
- Regex support  
- Case-insensitive search
- Context lines
- Gitignore support

## TODO

- [ ] Add more test cases
- [ ] Improve performance
- [ ] Add documentation

## Contact

For questions, email us at support@example.com

## Known Issues

- FIXME: Memory usage could be optimized
- HACK: Temporary workaround in place
`,
		"large_file.txt": strings.Repeat(`This is a large test file with many lines.
It contains various patterns like test, TODO, FIXME, and email addresses.
Contact us at info@company.com for more information.
The test suite should find multiple matches in this file.
Some lines have ERROR messages that need attention.
Other lines contain func definitions and main functions.
`, 100), // Repeat 100 times for a larger file
	}

	var testFiles []string
	for filename, content := range files {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		testFiles = append(testFiles, filePath)
	}

	return testDir, testFiles
}

// BenchmarkLiteralVsRegex compares literal search vs regex search
func BenchmarkLiteralVsRegex(b *testing.B) {
	testDir, _ := createBenchmarkData(b)
	defer os.RemoveAll(testDir)

	b.Run("LiteralSearch", func(b *testing.B) {
		config := SearchConfig{
			SearchPath:      testDir,
			UseOptimization: true,
		}
		engine := NewSearchEngine(config)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Search(ctx, "test") // Literal pattern
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("RegexSearch", func(b *testing.B) {
		config := SearchConfig{
			SearchPath:      testDir,
			UseOptimization: true,
		}
		engine := NewSearchEngine(config)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := engine.Search(ctx, "test.*pattern") // Regex pattern
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrency tests performance with different worker counts
func BenchmarkConcurrency(b *testing.B) {
	testDir, _ := createBenchmarkData(b)
	defer os.RemoveAll(testDir)

	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			config := SearchConfig{
				SearchPath:      testDir,
				MaxWorkers:      workers,
				UseOptimization: true,
			}
			engine := NewSearchEngine(config)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := engine.Search(ctx, "test")
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage tests memory efficiency
func BenchmarkMemoryUsage(b *testing.B) {
	testDir, _ := createBenchmarkData(b)
	defer os.RemoveAll(testDir)

	config := SearchConfig{
		SearchPath:      testDir,
		UseOptimization: true,
		MaxResults:      1000,
	}
	engine := NewSearchEngine(config)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		results, err := engine.Search(ctx, "test")
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
		// Force garbage collection to measure actual memory usage
		_ = results
	}
}

func TestPerformanceComparison(t *testing.T) {
	pattern := `\w+Sushi`
	testFile := "large_test.csv"

	// Skip if test file doesn't exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test file not found, skipping performance test")
	}

	t.Run("FastEngine", func(t *testing.T) {
		start := time.Now()
		results, err := QuickFind(pattern, testFile, false)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("FastEngine error: %v", err)
		}

		t.Logf("FastEngine: %d matches in %v", len(results), duration)
	})

	t.Run("CurrentEngine", func(t *testing.T) {
		start := time.Now()
		results, err := Find(pattern, testFile,
			WithWorkers(1),
			WithContextLines(0),
			WithOptimization(false),
			WithGitignore(false),
			WithStreamingSearch(false),
		)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("CurrentEngine error: %v", err)
		}

		t.Logf("CurrentEngine: %d matches in %v", len(results.Matches), duration)
	})

	t.Run("PureRegex", func(t *testing.T) {
		start := time.Now()

		re, err := regexp.Compile(pattern)
		if err != nil {
			t.Fatalf("Regex compile error: %v", err)
		}

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("File read error: %v", err)
		}

		matches := re.FindAll(content, -1)
		duration := time.Since(start)

		t.Logf("PureRegex: %d matches in %v", len(matches), duration)
	})
}

// BenchmarkMemoryMappedFiles tests the performance of memory-mapped file search
func BenchmarkMemoryMappedFiles(b *testing.B) {
	// Create a large test file
	testFile := "large_test_file.txt"
	content := ""
	for i := 0; i < 10000; i++ {
		content += "This is a test line with some BurntSushi content\n"
		content += "Another line without the pattern\n"
		content += "Yet another line with different content\n"
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	pattern := `\\w+Sushi`

	b.Run("WithoutMemoryMapping", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile,
				WithRecursive(false),
				// No memory mapping option = disabled by default
			)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("WithMemoryMapping", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile,
				WithRecursive(false),
				WithMemoryMappedFiles(), // Enable memory mapping
			)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("WithPerformanceMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile,
				WithRecursive(false),
				WithPerformanceMode(),
			)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})
}

// BenchmarkRealWorldPerformance tests performance on actual project files
func BenchmarkRealWorldPerformance(b *testing.B) {
	pattern := `\\w+Sushi`
	searchPath := "."

	b.Run("CurrentImplementation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := Find(pattern, searchPath,
				WithContext(ctx),
				WithRecursive(true),
			)
			cancel()
			if err != nil && err != context.DeadlineExceeded {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("PerformanceMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err := Find(pattern, searchPath,
				WithContext(ctx),
				WithRecursive(true),
				WithPerformanceMode(),
			)
			cancel()
			if err != nil && err != context.DeadlineExceeded {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})
}
