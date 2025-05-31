package goripgrep

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMmapSearcher(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mmap_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("SmallFile", func(t *testing.T) {
		// Test with a small file that shouldn't be memory-mapped
		content := "Hello world\nThis is a test\nAnother line"
		testFile := filepath.Join(tempDir, "small.txt")
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		options := DefaultMmapOptions()
		searcher, err := NewMmapSearcher(testFile, options)
		if err != nil {
			t.Fatalf("Failed to create searcher: %v", err)
		}
		defer searcher.Close()

		// Small file should not be mapped
		if searcher.IsMapped() {
			t.Error("Small file should not be memory-mapped")
		}

		// Test search functionality
		matcher := &LiteralMatcher{}
		matches, err := searcher.Search(context.Background(), "test", matcher)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(matches) != 1 {
			t.Errorf("Expected 1 match, got %d", len(matches))
		}

		if len(matches) > 0 && matches[0].Line != 2 {
			t.Errorf("Expected match on line 2, got line %d", matches[0].Line)
		}
	})

	t.Run("LargeFile", func(t *testing.T) {
		// Skip on platforms that don't support memory mapping
		if runtime.GOOS == "windows" {
			t.Skip("Memory mapping not implemented for Windows in this version")
		}

		// Create a large file that should be memory-mapped
		content := strings.Repeat("This is line number X with some test content\n", 2000000) // ~88MB
		testFile := filepath.Join(tempDir, "large.txt")
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create large test file: %v", err)
		}

		options := DefaultMmapOptions()
		searcher, err := NewMmapSearcher(testFile, options)
		if err != nil {
			t.Fatalf("Failed to create searcher: %v", err)
		}
		defer searcher.Close()

		// Large file should be mapped
		if !searcher.IsMapped() {
			t.Error("Large file should be memory-mapped")
		}

		// Test search functionality
		matcher := &LiteralMatcher{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		matches, err := searcher.Search(ctx, "test", matcher)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Should find many matches
		if len(matches) == 0 {
			t.Error("Expected to find matches in large file")
		}

		// Verify memory usage tracking
		usage := searcher.GetMemoryUsage()
		if usage.MappedSize == 0 {
			t.Error("Expected non-zero mapped size")
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		// Test with an empty file
		testFile := filepath.Join(tempDir, "empty.txt")
		if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create empty test file: %v", err)
		}

		options := DefaultMmapOptions()
		searcher, err := NewMmapSearcher(testFile, options)
		if err != nil {
			t.Fatalf("Failed to create searcher: %v", err)
		}
		defer searcher.Close()

		// Empty file should not be mapped
		if searcher.IsMapped() {
			t.Error("Empty file should not be memory-mapped")
		}

		// Test search functionality
		matcher := &LiteralMatcher{}
		matches, err := searcher.Search(context.Background(), "test", matcher)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(matches) != 0 {
			t.Errorf("Expected 0 matches in empty file, got %d", len(matches))
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		// Test with a non-existent file
		testFile := filepath.Join(tempDir, "nonexistent.txt")

		options := DefaultMmapOptions()
		_, err := NewMmapSearcher(testFile, options)
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		// Skip on platforms that don't support memory mapping
		if runtime.GOOS == "windows" {
			t.Skip("Memory mapping not implemented for Windows in this version")
		}

		// Create a large file for testing cancellation
		content := strings.Repeat("This is a very long line with test content that should be found\n", 1000000)
		testFile := filepath.Join(tempDir, "cancel_test.txt")
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		options := DefaultMmapOptions()
		searcher, err := NewMmapSearcher(testFile, options)
		if err != nil {
			t.Fatalf("Failed to create searcher: %v", err)
		}
		defer searcher.Close()

		// Create a context that will be cancelled quickly
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		matcher := &LiteralMatcher{}
		_, err = searcher.Search(ctx, "test", matcher)

		// Should get a context error (either Canceled or DeadlineExceeded)
		if err == nil {
			t.Skip("Search completed too quickly to test cancellation")
		}

		if err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Expected context error, got: %v", err)
		}
	})

	t.Run("FallbackDisabled", func(t *testing.T) {
		// Test with fallback disabled on a platform that doesn't support mapping
		content := strings.Repeat("test content\n", 10000000) // Large enough to trigger mapping
		testFile := filepath.Join(tempDir, "fallback_test.txt")
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		options := MmapSearchOptions{
			MinFileSize:     1024, // Very small threshold
			MaxMappingSize:  1024 * 1024 * 1024,
			FallbackEnabled: false, // Disable fallback
		}

		if runtime.GOOS == "windows" {
			// On Windows, this should fail since mapping isn't implemented
			_, err := NewMmapSearcher(testFile, options)
			if err == nil {
				t.Error("Expected error when fallback is disabled on Windows")
			}
		} else {
			// On Unix systems, this should work
			searcher, err := NewMmapSearcher(testFile, options)
			if err != nil {
				t.Fatalf("Failed to create searcher: %v", err)
			}
			defer searcher.Close()

			if !searcher.IsMapped() {
				t.Error("File should be memory-mapped")
			}
		}
	})
}

func TestLiteralMatcher(t *testing.T) {
	matcher := &LiteralMatcher{}

	testCases := []struct {
		name     string
		data     string
		pattern  string
		expected bool
		column   int
	}{
		{
			name:     "SimpleMatch",
			data:     "Hello world",
			pattern:  "world",
			expected: true,
			column:   7,
		},
		{
			name:     "NoMatch",
			data:     "Hello world",
			pattern:  "test",
			expected: false,
		},
		{
			name:     "EmptyPattern",
			data:     "Hello world",
			pattern:  "",
			expected: true,
			column:   1,
		},
		{
			name:     "EmptyData",
			data:     "",
			pattern:  "test",
			expected: false,
		},
		{
			name:     "ExactMatch",
			data:     "test",
			pattern:  "test",
			expected: true,
			column:   1,
		},
		{
			name:     "PatternLongerThanData",
			data:     "hi",
			pattern:  "hello",
			expected: false,
		},
		{
			name:     "MultipleOccurrences",
			data:     "test test test",
			pattern:  "test",
			expected: true,
			column:   1, // Should find first occurrence
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := matcher.Match([]byte(tc.data), tc.pattern)

			if tc.expected {
				if result == nil {
					t.Error("Expected match but got nil")
				} else if result.Column != tc.column {
					t.Errorf("Expected column %d, got %d", tc.column, result.Column)
				}
			} else {
				if result != nil {
					t.Error("Expected no match but got result")
				}
			}
		})
	}
}

func TestMmapSearcherMemoryUsage(t *testing.T) {
	// Skip on platforms that don't support memory mapping
	if runtime.GOOS == "windows" {
		t.Skip("Memory mapping not implemented for Windows in this version")
	}

	tempDir, err := os.MkdirTemp("", "mmap_memory_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a large file
	content := strings.Repeat("Memory usage test content\n", 3000000) // ~75MB
	testFile := filepath.Join(tempDir, "memory_test.txt")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	options := DefaultMmapOptions()
	searcher, err := NewMmapSearcher(testFile, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	if !searcher.IsMapped() {
		t.Skip("File was not memory-mapped, skipping memory usage test")
	}

	// Get memory usage
	usage := searcher.GetMemoryUsage()

	// Verify that mapped size is reported
	if usage.MappedSize == 0 {
		t.Error("Expected non-zero mapped size")
	}

	// Verify that heap stats are populated
	if usage.HeapAlloc == 0 {
		t.Error("Expected non-zero heap allocation")
	}

	// The mapped size should be approximately the file size
	expectedSize := uint64(len(content))
	if usage.MappedSize < expectedSize/2 || usage.MappedSize > expectedSize*2 {
		t.Errorf("Mapped size %d seems incorrect for file size %d", usage.MappedSize, expectedSize)
	}
}

func TestMmapSearcherAdviseSequential(t *testing.T) {
	// Skip on platforms that don't support memory mapping or madvise
	if runtime.GOOS == "windows" {
		t.Skip("Memory mapping not implemented for Windows in this version")
	}

	tempDir, err := os.MkdirTemp("", "mmap_advise_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a large file
	content := strings.Repeat("Sequential access test\n", 3000000)
	testFile := filepath.Join(tempDir, "advise_test.txt")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	options := DefaultMmapOptions()
	searcher, err := NewMmapSearcher(testFile, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	if !searcher.IsMapped() {
		t.Skip("File was not memory-mapped, skipping advise test")
	}

	// Test advising sequential access
	err = searcher.AdviseSequential()
	if err != nil {
		t.Errorf("AdviseSequential failed: %v", err)
	}
}

func BenchmarkMmapSearch(b *testing.B) {
	// Skip on platforms that don't support memory mapping
	if runtime.GOOS == "windows" {
		b.Skip("Memory mapping not implemented for Windows in this version")
	}

	tempDir, err := os.MkdirTemp("", "mmap_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a large test file
	content := strings.Repeat("This is a benchmark test line with some content to search\n", 1000000)
	testFile := filepath.Join(tempDir, "bench.txt")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	options := DefaultMmapOptions()
	searcher, err := NewMmapSearcher(testFile, options)
	if err != nil {
		b.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	if !searcher.IsMapped() {
		b.Skip("File was not memory-mapped, skipping benchmark")
	}

	// Advise sequential access for better performance
	if err := searcher.AdviseSequential(); err != nil {
		b.Logf("Warning: AdviseSequential failed: %v", err)
	}

	matcher := &LiteralMatcher{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := searcher.Search(ctx, "benchmark", matcher)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}
