package goripgrep

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOptimizedEngine(t *testing.T) {
	optimized := NewOptimizedEngine()

	t.Run("CPUFeatureDetection", func(t *testing.T) {
		caps := optimized.GetCapabilities()

		// Should always have basic capabilities
		if !caps["PURE_GO"] {
			t.Error("Expected PURE_GO capability to be true")
		}
		if !caps["WORD_LEVEL_OPT"] {
			t.Error("Expected WORD_LEVEL_OPT capability to be true")
		}

		// Log detected capabilities for debugging
		t.Logf("Optimized Engine Capabilities: %+v", caps)
	})

	t.Run("FastIndexByte", func(t *testing.T) {
		testCases := []struct {
			name     string
			data     []byte
			target   byte
			expected int
		}{
			{"EmptyData", []byte{}, 'a', -1},
			{"SingleByte", []byte{'a'}, 'a', 0},
			{"NotFound", []byte("hello"), 'x', -1},
			{"FirstByte", []byte("hello"), 'h', 0},
			{"LastByte", []byte("hello"), 'o', 4},
			{"MiddleByte", []byte("hello"), 'l', 2},
			{"LargeData", bytes.Repeat([]byte("abcdefgh"), 100), 'g', 6},
			{"VeryLargeData", bytes.Repeat([]byte("abcdefghijklmnop"), 1000), 'p', 15},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := optimized.FastIndexByte(tc.data, tc.target)
				if result != tc.expected {
					t.Errorf("Expected %d, got %d for target '%c' in %q",
						tc.expected, result, tc.target, string(tc.data))
				}
			})
		}
	})

	t.Run("FastCountLines", func(t *testing.T) {
		testCases := []struct {
			name     string
			data     []byte
			expected int
		}{
			{"EmptyData", []byte{}, 0},
			{"NoNewlines", []byte("hello world"), 0},
			{"SingleNewline", []byte("hello\nworld"), 1},
			{"MultipleNewlines", []byte("line1\nline2\nline3\n"), 3},
			{"OnlyNewlines", []byte("\n\n\n"), 3},
			{"LargeData", bytes.Repeat([]byte("line\n"), 1000), 1000},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := optimized.FastCountLines(tc.data)
				if result != tc.expected {
					t.Errorf("Expected %d lines, got %d in %q",
						tc.expected, result, string(tc.data))
				}
			})
		}
	})

	t.Run("BenchmarkMethods", func(t *testing.T) {
		data := []byte("The quick brown fox jumps over the lazy dog")
		target := byte('o')

		results := optimized.BenchmarkMethods(data, target)

		// Both methods should find the same positions
		if results["word_optimized"] != results["byte_by_byte"] {
			t.Errorf("Optimization methods disagree: word_optimized=%d, byte_by_byte=%d",
				results["word_optimized"], results["byte_by_byte"])
		}

		// Should find 'o' at position 12 ("brown fox")
		if results["word_optimized"] != 12 {
			t.Errorf("Expected to find 'o' at position 12, got %d", results["word_optimized"])
		}
	})
}

func TestDFACache(t *testing.T) {
	cache := NewDFACache(10, 5*time.Minute)

	t.Run("BasicCaching", func(t *testing.T) {
		pattern := "test.*pattern"
		flags := "(?i)"

		// First compilation should be a cache miss
		regex1, err := cache.GetOrCompile(pattern, flags)
		if err != nil {
			t.Fatalf("Failed to compile regex: %v", err)
		}

		stats := cache.Stats()
		if stats.Misses != 1 {
			t.Errorf("Expected 1 miss, got %d", stats.Misses)
		}

		// Second compilation should be a cache hit
		regex2, err := cache.GetOrCompile(pattern, flags)
		if err != nil {
			t.Fatalf("Failed to compile regex: %v", err)
		}

		if regex1 != regex2 {
			t.Error("Expected same regex instance from cache")
		}

		stats = cache.Stats()
		if stats.Hits != 1 {
			t.Errorf("Expected 1 hit, got %d", stats.Hits)
		}
	})

	t.Run("CacheEviction", func(t *testing.T) {
		smallCache := NewDFACache(2, 5*time.Minute)

		// Fill cache beyond capacity
		patterns := []string{"pattern1", "pattern2", "pattern3"}
		for _, pattern := range patterns {
			_, err := smallCache.GetOrCompile(pattern, "")
			if err != nil {
				t.Fatalf("Failed to compile pattern %s: %v", pattern, err)
			}
		}

		stats := smallCache.Stats()
		if stats.Evicted == 0 {
			t.Error("Expected some evictions with small cache")
		}
	})

	t.Run("InvalidPattern", func(t *testing.T) {
		_, err := cache.GetOrCompile("[invalid", "")
		if err == nil {
			t.Error("Expected error for invalid regex pattern")
		}
	})

	t.Run("CacheStats", func(t *testing.T) {
		stats := cache.Stats()

		if stats.HitRate < 0 || stats.HitRate > 1 {
			t.Errorf("Invalid hit rate: %f", stats.HitRate)
		}

		patterns := cache.GetCachedPatterns()
		if len(patterns) == 0 {
			t.Error("Expected some cached patterns")
		}
	})
}

func TestEngineOptimizations(t *testing.T) {
	// Create test file
	testDir := t.TempDir()
	testFile := filepath.Join(testDir, "test.txt")
	testContent := `The quick brown fox jumps over the lazy dog.
This is a test file with multiple lines.
Some lines contain the word "test" multiple times.
Testing optimization performance with various patterns.
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("LiteralSearch", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "test",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		matches, err := engine.Search(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(matches) == 0 {
			t.Error("Expected to find matches for 'test'")
		}

		// Verify engine detected literal pattern
		if !engine.isLiteral {
			t.Error("Expected engine to detect literal pattern")
		}
	})

	t.Run("RegexSearch", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "test.*file",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		matches, err := engine.Search(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(matches) == 0 {
			t.Error("Expected to find matches for regex pattern")
		}

		// Verify engine detected regex pattern
		if engine.isLiteral {
			t.Error("Expected engine to detect regex pattern")
		}
	})

	t.Run("CaseInsensitiveSearch", func(t *testing.T) {
		ignoreCase := true
		args := SearchArgs{
			Pattern:    "TEST",
			IgnoreCase: &ignoreCase,
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		matches, err := engine.Search(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(matches) == 0 {
			t.Error("Expected to find case-insensitive matches")
		}
	})

	t.Run("ContextLines", func(t *testing.T) {
		contextLines := 1
		args := SearchArgs{
			Pattern:      "optimization",
			ContextLines: &contextLines,
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		matches, err := engine.Search(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(matches) == 0 {
			t.Error("Expected to find matches")
		}

		// Check that context lines are included
		for _, match := range matches {
			if len(match.Context) == 0 {
				t.Error("Expected context lines to be included")
			}
		}
	})

	t.Run("EngineStats", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "test",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		// Perform search to generate stats
		_, err = engine.Search(context.Background(), testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		stats := engine.GetStats()

		// Check basic stats
		if stats["files_scanned"].(int64) == 0 {
			t.Error("Expected files_scanned > 0")
		}
		if stats["bytes_scanned"].(int64) == 0 {
			t.Error("Expected bytes_scanned > 0")
		}

		// Check optimization capabilities
		if !stats["simd_pure_go"].(bool) {
			t.Error("Expected pure Go optimization to be enabled")
		}

		// Check advanced stats
		advStats := engine.GetAdvancedStats()
		if len(advStats.SIMDCapabilities) == 0 {
			t.Error("Expected SIMD capabilities to be reported")
		}
	})
}

func BenchmarkOptimizations(b *testing.B) {
	optimized := NewOptimizedEngine()

	// Test data of various sizes
	testSizes := []int{64, 1024, 8192, 65536}

	for _, size := range testSizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte('a' + (i % 26))
		}
		// Add target byte at 75% position
		target := byte('z')
		data[size*3/4] = target

		b.Run(fmt.Sprintf("FastIndexByte_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				optimized.FastIndexByte(data, target)
			}
		})

		// Add some newlines for line counting benchmark
		dataWithNewlines := bytes.Replace(data, []byte("abcde"), []byte("ab\nde"), -1)

		b.Run(fmt.Sprintf("FastCountLines_%d", size), func(b *testing.B) {
			b.SetBytes(int64(len(dataWithNewlines)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				optimized.FastCountLines(dataWithNewlines)
			}
		})
	}
}

func BenchmarkDFACache(b *testing.B) {
	cache := NewDFACache(1000, 30*time.Minute)
	patterns := []string{
		"simple",
		"test.*pattern",
		"^start.*end$",
		"[a-z]+@[a-z]+\\.[a-z]+",
		"\\d{3}-\\d{3}-\\d{4}",
	}

	b.Run("CacheHits", func(b *testing.B) {
		// Pre-populate cache
		for _, pattern := range patterns {
			cache.GetOrCompile(pattern, "")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pattern := patterns[i%len(patterns)]
			cache.GetOrCompile(pattern, "")
		}
	})

	b.Run("CacheMisses", func(b *testing.B) {
		freshCache := NewDFACache(1000, 30*time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pattern := fmt.Sprintf("pattern_%d", i)
			freshCache.GetOrCompile(pattern, "")
		}
	})
}

func BenchmarkEngineComparison(b *testing.B) {
	// Create test data
	testData := strings.Repeat("The quick brown fox jumps over the lazy dog.\n", 1000)
	testFile := filepath.Join(b.TempDir(), "benchmark.txt")

	err := os.WriteFile(testFile, []byte(testData), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	patterns := []string{
		"fox",          // Simple literal
		"quick.*brown", // Simple regex
		"lazy|dog",     // Alternation
	}

	for _, pattern := range patterns {
		b.Run(fmt.Sprintf("Pattern_%s", pattern), func(b *testing.B) {
			args := SearchArgs{Pattern: pattern}
			engine, err := NewEngine(args)
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				engine.Search(context.Background(), testFile)
			}
		})
	}
}
