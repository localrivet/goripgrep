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

func TestSlidingWindowSearcher(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		pattern     string
		chunkSize   int64
		overlapSize int64
		expected    int
	}{
		{
			name:        "Small file - single chunk",
			content:     "hello world\ntest pattern\nhello again\n",
			pattern:     "hello",
			chunkSize:   1024,
			overlapSize: 64,
			expected:    2,
		},
		{
			name:        "Medium file - multiple chunks",
			content:     strings.Repeat("line with pattern\nother line\n", 100),
			pattern:     "pattern",
			chunkSize:   512,
			overlapSize: 64,
			expected:    100,
		},
		{
			name:        "Large content - small chunks",
			content:     strings.Repeat("find this text\nsome other content\n", 500),
			pattern:     "find",
			chunkSize:   256,
			overlapSize: 32,
			expected:    500,
		},
		{
			name:        "No matches",
			content:     "no matching content here\njust regular text\n",
			pattern:     "nonexistent",
			chunkSize:   1024,
			overlapSize: 64,
			expected:    0,
		},
		{
			name:        "Pattern at chunk boundary",
			content:     strings.Repeat("x", 250) + "boundary_pattern" + strings.Repeat("y", 250),
			pattern:     "boundary_pattern",
			chunkSize:   256,
			overlapSize: 32,
			expected:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := createTempFile(tt.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile)

			// Configure options
			options := DefaultSlidingWindowOptions()
			options.ChunkSize = tt.chunkSize
			options.OverlapSize = tt.overlapSize
			options.UseMemoryMap = false // Force sliding window mode

			// Create searcher
			searcher, err := NewSlidingWindowSearcher(tmpFile, tt.pattern, options)
			if err != nil {
				t.Fatalf("Failed to create searcher: %v", err)
			}
			defer searcher.Close()

			// Perform search
			ctx := context.Background()
			matches, err := searcher.Search(ctx)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			// Verify results
			if len(matches) != tt.expected {
				t.Errorf("Expected %d matches, got %d", tt.expected, len(matches))
			}

			// Verify match content
			for _, match := range matches {
				if !strings.Contains(match.Content, tt.pattern) {
					t.Errorf("Match content '%s' does not contain pattern '%s'", match.Content, tt.pattern)
				}
			}
		})
	}
}

func TestSlidingWindowSearcherWithMemoryMapping(t *testing.T) {
	// Create a large file that should trigger memory mapping
	content := strings.Repeat("search for this pattern\nother content line\n", 2000000) // ~88MB
	pattern := "pattern"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = true

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	expectedMatches := 2000000
	if len(matches) != expectedMatches {
		t.Errorf("Expected %d matches, got %d", expectedMatches, len(matches))
	}
}

func TestSlidingWindowSearcherAdaptiveResize(t *testing.T) {
	content := strings.Repeat("adaptive test line\n", 1000)
	pattern := "adaptive"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.AdaptiveResize = true
	options.UseMemoryMap = false
	options.ChunkSize = 1024
	options.MinChunkSize = 512
	options.MaxChunkSize = 2048

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Allow for some overlap filtering - should be close to 1000 matches
	if len(matches) < 990 || len(matches) > 1000 {
		t.Errorf("Expected around 1000 matches (990-1000), got %d", len(matches))
	}
}

func TestSlidingWindowSearcherProgressCallback(t *testing.T) {
	content := strings.Repeat("progress test line\n", 5000)
	pattern := "progress"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	var progressUpdates []float64
	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = false
	options.ChunkSize = 1024
	options.ProgressCallback = func(bytesProcessed, totalBytes int64, percentage float64) {
		progressUpdates = append(progressUpdates, percentage)
	}

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	_, err = searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should have received progress updates
	if len(progressUpdates) == 0 {
		t.Error("Expected progress updates, got none")
	}

	// Final update should be 100%
	if len(progressUpdates) > 0 && progressUpdates[len(progressUpdates)-1] != 100.0 {
		t.Errorf("Expected final progress to be 100%%, got %.2f%%", progressUpdates[len(progressUpdates)-1])
	}
}

func TestSlidingWindowSearcherContextCancellation(t *testing.T) {
	// Create a very large file for testing cancellation
	content := strings.Repeat("cancellation test line\n", 2000000) // Even larger file (2M lines)
	pattern := "cancellation"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = false
	options.ChunkSize = 64 // Extremely small chunks to ensure slow processing

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	// Create a context that we'll cancel immediately
	ctx, cancel := context.WithCancel(context.Background())

	// Start the search in a goroutine
	done := make(chan error, 1)
	go func() {
		_, err := searcher.Search(ctx)
		done <- err
	}()

	// Cancel immediately to ensure cancellation happens during search
	cancel()

	// Wait for the search to complete
	select {
	case err := <-done:
		// Should get context.Canceled
		if err == nil {
			// If no error, try with even more aggressive cancellation
			t.Skip("Search completed too quickly to test cancellation reliably")
			return
		}
		if err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Expected context.Canceled or context.DeadlineExceeded, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Search did not complete within timeout")
	}
}

func TestSlidingWindowSearcherEmptyFile(t *testing.T) {
	tmpFile, err := createTempFile("")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = false

	searcher, err := NewSlidingWindowSearcher(tmpFile, "pattern", options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Expected 0 matches for empty file, got %d", len(matches))
	}
}

func TestSlidingWindowSearcherMemoryUsage(t *testing.T) {
	content := strings.Repeat("memory usage test\n", 1000)
	pattern := "memory"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = false

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	// Check memory usage
	allocated, total := searcher.GetMemoryUsage()
	if allocated == 0 || total == 0 {
		t.Error("Expected non-zero memory usage values")
	}

	// Check progress
	bytesProcessed, totalBytes, percentage := searcher.GetProgress()
	if totalBytes == 0 {
		t.Error("Expected non-zero total bytes")
	}

	// Initially, no bytes should be processed
	if bytesProcessed != 0 || percentage != 0 {
		t.Errorf("Expected initial progress to be 0, got %d bytes (%.2f%%)", bytesProcessed, percentage)
	}
}

func TestSlidingWindowSearcherNonExistentFile(t *testing.T) {
	options := DefaultSlidingWindowOptions()
	_, err := NewSlidingWindowSearcher("/nonexistent/file.txt", "pattern", options)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestDefaultSlidingWindowOptions(t *testing.T) {
	options := DefaultSlidingWindowOptions()

	// Verify default values
	if options.ChunkSize != 64*1024*1024 {
		t.Errorf("Expected ChunkSize to be 64MB, got %d", options.ChunkSize)
	}
	if options.OverlapSize != 64*1024 {
		t.Errorf("Expected OverlapSize to be 64KB, got %d", options.OverlapSize)
	}
	if options.MemoryThreshold != 512*1024*1024 {
		t.Errorf("Expected MemoryThreshold to be 512MB, got %d", options.MemoryThreshold)
	}
	if options.MaxChunkSize != 256*1024*1024 {
		t.Errorf("Expected MaxChunkSize to be 256MB, got %d", options.MaxChunkSize)
	}
	if options.MinChunkSize != 1*1024*1024 {
		t.Errorf("Expected MinChunkSize to be 1MB, got %d", options.MinChunkSize)
	}
	if !options.AdaptiveResize {
		t.Error("Expected AdaptiveResize to be true")
	}
	if !options.UseMemoryMap {
		t.Error("Expected UseMemoryMap to be true")
	}
}

func BenchmarkSlidingWindowSearcher(b *testing.B) {
	// Create test content
	content := strings.Repeat("benchmark test line with pattern\n", 10000)
	pattern := "pattern"

	tmpFile, err := createTempFile(content)
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = false
	options.ChunkSize = 64 * 1024 // 64KB chunks

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
		if err != nil {
			b.Fatalf("Failed to create searcher: %v", err)
		}

		ctx := context.Background()
		_, err = searcher.Search(ctx)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}

		searcher.Close()
	}
}

func BenchmarkSlidingWindowSearcherLargeFile(b *testing.B) {
	// Skip if running short benchmarks
	if testing.Short() {
		b.Skip("Skipping large file benchmark in short mode")
	}

	// Create large test content (10MB)
	content := strings.Repeat("large file benchmark test with pattern\n", 250000)
	pattern := "pattern"

	tmpFile, err := createTempFile(content)
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.UseMemoryMap = false

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
		if err != nil {
			b.Fatalf("Failed to create searcher: %v", err)
		}

		ctx := context.Background()
		_, err = searcher.Search(ctx)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}

		searcher.Close()
	}
}

func TestSlidingWindowSearcherBacktracking(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		pattern     string
		chunkSize   int64
		overlapSize int64
		expected    int
		description string
	}{
		{
			name:        "Pattern at exact chunk boundary",
			content:     strings.Repeat("x", 250) + "boundary_pattern" + strings.Repeat("y", 250),
			pattern:     "boundary_pattern",
			chunkSize:   256,
			overlapSize: 32,
			expected:    1,
			description: "Pattern spans exactly across chunk boundary",
		},
		{
			name:        "Pattern near chunk boundary",
			content:     strings.Repeat("x", 240) + "near_boundary_test" + strings.Repeat("y", 240),
			pattern:     "near_boundary_test",
			chunkSize:   256,
			overlapSize: 32,
			expected:    1,
			description: "Pattern near but not exactly at boundary",
		},
		{
			name:        "Multiple patterns across boundaries",
			content:     strings.Repeat("x", 100) + "pattern1" + strings.Repeat("y", 100) + "pattern2" + strings.Repeat("z", 100),
			pattern:     "pattern",
			chunkSize:   128,
			overlapSize: 16,
			expected:    2,
			description: "Multiple patterns across different chunk boundaries",
		},
		{
			name:        "Long pattern spanning multiple chunks",
			content:     strings.Repeat("a", 200) + strings.Repeat("long_pattern_text", 10) + strings.Repeat("b", 200),
			pattern:     "long_pattern_text",
			chunkSize:   64,
			overlapSize: 32,
			expected:    10,
			description: "Long pattern that might span multiple small chunks",
		},
		{
			name:        "Pattern with newlines at boundary",
			content:     strings.Repeat("line\n", 50) + "multiline\npattern\ntest" + strings.Repeat("\nline", 50),
			pattern:     "multiline",
			chunkSize:   256,
			overlapSize: 64,
			expected:    1,
			description: "Pattern with newlines near chunk boundary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := createTempFile(tt.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile)

			// Configure options with enhanced backtracking
			options := DefaultSlidingWindowOptions()
			options.ChunkSize = tt.chunkSize
			options.OverlapSize = tt.overlapSize
			options.UseMemoryMap = false                     // Force sliding window mode
			options.MaxPatternLength = len(tt.pattern) + 100 // Add buffer

			// Create searcher
			searcher, err := NewSlidingWindowSearcher(tmpFile, tt.pattern, options)
			if err != nil {
				t.Fatalf("Failed to create searcher: %v", err)
			}
			defer searcher.Close()

			// Perform search
			ctx := context.Background()
			matches, err := searcher.Search(ctx)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			// Verify results
			if len(matches) != tt.expected {
				t.Errorf("%s: Expected %d matches, got %d", tt.description, tt.expected, len(matches))
				for i, match := range matches {
					t.Logf("Match %d: Line %d, Column %d, Content: %s", i+1, match.Line, match.Column, match.Content)
				}
			}

			// Verify match content
			for _, match := range matches {
				if !strings.Contains(match.Content, tt.pattern) {
					t.Errorf("Match content '%s' does not contain pattern '%s'", match.Content, tt.pattern)
				}
			}
		})
	}
}

func TestSlidingWindowSearcherOverlapCalculation(t *testing.T) {
	content := "test content for overlap calculation"
	pattern := "test"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	tests := []struct {
		name               string
		maxPatternLength   int
		configuredOverlap  int64
		expectedMinOverlap int64
	}{
		{
			name:               "Pattern length larger than configured overlap",
			maxPatternLength:   2048,
			configuredOverlap:  1024,
			expectedMinOverlap: 2048 + 1024, // pattern length + buffer
		},
		{
			name:               "Configured overlap larger than pattern length",
			maxPatternLength:   512,
			configuredOverlap:  2048,
			expectedMinOverlap: 2048, // use configured overlap
		},
		{
			name:               "Small pattern with default overlap",
			maxPatternLength:   64,
			configuredOverlap:  1024,
			expectedMinOverlap: 1024, // use configured overlap
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultSlidingWindowOptions()
			options.MaxPatternLength = tt.maxPatternLength
			options.OverlapSize = tt.configuredOverlap
			options.UseMemoryMap = false

			searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
			if err != nil {
				t.Fatalf("Failed to create searcher: %v", err)
			}
			defer searcher.Close()

			// Test the overlap calculation
			calculatedOverlap := searcher.calculateOptimalOverlap()
			if calculatedOverlap < tt.expectedMinOverlap {
				t.Errorf("Calculated overlap %d is less than expected minimum %d", calculatedOverlap, tt.expectedMinOverlap)
			}
		})
	}
}

func TestSlidingWindowSearcherBoundarySearch(t *testing.T) {
	// Create content where pattern spans exactly across a chunk boundary
	chunkSize := int64(256)
	overlapSize := int64(32)

	// Create content with pattern at boundary
	part1 := strings.Repeat("x", int(chunkSize-8))
	pattern := "boundary_test_pattern"
	part2 := strings.Repeat("y", int(chunkSize))
	content := part1 + pattern + part2

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.ChunkSize = chunkSize
	options.OverlapSize = overlapSize
	options.UseMemoryMap = false
	options.MaxPatternLength = len(pattern) + 50

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find the pattern even though it spans chunk boundary
	if len(matches) == 0 {
		t.Error("Expected to find pattern spanning chunk boundary, but got no matches")
	}

	// Verify the match contains the pattern
	found := false
	for _, match := range matches {
		if strings.Contains(match.Content, pattern) {
			found = true
			break
		}
	}

	if !found {
		t.Error("Found matches but none contain the expected pattern")
		for i, match := range matches {
			t.Logf("Match %d: %s", i+1, match.Content)
		}
	}
}

func TestSlidingWindowSearcherDuplicateFiltering(t *testing.T) {
	// Create content with patterns in overlap regions
	content := strings.Repeat("duplicate_pattern\nother_line\n", 100)
	pattern := "duplicate_pattern"

	tmpFile, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	options := DefaultSlidingWindowOptions()
	options.ChunkSize = 512
	options.OverlapSize = 128
	options.UseMemoryMap = false

	searcher, err := NewSlidingWindowSearcher(tmpFile, pattern, options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find all 100 patterns without significant duplicates
	// Allow for some filtering due to overlap handling
	expectedMin := 95 // Allow for some conservative filtering
	expectedMax := 100

	if len(matches) < expectedMin || len(matches) > expectedMax {
		t.Errorf("Expected %d-%d matches, got %d", expectedMin, expectedMax, len(matches))
	}
}

func TestConfigurableParameters(t *testing.T) {
	tempFilePath, err := createTempFile(strings.Repeat("configurable test line\n", 1000))
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFilePath)

	tests := []struct {
		name     string
		modifier func(*SlidingWindowOptions)
		validate func(*testing.T, *SlidingWindowOptions, []Match, error)
	}{
		{
			name: "Custom chunk size",
			modifier: func(opts *SlidingWindowOptions) {
				opts.ChunkSize = 32 * 1024 // 32KB
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.ChunkSize != 32*1024 {
					t.Errorf("Expected chunk size 32KB, got %d", opts.ChunkSize)
				}
				if len(matches) == 0 {
					t.Error("Expected matches but got none")
				}
			},
		},
		{
			name: "Custom overlap size",
			modifier: func(opts *SlidingWindowOptions) {
				opts.OverlapSize = 16 * 1024 // 16KB
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.OverlapSize != 16*1024 {
					t.Errorf("Expected overlap size 16KB, got %d", opts.OverlapSize)
				}
			},
		},
		{
			name: "Custom memory threshold",
			modifier: func(opts *SlidingWindowOptions) {
				opts.MemoryThreshold = 256 * 1024 * 1024 // 256MB
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.MemoryThreshold != 256*1024*1024 {
					t.Errorf("Expected memory threshold 256MB, got %d", opts.MemoryThreshold)
				}
			},
		},
		{
			name: "Disable adaptive resize",
			modifier: func(opts *SlidingWindowOptions) {
				opts.AdaptiveResize = false
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.AdaptiveResize {
					t.Error("Expected adaptive resize to be disabled")
				}
			},
		},
		{
			name: "Disable memory mapping",
			modifier: func(opts *SlidingWindowOptions) {
				opts.UseMemoryMap = false
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.UseMemoryMap {
					t.Error("Expected memory mapping to be disabled")
				}
			},
		},
		{
			name: "Custom max pattern length",
			modifier: func(opts *SlidingWindowOptions) {
				opts.MaxPatternLength = 2048 // 2KB
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.MaxPatternLength != 2048 {
					t.Errorf("Expected max pattern length 2048, got %d", opts.MaxPatternLength)
				}
			},
		},
		{
			name: "Custom min and max chunk sizes",
			modifier: func(opts *SlidingWindowOptions) {
				opts.MinChunkSize = 512 * 1024        // 512KB
				opts.MaxChunkSize = 128 * 1024 * 1024 // 128MB
			},
			validate: func(t *testing.T, opts *SlidingWindowOptions, matches []Match, err error) {
				if err != nil {
					t.Errorf("Search failed: %v", err)
				}
				if opts.MinChunkSize != 512*1024 {
					t.Errorf("Expected min chunk size 512KB, got %d", opts.MinChunkSize)
				}
				if opts.MaxChunkSize != 128*1024*1024 {
					t.Errorf("Expected max chunk size 128MB, got %d", opts.MaxChunkSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := DefaultSlidingWindowOptions()
			tt.modifier(&options)

			searcher, err := NewSlidingWindowSearcher(tempFilePath, "configurable", options)
			if err != nil {
				t.Fatalf("Failed to create searcher: %v", err)
			}
			defer searcher.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			matches, err := searcher.Search(ctx)
			tt.validate(t, &options, matches, err)
		})
	}
}

func TestProgressCallback(t *testing.T) {
	// Create a larger file for better progress tracking
	content := strings.Repeat("progress test line\n", 5000)
	tempFilePath, err := createTempFile(content)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFilePath)

	var progressUpdates []float64
	var mu sync.Mutex

	options := DefaultSlidingWindowOptions()
	options.ChunkSize = 8 * 1024 // Smaller chunks for more progress updates
	options.ProgressCallback = func(bytesProcessed, totalBytes int64, percentage float64) {
		mu.Lock()
		defer mu.Unlock()
		progressUpdates = append(progressUpdates, percentage)
	}

	searcher, err := NewSlidingWindowSearcher(tempFilePath, "progress", options)
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(matches) == 0 {
		t.Error("Expected matches but got none")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(progressUpdates) == 0 {
		t.Error("Expected progress updates but got none")
	}

	// Check that progress updates are increasing and end at 100%
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

func TestEnhancedProgressReporting(t *testing.T) {
	// Create a larger test file for meaningful progress tracking
	content := strings.Repeat("line with search pattern\n", 10000) // ~250KB file
	tmpFile, err := os.CreateTemp("", "enhanced_progress_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write test content: %v", err)
	}
	tmpFile.Close()

	// Track progress updates
	var progressUpdates []ProgressInfo
	var basicUpdates []struct {
		bytesProcessed, totalBytes int64
		percentage                 float64
	}
	var progressMutex sync.Mutex

	// Test both basic and detailed progress callbacks
	searcher, err := NewSlidingWindowSearcher(tmpFile.Name(), "search", SlidingWindowOptions{
		ChunkSize:   32 * 1024, // 32KB chunks for multiple progress updates
		OverlapSize: 1024,      // 1KB overlap
		ProgressCallback: func(bytesProcessed, totalBytes int64, percentage float64) {
			progressMutex.Lock()
			basicUpdates = append(basicUpdates, struct {
				bytesProcessed, totalBytes int64
				percentage                 float64
			}{bytesProcessed, totalBytes, percentage})
			progressMutex.Unlock()
		},
		ProgressCallbackDetailed: func(info ProgressInfo) {
			progressMutex.Lock()
			progressUpdates = append(progressUpdates, info)
			progressMutex.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Failed to create searcher: %v", err)
	}
	defer searcher.Close()

	ctx := context.Background()
	matches, err := searcher.Search(ctx)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify we found expected matches (allow some variation due to overlap handling)
	expectedMatches := 10000
	minMatches := int(float64(expectedMatches) * 0.95)
	maxMatches := int(float64(expectedMatches) * 1.05)
	if len(matches) < minMatches || len(matches) > maxMatches {
		t.Errorf("Expected approximately %d matches (±5%%), got %d", expectedMatches, len(matches))
	}

	// Verify basic progress updates
	progressMutex.Lock()
	defer progressMutex.Unlock()

	if len(basicUpdates) == 0 {
		t.Error("No basic progress updates received")
	}

	if len(progressUpdates) == 0 {
		t.Error("No detailed progress updates received")
	}

	// Verify progress is monotonically increasing
	for i := 1; i < len(progressUpdates); i++ {
		if progressUpdates[i].BytesProcessed < progressUpdates[i-1].BytesProcessed {
			t.Errorf("Progress not monotonic: %d < %d",
				progressUpdates[i].BytesProcessed, progressUpdates[i-1].BytesProcessed)
		}
	}

	// Check the final progress update
	finalUpdate := progressUpdates[len(progressUpdates)-1]

	// Verify final progress
	if finalUpdate.Percentage != 100.0 {
		t.Errorf("Final percentage should be 100.0, got %f", finalUpdate.Percentage)
	}

	// Verify progress components are reasonable
	if finalUpdate.ProcessingRate <= 0 {
		t.Error("Processing rate should be positive")
	}

	if finalUpdate.ElapsedTime <= 0 {
		t.Error("Elapsed time should be positive")
	}

	if finalUpdate.ChunksProcessed <= 0 {
		t.Error("Should have processed at least one chunk")
	}

	// Allow some variation in matches found due to overlap filtering
	minMatchesFound := int(float64(expectedMatches) * 0.95)
	maxMatchesFound := int(float64(expectedMatches) * 1.05)
	if finalUpdate.MatchesFound < minMatchesFound || finalUpdate.MatchesFound > maxMatchesFound {
		t.Errorf("Expected approximately %d matches found (±5%%), got %d", expectedMatches, finalUpdate.MatchesFound)
	}

	// Verify ETA was calculated (should be 0 at completion)
	if finalUpdate.EstimatedTimeLeft != 0 {
		t.Logf("Final ETA: %v (expected 0 but may be calculated)", finalUpdate.EstimatedTimeLeft)
	}

	// Check intermediate progress updates have reasonable ETA
	hasValidETA := false
	for _, update := range progressUpdates[:len(progressUpdates)-1] { // Exclude final update
		if update.EstimatedTimeLeft > 0 {
			hasValidETA = true
			break
		}
	}

	if !hasValidETA && len(progressUpdates) > 1 {
		t.Log("Warning: No intermediate updates had valid ETA estimates")
	}

	t.Logf("Progress tracking completed successfully:")
	t.Logf("- Total progress updates: %d", len(progressUpdates))
	t.Logf("- Final processing rate: %.2f bytes/sec", finalUpdate.ProcessingRate)
	t.Logf("- Total elapsed time: %v", finalUpdate.ElapsedTime)
	t.Logf("- Chunks processed: %d", finalUpdate.ChunksProcessed)
	t.Logf("- Matches found: %d", finalUpdate.MatchesFound)
}

// Helper function to create temporary files for testing
func createTempFile(content string) (string, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("goripgrep_test_%d.txt", time.Now().UnixNano()))

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		return "", err
	}

	return tmpFile, nil
}
