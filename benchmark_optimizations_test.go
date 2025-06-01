package goripgrep

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
)

// BenchmarkStringSearch compares different string search approaches
func BenchmarkStringSearch(b *testing.B) {
	content := strings.Repeat("BurntSushi is a great developer\n", 1000)
	pattern := "Sushi"

	b.Run("StandardStringsContains", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = strings.Contains(content, pattern)
		}
	})

	b.Run("BytesContains", func(b *testing.B) {
		contentBytes := []byte(content)
		patternBytes := []byte(pattern)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = bytes.Contains(contentBytes, patternBytes)
		}
	})

	b.Run("RegexpFindString", func(b *testing.B) {
		re := regexp.MustCompile(pattern)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = re.FindString(content)
		}
	})
}

// BenchmarkMemoryAllocation tests allocation reduction strategies
func BenchmarkMemoryAllocation(b *testing.B) {
	content := strings.Repeat("BurntSushi is a great developer\n", 1000)

	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results := make([]Match, 0, 10)
			lines := strings.Split(content, "\n")
			for lineNum, line := range lines {
				if strings.Contains(line, "Sushi") {
					results = append(results, Match{
						File:    "test.txt",
						Line:    lineNum + 1,
						Column:  strings.Index(line, "Sushi") + 1,
						Content: line,
					})
				}
			}
		}
	})

	b.Run("WithPool", func(b *testing.B) {
		pool := sync.Pool{
			New: func() interface{} {
				slice := make([]Match, 0, 10)
				return &slice
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			results := pool.Get().(*[]Match)
			*results = (*results)[:0] // Reset slice

			lines := strings.Split(content, "\n")
			for lineNum, line := range lines {
				if strings.Contains(line, "Sushi") {
					*results = append(*results, Match{
						File:    "test.txt",
						Line:    lineNum + 1,
						Column:  strings.Index(line, "Sushi") + 1,
						Content: line,
					})
				}
			}

			pool.Put(results)
		}
	})
}

// BenchmarkFileReading compares different file reading strategies
func BenchmarkFileReading(b *testing.B) {
	// Create a test file
	testFile := "benchmark_file_reading.txt"
	content := strings.Repeat("BurntSushi is a great developer\n", 10000)
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	b.Run("ScannerDefault", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file, err := os.Open(testFile)
			if err != nil {
				b.Fatalf("Failed to open file: %v", err)
			}

			scanner := bufio.NewScanner(file)
			count := 0
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "Sushi") {
					count++
				}
			}

			file.Close()
		}
	})

	b.Run("ScannerLargeBuffer", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			file, err := os.Open(testFile)
			if err != nil {
				b.Fatalf("Failed to open file: %v", err)
			}

			scanner := bufio.NewScanner(file)
			buf := make([]byte, 0, 128*1024) // 128KB buffer
			scanner.Buffer(buf, 1024*1024)   // 1MB max

			count := 0
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), "Sushi") {
					count++
				}
			}

			file.Close()
		}
	})

	b.Run("ReadAll", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			content, err := os.ReadFile(testFile)
			if err != nil {
				b.Fatalf("Failed to read file: %v", err)
			}

			lines := bytes.Split(content, []byte("\n"))
			count := 0
			for _, line := range lines {
				if bytes.Contains(line, []byte("Sushi")) {
					count++
				}
			}
		}
	})
}

// BenchmarkRegexCompilation tests regex caching benefits
func BenchmarkRegexCompilation(b *testing.B) {
	patterns := []string{
		`\w+Sushi`,
		`\b\w+[aeiou]{2,}\w+\b`,
		`(https?://)?([\da-z\.-]+)\.([a-z\.]{2,6})`,
		`\d{4}-\d{2}-\d{2}`,
	}

	content := "BurntSushi is a great developer working on https://github.com and was born on 1985-03-15"

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pattern := range patterns {
				re, err := regexp.Compile(pattern)
				if err != nil {
					b.Fatalf("Failed to compile pattern: %v", err)
				}
				_ = re.FindString(content)
			}
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		cache := make(map[string]*regexp.Regexp)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, pattern := range patterns {
				re, exists := cache[pattern]
				if !exists {
					var err error
					re, err = regexp.Compile(pattern)
					if err != nil {
						b.Fatalf("Failed to compile pattern: %v", err)
					}
					cache[pattern] = re
				}
				_ = re.FindString(content)
			}
		}
	})
}

// BenchmarkContextCancellation tests overhead of context checking
func BenchmarkContextCancellation(b *testing.B) {
	ctx := context.Background()
	content := strings.Repeat("BurntSushi is a great developer\n", 10000)
	lines := strings.Split(content, "\n")

	b.Run("WithoutContextCheck", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for _, line := range lines {
				if strings.Contains(line, "Sushi") {
					count++
				}
			}
		}
	})

	b.Run("WithContextCheckEvery1000", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for idx, line := range lines {
				if idx%1000 == 0 {
					select {
					case <-ctx.Done():
						return
					default:
					}
				}
				if strings.Contains(line, "Sushi") {
					count++
				}
			}
		}
	})

	b.Run("WithContextCheckEveryLine", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for _, line := range lines {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if strings.Contains(line, "Sushi") {
					count++
				}
			}
		}
	})
}

// BenchmarkCurrentOptimizations compares our current implementation with potential improvements
func BenchmarkCurrentOptimizations(b *testing.B) {
	pattern := `\w+Sushi`
	testFile := "benchmark_optimization_test.txt"
	content := strings.Repeat("BurntSushi is a great developer\n", 1000)
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	b.Run("CurrentImplementation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("DirectSimplifiedFind", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := SimplifiedFind(pattern, testFile, false)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})
}
