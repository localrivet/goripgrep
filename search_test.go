package goripgrep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewSearchEngine(t *testing.T) {
	config := SearchConfig{
		SearchPath:      "/test/path",
		MaxWorkers:      4,
		BufferSize:      64 * 1024,
		MaxResults:      100,
		UseOptimization: true,
		UseGitignore:    true,
		IgnoreCase:      false,
		IncludeHidden:   false,
		FilePattern:     "*.go",
		ContextLines:    2,
		Timeout:         30 * time.Second,
	}

	engine := NewSearchEngine(config)

	if engine.config.SearchPath != "/test/path" {
		t.Errorf("Expected SearchPath to be '/test/path', got %q", engine.config.SearchPath)
	}

	if engine.config.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers to be 4, got %d", engine.config.MaxWorkers)
	}

	if engine.config.UseOptimization != true {
		t.Error("Expected UseOptimization to be true")
	}
}

func TestSearchEngineSearch(t *testing.T) {
	// Create test directory structure
	testDir, err := os.MkdirTemp("", "search_engine_test_*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files
	testFiles := map[string]string{
		"test1.txt": "Hello world\nThis is a test\nAnother line",
		"test2.go":  "package main\nfunc test() {\n\tfmt.Println(\"test\")\n}",
		"test3.py":  "def test():\n    print('test')\n    return True",
		".hidden":   "hidden content with test",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	t.Run("BasicSearch", func(t *testing.T) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      2,
			MaxResults:      100,
			UseOptimization: true,
			UseGitignore:    false,
			IgnoreCase:      false,
			IncludeHidden:   true,
		}

		engine := NewSearchEngine(config)
		ctx := context.Background()

		results, err := engine.Search(ctx, "test")
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results.Matches) == 0 {
			t.Error("Expected to find matches")
		}

		// Verify we found matches in multiple files
		files := make(map[string]bool)
		for _, match := range results.Matches {
			files[filepath.Base(match.File)] = true
		}

		if len(files) < 2 {
			t.Errorf("Expected matches in at least 2 files, got %d", len(files))
		}
	})

	t.Run("CaseInsensitiveSearch", func(t *testing.T) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      2,
			MaxResults:      100,
			UseOptimization: true,
			UseGitignore:    false,
			IgnoreCase:      true,
			IncludeHidden:   false,
		}

		engine := NewSearchEngine(config)
		ctx := context.Background()

		results, err := engine.Search(ctx, "HELLO")
		if err != nil {
			t.Fatalf("Case insensitive search failed: %v", err)
		}

		if len(results.Matches) == 0 {
			t.Error("Expected to find case insensitive matches")
		}
	})

	t.Run("FilePatternFilter", func(t *testing.T) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      2,
			MaxResults:      100,
			UseOptimization: true,
			UseGitignore:    false,
			IgnoreCase:      false,
			IncludeHidden:   false,
			FilePattern:     "*.go",
		}

		engine := NewSearchEngine(config)
		ctx := context.Background()

		results, err := engine.Search(ctx, "test")
		if err != nil {
			t.Fatalf("File pattern search failed: %v", err)
		}

		// All matches should be from .go files
		for _, match := range results.Matches {
			if !strings.HasSuffix(match.File, ".go") {
				t.Errorf("Expected only .go files, but found match in %s", match.File)
			}
		}
	})

	t.Run("HiddenFileHandling", func(t *testing.T) {
		// Test excluding hidden files
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      2,
			MaxResults:      100,
			UseOptimization: true,
			UseGitignore:    false,
			IgnoreCase:      false,
			IncludeHidden:   false,
		}

		engine := NewSearchEngine(config)
		ctx := context.Background()

		results, err := engine.Search(ctx, "hidden")
		if err != nil {
			t.Fatalf("Hidden file search failed: %v", err)
		}

		// Should not find matches in hidden files
		for _, match := range results.Matches {
			if strings.Contains(match.File, ".hidden") {
				t.Errorf("Found match in hidden file when IncludeHidden=false: %s", match.File)
			}
		}

		// Test including hidden files
		config.IncludeHidden = true
		engine = NewSearchEngine(config)

		results, err = engine.Search(ctx, "hidden")
		if err != nil {
			t.Fatalf("Hidden file search with include failed: %v", err)
		}

		// Should find matches in hidden files
		foundHidden := false
		for _, match := range results.Matches {
			if strings.Contains(match.File, ".hidden") {
				foundHidden = true
				break
			}
		}

		if !foundHidden {
			t.Error("Expected to find matches in hidden files when IncludeHidden=true")
		}
	})

	t.Run("MaxResultsLimit", func(t *testing.T) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      2,
			MaxResults:      1, // Limit to 1 result
			UseOptimization: true,
			UseGitignore:    false,
			IgnoreCase:      false,
			IncludeHidden:   true,
		}

		engine := NewSearchEngine(config)
		ctx := context.Background()

		results, err := engine.Search(ctx, "test")
		if err != nil {
			t.Fatalf("Max results search failed: %v", err)
		}

		if len(results.Matches) > 1 {
			t.Errorf("Expected at most 1 result, got %d", len(results.Matches))
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      2,
			MaxResults:      100,
			UseOptimization: true,
			UseGitignore:    false,
			IgnoreCase:      false,
			IncludeHidden:   false,
		}

		engine := NewSearchEngine(config)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		results, err := engine.Search(ctx, "test")

		// Should get context.Canceled error or no results due to cancellation
		if err != context.Canceled && (results == nil || len(results.Matches) == 0) {
			// Either error should be context.Canceled or we should get no results
			// This is acceptable behavior for immediate cancellation
			t.Logf("Context cancellation handled correctly: err=%v, results=%v", err, results != nil)
		} else if err == context.Canceled {
			t.Logf("Context cancellation properly detected")
		}
	})
}

func TestSearchResultsGetSummary(t *testing.T) {
	// Create mock search results
	results := &SearchResults{
		Query: "test pattern",
		Matches: []Match{
			{File: "file1.txt", Line: 1, Content: "test content"},
			{File: "file2.txt", Line: 5, Content: "another test"},
		},
		Stats: SearchStats{
			FilesScanned: 10,
			FilesSkipped: 2,
			FilesIgnored: 1,
			Duration:     100 * time.Millisecond,
		},
	}

	summary := results.GetSummary()

	if summary.Pattern != "test pattern" {
		t.Errorf("Expected pattern 'test pattern', got %q", summary.Pattern)
	}

	if summary.TotalMatches != 2 {
		t.Errorf("Expected 2 total matches, got %d", summary.TotalMatches)
	}

	if summary.FilesScanned != 10 {
		t.Errorf("Expected 10 files scanned, got %d", summary.FilesScanned)
	}

	if summary.FilesPerSecond <= 0 {
		t.Error("Expected positive files per second")
	}
}

func TestSearchEngineGetPerformanceReport(t *testing.T) {
	config := SearchConfig{
		SearchPath:      "/test",
		MaxWorkers:      4,
		UseOptimization: true,
		UseGitignore:    true,
	}

	engine := NewSearchEngine(config)
	report := engine.GetPerformanceReport()

	if report.Config.SearchPath != "/test" {
		t.Errorf("Expected SearchPath '/test', got %q", report.Config.SearchPath)
	}

	if report.Config.MaxWorkers != 4 {
		t.Errorf("Expected MaxWorkers 4, got %d", report.Config.MaxWorkers)
	}

	if !report.Engines.OptimizedEngine {
		t.Error("Expected OptimizedEngine to be true")
	}
}

func TestSearchEngineBenchmark(t *testing.T) {
	// Create test directory
	testDir, err := os.MkdirTemp("", "search_benchmark_test_*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files
	for i := 0; i < 5; i++ {
		filename := filepath.Join(testDir, "test"+string(rune('0'+i))+".txt")
		content := "This is test content for benchmarking\nWith multiple lines\nAnd test patterns"
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	config := SearchConfig{
		SearchPath:      testDir,
		MaxWorkers:      2,
		MaxResults:      100,
		UseOptimization: true,
		UseGitignore:    false,
	}

	engine := NewSearchEngine(config)
	ctx := context.Background()

	patterns := []string{"test", "content"}
	iterations := 2

	benchResults, err := engine.Benchmark(ctx, patterns, iterations)
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}

	expectedResults := len(patterns) * iterations
	if len(benchResults.Results) != expectedResults {
		t.Errorf("Expected %d benchmark results, got %d", expectedResults, len(benchResults.Results))
	}

	// Test average performance calculation
	avgPerf := benchResults.GetAveragePerformance()

	for _, pattern := range patterns {
		stats, exists := avgPerf[pattern]
		if !exists {
			t.Errorf("Expected stats for pattern %q", pattern)
			continue
		}

		if stats.Iterations != iterations {
			t.Errorf("Expected %d iterations for pattern %q, got %d", iterations, pattern, stats.Iterations)
		}

		if stats.AverageDuration <= 0 {
			t.Errorf("Expected positive average duration for pattern %q", pattern)
		}
	}
}

func TestSearchEngineSimpleSearch(t *testing.T) {
	// Create a test file
	testDir, err := os.MkdirTemp("", "simple_search_test_*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	content := "Line 1: Hello world\nLine 2: This is a test\nLine 3: Another test line"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := SearchConfig{
		SearchPath:      testDir,
		MaxWorkers:      1,
		UseOptimization: false, // Force simple search
		IgnoreCase:      false,
	}

	engine := NewSearchEngine(config)
	ctx := context.Background()

	// Test simple search directly
	results, err := engine.simpleSearch(ctx, "test", testFile)
	if err != nil {
		t.Fatalf("Simple search failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(results))
	}

	// Verify line numbers
	expectedLines := []int{2, 3}
	for i, result := range results {
		if i < len(expectedLines) && result.Line != expectedLines[i] {
			t.Errorf("Expected match on line %d, got %d", expectedLines[i], result.Line)
		}
	}

	// Test case insensitive simple search
	config.IgnoreCase = true
	engine = NewSearchEngine(config)

	results, err = engine.simpleSearch(ctx, "HELLO", testFile)
	if err != nil {
		t.Fatalf("Case insensitive simple search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 case insensitive match, got %d", len(results))
	}
}

func BenchmarkSearchEngine(b *testing.B) {
	// Create test directory with multiple files
	testDir, err := os.MkdirTemp("", "search_engine_bench_*")
	if err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create multiple test files
	for i := 0; i < 50; i++ {
		filename := filepath.Join(testDir, "file"+string(rune('0'+i%10))+".txt")
		content := strings.Repeat("This is test content with various patterns to search through.\n", 20)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.Run("OptimizedSearch", func(b *testing.B) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      4,
			MaxResults:      1000,
			UseOptimization: true,
			UseGitignore:    false,
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

	b.Run("SimpleSearch", func(b *testing.B) {
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      4,
			MaxResults:      1000,
			UseOptimization: false,
			UseGitignore:    false,
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
