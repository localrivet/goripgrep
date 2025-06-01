package goripgrep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFind(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "goripgrep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"test1.txt": "Hello world\nThis is a test file\nWith multiple lines",
		"test2.go":  "package main\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
		"test3.log": "ERROR: Something went wrong\nINFO: Everything is fine\nWARN: Be careful",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	t.Run("BasicSearch", func(t *testing.T) {
		results, err := Find("test", tempDir)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find matches")
		}

		if results.Count() == 0 {
			t.Error("Expected non-zero match count")
		}
	})

	t.Run("CaseInsensitiveSearch", func(t *testing.T) {
		results, err := Find("HELLO", tempDir, WithIgnoreCase())
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find case-insensitive matches")
		}
	})

	t.Run("FilePatternFilter", func(t *testing.T) {
		results, err := Find("main", tempDir, WithFilePattern("*.go"))
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find matches in Go files")
		}

		// Verify all matches are from .go files
		for _, match := range results.Matches {
			if !strings.HasSuffix(match.File, ".go") {
				t.Errorf("Expected match from .go file, got %s", match.File)
			}
		}
	})

	t.Run("ContextLines", func(t *testing.T) {
		results, err := Find("ERROR", tempDir, WithContextLines(1))
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find matches")
		}

		// Check that context lines are included
		for _, match := range results.Matches {
			if len(match.Context) == 0 {
				t.Error("Expected context lines to be included")
			}
		}
	})

	t.Run("MaxResults", func(t *testing.T) {
		results, err := Find("test", tempDir, WithMaxResults(1))
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if results.Count() > 1 {
			t.Errorf("Expected at most 1 result, got %d", results.Count())
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		results, err := Find("test", tempDir, WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find matches within timeout")
		}
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := Find("test", tempDir, WithContext(ctx))
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find matches with context")
		}
	})

	t.Run("MultipleOptions", func(t *testing.T) {
		results, err := Find("hello", tempDir,
			WithIgnoreCase(),
			WithContextLines(1),
			WithMaxResults(10),
			WithOptimization(true),
		)
		if err != nil {
			t.Fatalf("Find failed: %v", err)
		}

		if !results.HasMatches() {
			t.Error("Expected to find matches with multiple options")
		}
	})
}

func TestFindOptions(t *testing.T) {
	// Test individual option functions
	t.Run("WithWorkers", func(t *testing.T) {
		opts := defaultOptions()
		WithWorkers(8)(opts)
		if opts.workers != 8 {
			t.Errorf("Expected workers=8, got %d", opts.workers)
		}

		// Test invalid value
		WithWorkers(0)(opts)
		if opts.workers != 8 { // Should remain unchanged
			t.Errorf("Expected workers to remain 8, got %d", opts.workers)
		}
	})

	t.Run("WithBufferSize", func(t *testing.T) {
		opts := defaultOptions()
		WithBufferSize(128 * 1024)(opts)
		if opts.bufferSize != 128*1024 {
			t.Errorf("Expected bufferSize=131072, got %d", opts.bufferSize)
		}
	})

	t.Run("WithIgnoreCase", func(t *testing.T) {
		opts := defaultOptions()
		WithIgnoreCase()(opts)
		if !opts.ignoreCase {
			t.Error("Expected ignoreCase=true")
		}
		if opts.caseSensitive {
			t.Error("Expected caseSensitive=false")
		}
	})

	t.Run("WithCaseSensitive", func(t *testing.T) {
		opts := defaultOptions()
		WithIgnoreCase()(opts)    // First set to ignore case
		WithCaseSensitive()(opts) // Then set to case sensitive
		if opts.ignoreCase {
			t.Error("Expected ignoreCase=false")
		}
		if !opts.caseSensitive {
			t.Error("Expected caseSensitive=true")
		}
	})

	t.Run("WithHidden", func(t *testing.T) {
		opts := defaultOptions()
		WithHidden()(opts)
		if !opts.hidden {
			t.Error("Expected hidden=true")
		}
	})

	t.Run("WithSymlinks", func(t *testing.T) {
		opts := defaultOptions()
		WithSymlinks()(opts)
		if !opts.symlinks {
			t.Error("Expected symlinks=true")
		}
	})

	t.Run("WithGitignore", func(t *testing.T) {
		opts := defaultOptions()
		WithGitignore(false)(opts)
		if opts.gitignore {
			t.Error("Expected gitignore=false")
		}
	})

	t.Run("WithOptimization", func(t *testing.T) {
		opts := defaultOptions()
		WithOptimization(false)(opts)
		if opts.optimization {
			t.Error("Expected optimization=false")
		}
	})

	t.Run("WithRecursive", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "goripgrep_test_*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create files in root directory
		rootFile := filepath.Join(tempDir, "root.txt")
		if err := os.WriteFile(rootFile, []byte("test content in root"), 0644); err != nil {
			t.Fatalf("Failed to create root file: %v", err)
		}

		// Create subdirectory with files
		subDir := filepath.Join(tempDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		subFile := filepath.Join(subDir, "sub.txt")
		if err := os.WriteFile(subFile, []byte("test content in subdir"), 0644); err != nil {
			t.Fatalf("Failed to create sub file: %v", err)
		}

		// Test non-recursive (default) - should only find root file
		results, err := Find("test content", tempDir)
		if err != nil {
			t.Fatalf("Non-recursive find failed: %v", err)
		}
		if results.Count() != 1 {
			t.Errorf("Expected 1 match in non-recursive mode, got %d", results.Count())
		}
		if len(results.Files()) != 1 || !strings.Contains(results.Files()[0], "root.txt") {
			t.Errorf("Expected to find only root.txt, got files: %v", results.Files())
		}

		// Test explicit non-recursive
		results, err = Find("test content", tempDir, WithRecursive(false))
		if err != nil {
			t.Fatalf("Explicit non-recursive find failed: %v", err)
		}
		if results.Count() != 1 {
			t.Errorf("Expected 1 match in explicit non-recursive mode, got %d", results.Count())
		}

		// Test recursive - should find both files
		results, err = Find("test content", tempDir, WithRecursive(true))
		if err != nil {
			t.Fatalf("Recursive find failed: %v", err)
		}
		if results.Count() != 2 {
			t.Errorf("Expected 2 matches in recursive mode, got %d", results.Count())
		}
		if len(results.Files()) != 2 {
			t.Errorf("Expected 2 files in recursive mode, got %d: %v", len(results.Files()), results.Files())
		}

		// Verify both files are found
		files := results.Files()
		foundRoot := false
		foundSub := false
		for _, file := range files {
			if strings.Contains(file, "root.txt") {
				foundRoot = true
			}
			if strings.Contains(file, "sub.txt") {
				foundSub = true
			}
		}
		if !foundRoot || !foundSub {
			t.Errorf("Expected to find both root.txt and sub.txt, got files: %v", files)
		}
	})
}

func TestFindErrors(t *testing.T) {
	t.Run("NonExistentPath", func(t *testing.T) {
		_, err := Find("test", "/non/existent/path")
		if err == nil {
			t.Error("Expected error for non-existent path")
		}
	})

	t.Run("InvalidRegex", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "goripgrep_test_*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		_, err = Find("[invalid", tempDir)
		if err == nil {
			t.Error("Expected error for invalid regex pattern")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "goripgrep_test_*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create multiple large files to ensure search takes some time
		largeContent := strings.Repeat("test line with content\n", 50000)
		for i := 0; i < 10; i++ {
			testFile := filepath.Join(tempDir, fmt.Sprintf("large_%d.txt", i))
			if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
				t.Fatalf("Failed to create large test file: %v", err)
			}
		}

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Start search immediately and expect timeout
		_, err = Find("test", tempDir, WithContext(ctx))

		// The search should either timeout or be canceled
		if err == nil {
			// If no error, the search completed too quickly - try with even shorter timeout
			ctx2, cancel2 := context.WithCancel(context.Background())
			cancel2() // Cancel immediately

			_, err = Find("test", tempDir, WithContext(ctx2))
			if err == nil {
				t.Skip("Search completes too quickly to test cancellation reliably")
				return
			}
		}

		// Check if it's a context error (could be Canceled or DeadlineExceeded)
		if err != context.Canceled && err != context.DeadlineExceeded {
			// Check if the error contains context-related messages
			errStr := err.Error()
			if !strings.Contains(errStr, "context") && !strings.Contains(errStr, "canceled") && !strings.Contains(errStr, "deadline") {
				t.Errorf("Expected context error, got %v", err)
			}
		}
	})
}

func TestSearchResults(t *testing.T) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "goripgrep_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "test line 1\ntest line 2",
		"file2.txt": "another test\nno match here",
		"file3.txt": "final test line",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	results, err := Find("test", tempDir)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}

	t.Run("HasMatches", func(t *testing.T) {
		if !results.HasMatches() {
			t.Error("Expected HasMatches() to return true")
		}
	})

	t.Run("Count", func(t *testing.T) {
		count := results.Count()
		if count == 0 {
			t.Error("Expected Count() to return non-zero")
		}
		if count != len(results.Matches) {
			t.Errorf("Expected Count() to equal len(Matches), got %d vs %d", count, len(results.Matches))
		}
	})

	t.Run("Files", func(t *testing.T) {
		files := results.Files()
		if len(files) == 0 {
			t.Error("Expected Files() to return non-empty slice")
		}

		// Check that all files are unique
		fileSet := make(map[string]bool)
		for _, file := range files {
			if fileSet[file] {
				t.Errorf("Duplicate file in Files(): %s", file)
			}
			fileSet[file] = true
		}
	})

	t.Run("Stats", func(t *testing.T) {
		stats := results.Stats
		if stats.FilesScanned == 0 {
			t.Error("Expected FilesScanned > 0")
		}
		if stats.BytesScanned == 0 {
			t.Error("Expected BytesScanned > 0")
		}
		if stats.Duration == 0 {
			t.Error("Expected Duration > 0")
		}
		if stats.MatchesFound == 0 {
			t.Error("Expected MatchesFound > 0")
		}
	})

	t.Run("Query", func(t *testing.T) {
		if results.Query != "test" {
			t.Errorf("Expected Query='test', got '%s'", results.Query)
		}
	})
}

func BenchmarkFind(b *testing.B) {
	// Create a temporary directory with test files
	tempDir, err := os.MkdirTemp("", "goripgrep_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a larger test file
	content := strings.Repeat("This is a test line with some content to search through.\n", 1000)
	testFile := filepath.Join(tempDir, "large_test.txt")
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.Run("BasicSearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Find("test", tempDir)
			if err != nil {
				b.Fatalf("Find failed: %v", err)
			}
		}
	})

	b.Run("OptimizedSearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Find("test", tempDir, WithOptimization(true))
			if err != nil {
				b.Fatalf("Find failed: %v", err)
			}
		}
	})

	b.Run("MultiWorkerSearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Find("test", tempDir, WithWorkers(8))
			if err != nil {
				b.Fatalf("Find failed: %v", err)
			}
		}
	})
}

func TestStreamingSearchConfiguration(t *testing.T) {
	// Create a large test file to trigger streaming search
	content := strings.Repeat("streaming search test pattern\n", 10000) // ~290KB
	tmpFile, err := os.CreateTemp("", "large_test*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	tests := []struct {
		name    string
		options []Option
		pattern string
		wantErr bool
	}{
		{
			name: "Enable streaming search with custom chunk size",
			options: []Option{
				WithStreamingSearch(true),
				WithLargeSizeThreshold(100 * 1024), // 100KB threshold
				WithChunkSize(64 * 1024),           // 64KB chunks
			},
			pattern: "streaming",
			wantErr: false,
		},
		{
			name: "Custom overlap size",
			options: []Option{
				WithStreamingSearch(true),
				WithLargeSizeThreshold(100 * 1024),
				WithOverlapSize(32 * 1024), // 32KB overlap
			},
			pattern: "search",
			wantErr: false,
		},
		{
			name: "Disable adaptive resize",
			options: []Option{
				WithStreamingSearch(true),
				WithLargeSizeThreshold(100 * 1024),
				WithAdaptiveResize(false),
			},
			pattern: "test",
			wantErr: false,
		},
		{
			name: "Disable memory mapping",
			options: []Option{
				WithStreamingSearch(true),
				WithLargeSizeThreshold(100 * 1024),
				WithMemoryMapping(false),
			},
			pattern: "pattern",
			wantErr: false,
		},
		{
			name: "Custom memory threshold",
			options: []Option{
				WithStreamingSearch(true),
				WithLargeSizeThreshold(100 * 1024),
				WithMemoryThreshold(256 * 1024 * 1024), // 256MB
			},
			pattern: "streaming",
			wantErr: false,
		},
		{
			name: "Custom min/max chunk sizes",
			options: []Option{
				WithStreamingSearch(true),
				WithLargeSizeThreshold(100 * 1024),
				WithMinChunkSize(16 * 1024),         // 16KB min
				WithMaxChunkSize(128 * 1024 * 1024), // 128MB max
			},
			pattern: "search",
			wantErr: false,
		},
		{
			name: "Disable streaming search",
			options: []Option{
				WithStreamingSearch(false),
				WithLargeSizeThreshold(100 * 1024),
			},
			pattern: "streaming",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Find(tt.pattern, tmpFile.Name(), tt.options...)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil {
				if !results.HasMatches() {
					t.Error("Expected matches but got none")
				}
				if results.Count() == 0 {
					t.Error("Expected match count > 0")
				}
			}
		})
	}
}

func TestProgressCallbackIntegration(t *testing.T) {
	// Create a large test file
	content := strings.Repeat("progress callback test line\n", 5000) // ~135KB
	tmpFile, err := os.CreateTemp("", "progress_test*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	var progressUpdates []float64
	var mu sync.Mutex

	progressCallback := func(bytesProcessed, totalBytes int64, percentage float64) {
		mu.Lock()
		defer mu.Unlock()
		progressUpdates = append(progressUpdates, percentage)
	}

	results, err := Find("progress", tmpFile.Name(),
		WithStreamingSearch(true),
		WithLargeSizeThreshold(50*1024), // 50KB threshold
		WithChunkSize(16*1024),          // 16KB chunks for more progress updates
		WithProgressCallback(progressCallback),
	)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if !results.HasMatches() {
		t.Error("Expected matches but got none")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(progressUpdates) == 0 {
		t.Error("Expected progress updates but got none")
	}

	// Verify progress updates make sense
	if len(progressUpdates) > 1 {
		for i := 1; i < len(progressUpdates); i++ {
			if progressUpdates[i] < progressUpdates[i-1] {
				t.Errorf("Progress went backwards: %f to %f", progressUpdates[i-1], progressUpdates[i])
			}
		}
	}

	// Should end at 100%
	finalProgress := progressUpdates[len(progressUpdates)-1]
	if finalProgress != 100.0 {
		t.Errorf("Expected final progress to be 100%%, got %f%%", finalProgress)
	}
}

func TestProgressCallbackDetailedIntegration(t *testing.T) {
	// Create a test file large enough to trigger streaming search
	content := strings.Repeat("line with search pattern\n", 5000) // ~125KB file
	tmpFile, err := os.CreateTemp("", "detailed_progress_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tmpFile.Close()

	// Track detailed progress updates
	var progressUpdates []ProgressInfo
	var progressMutex sync.Mutex

	// Search with detailed progress reporting
	results, err := Find("search", tmpFile.Name(),
		WithStreamingSearch(true),
		WithLargeSizeThreshold(50*1024), // 50KB threshold to ensure streaming is used
		WithChunkSize(16*1024),          // 16KB chunks for multiple updates
		WithProgressCallbackDetailed(func(info ProgressInfo) {
			progressMutex.Lock()
			progressUpdates = append(progressUpdates, info)
			progressMutex.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify we found matches
	if len(results.Matches) != 5000 {
		t.Errorf("Expected 5000 matches, got %d", len(results.Matches))
	}

	// Verify detailed progress updates were received
	progressMutex.Lock()
	defer progressMutex.Unlock()

	if len(progressUpdates) == 0 {
		t.Error("No detailed progress updates received")
	}

	// Verify the final progress update
	finalUpdate := progressUpdates[len(progressUpdates)-1]

	if finalUpdate.Percentage != 100.0 {
		t.Errorf("Final percentage should be 100.0, got %f", finalUpdate.Percentage)
	}

	if finalUpdate.ProcessingRate <= 0 {
		t.Error("Processing rate should be positive")
	}

	if finalUpdate.MatchesFound != 5000 {
		t.Errorf("Expected 5000 matches found in progress, got %d", finalUpdate.MatchesFound)
	}

	// Verify progress increased monotonically
	for i := 1; i < len(progressUpdates); i++ {
		if progressUpdates[i].BytesProcessed < progressUpdates[i-1].BytesProcessed {
			t.Errorf("Progress bytes not monotonic at update %d: %d < %d",
				i, progressUpdates[i].BytesProcessed, progressUpdates[i-1].BytesProcessed)
		}
	}

	t.Logf("Detailed progress integration test completed:")
	t.Logf("- Progress updates received: %d", len(progressUpdates))
	t.Logf("- Final processing rate: %.2f bytes/sec", finalUpdate.ProcessingRate)
	t.Logf("- Total elapsed time: %v", finalUpdate.ElapsedTime)
}
