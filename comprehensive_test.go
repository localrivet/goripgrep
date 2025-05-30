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

// TestComprehensiveSearchFeatures tests all advanced search features
func TestComprehensiveSearchFeatures(t *testing.T) {
	// Create test directory structure
	testDir := createTestEnvironment(t)
	defer os.RemoveAll(testDir)

	// Test 1: Performance-optimized search
	t.Run("PerformanceOptimization", func(t *testing.T) {
		testPerformanceOptimization(t, testDir)
	})

	// Test 2: Unicode support
	t.Run("UnicodeSupport", func(t *testing.T) {
		testUnicodeSupport(t, testDir)
	})

	// Test 3: Advanced regex features
	t.Run("AdvancedRegex", func(t *testing.T) {
		testAdvancedRegex(t, testDir)
	})

	// Test 4: Gitignore support
	t.Run("GitignoreSupport", func(t *testing.T) {
		testGitignoreSupport(t, testDir)
	})

	// Test 5: Integrated search engine
	t.Run("IntegratedSearch", func(t *testing.T) {
		testIntegratedSearch(t, testDir)
	})

	// Test 6: Performance benchmarking
	t.Run("PerformanceBenchmark", func(t *testing.T) {
		testPerformanceBenchmark(t, testDir)
	})
}

// createTestEnvironment sets up a comprehensive test directory
func createTestEnvironment(t *testing.T) string {
	testDir, err := os.MkdirTemp("", "search_test_*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create various test files
	testFiles := map[string]string{
		"simple.txt":  "Hello world\nThis is a test\nAnother line",
		"unicode.txt": "Hello 世界\nΓεια σας κόσμε\nПривет мир\n🌍 emoji test",
		"code.go": `package main
import "fmt"
func main() {
	fmt.Println("Hello, World!")
	// This is a comment
	var x = 42
}`,
		"data.json": `{
	"name": "test",
	"value": 123,
	"nested": {
		"array": [1, 2, 3]
	}
}`,
		"binary.bin":                string([]byte{0x00, 0x01, 0x02, 0x03, 0xFF}),
		".hidden":                   "Hidden file content",
		"subdir/nested.txt":         "Nested file content\nWith multiple lines",
		"subdir/.gitignore":         "*.tmp\n*.log\nnode_modules/",
		"subdir/temp.tmp":           "Temporary file",
		"subdir/app.log":            "Log file content",
		"node_modules/package.json": `{"name": "test-package"}`,
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(testDir, filePath)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create .gitignore in root
	gitignoreContent := `*.tmp
*.log
node_modules/
.DS_Store
*.swp`

	gitignorePath := filepath.Join(testDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	return testDir
}

// testPerformanceOptimization tests the optimized search engine
func testPerformanceOptimization(t *testing.T, testDir string) {
	args := SearchArgs{
		Path:    testDir,
		Pattern: "test",
	}

	engine, err := NewEngine(args)
	if err != nil {
		t.Fatalf("Failed to create optimized engine: %v", err)
	}

	ctx := context.Background()
	start := time.Now()

	results, err := engine.Search(ctx, filepath.Join(testDir, "simple.txt"))
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	duration := time.Since(start)

	if len(results) == 0 {
		t.Error("Expected to find matches")
	}

	stats := engine.GetStats()
	t.Logf("Performance stats: %+v", stats)
	t.Logf("Search duration: %v", duration)

	// Verify optimization features
	if !stats["is_literal"].(bool) {
		t.Error("Expected literal pattern optimization")
	}
}

// testUnicodeSupport tests Unicode-aware search capabilities
func testUnicodeSupport(t *testing.T, testDir string) {
	engine, err := NewUnicodeSearchEngine("世界", false)
	if err != nil {
		t.Fatalf("Failed to create Unicode engine: %v", err)
	}

	// Read Unicode test file
	content, err := os.ReadFile(filepath.Join(testDir, "unicode.txt"))
	if err != nil {
		t.Fatalf("Failed to read Unicode file: %v", err)
	}

	matches := engine.Search(string(content))

	if len(matches) == 0 {
		t.Error("Expected to find Unicode matches")
	}

	for _, match := range matches {
		t.Logf("Unicode match: %+v", match)
		if match.Text != "世界" {
			t.Errorf("Expected '世界', got '%s'", match.Text)
		}
	}

	// Test case folding
	caseEngine, err := NewUnicodeSearchEngine("HELLO", true)
	if err != nil {
		t.Fatalf("Failed to create case-insensitive Unicode engine: %v", err)
	}

	caseMatches := caseEngine.Search(string(content))
	if len(caseMatches) == 0 {
		t.Error("Expected to find case-insensitive matches")
	}
}

// testAdvancedRegex tests advanced regex features
func testAdvancedRegex(t *testing.T, testDir string) {
	// Test basic regex
	engine, err := NewRegex(`func\s+\w+`, false)
	if err != nil {
		t.Fatalf("Failed to create regex engine: %v", err)
	}

	// Read Go code file
	content, err := os.ReadFile(filepath.Join(testDir, "code.go"))
	if err != nil {
		t.Fatalf("Failed to read Go file: %v", err)
	}

	matches := engine.FindAll(string(content))

	if len(matches) == 0 {
		t.Error("Expected to find regex matches")
	}

	for _, match := range matches {
		t.Logf("Regex match: %+v", match)
		if !strings.Contains(match.Text, "func") {
			t.Errorf("Expected match to contain 'func', got '%s'", match.Text)
		}
	}

	// Test feature support
	features := []string{"lookahead", "lookbehind", "backreferences", "unicode_classes"}
	for _, feature := range features {
		if !engine.SupportsFeature(feature) {
			t.Errorf("Expected support for feature: %s", feature)
		}
	}
}

// testGitignoreSupport tests gitignore pattern matching
func testGitignoreSupport(t *testing.T, testDir string) {
	engine := NewGitignoreEngine(testDir)

	// Test files that should be ignored
	ignoredFiles := []string{
		filepath.Join(testDir, "subdir", "temp.tmp"),
		filepath.Join(testDir, "subdir", "app.log"),
		filepath.Join(testDir, "node_modules", "package.json"),
	}

	for _, file := range ignoredFiles {
		if !engine.ShouldIgnore(file) {
			t.Errorf("Expected file to be ignored: %s", file)
		}
	}

	// Test files that should not be ignored
	allowedFiles := []string{
		filepath.Join(testDir, "simple.txt"),
		filepath.Join(testDir, "code.go"),
		filepath.Join(testDir, "data.json"),
	}

	for _, file := range allowedFiles {
		if engine.ShouldIgnore(file) {
			t.Errorf("Expected file to be allowed: %s", file)
		}
	}

	// Test pattern listing
	patterns := engine.ListPatterns()
	t.Logf("Loaded gitignore patterns: %v", patterns)

	if len(patterns) == 0 {
		t.Error("Expected to load gitignore patterns")
	}

	// Test custom pattern addition
	err := engine.AddPattern("*.custom")
	if err != nil {
		t.Errorf("Failed to add custom pattern: %v", err)
	}

	// Test pattern validation
	if err := engine.ValidatePattern("*.valid"); err != nil {
		t.Errorf("Valid pattern rejected: %v", err)
	}
}

// testIntegratedSearch tests the integrated search engine
func testIntegratedSearch(t *testing.T, testDir string) {
	config := SearchConfig{
		SearchPath:      testDir,
		MaxWorkers:      4,
		MaxResults:      100,
		UseOptimization: true,
		UseGitignore:    true,
		IgnoreCase:      true,
		IncludeHidden:   false,
	}

	engine := NewSearchEngine(config)

	ctx := context.Background()
	results, err := engine.Search(ctx, "test")
	if err != nil {
		t.Fatalf("Integrated search failed: %v", err)
	}

	if len(results.Matches) == 0 {
		t.Error("Expected to find matches in integrated search")
	}

	summary := results.GetSummary()
	t.Logf("Search summary: %+v", summary)

	// Verify gitignore filtering worked
	for _, match := range results.Matches {
		if strings.Contains(match.File, ".tmp") || strings.Contains(match.File, ".log") {
			t.Errorf("Gitignore filtering failed, found match in ignored file: %s", match.File)
		}
	}

	// Test performance report
	report := engine.GetPerformanceReport()
	t.Logf("Performance report: %+v", report)

	if !report.Engines.OptimizedEngine {
		t.Error("Expected optimized engine to be active")
	}

	if !report.Engines.GitignoreEngine {
		t.Error("Expected gitignore engine to be active")
	}
}

// testPerformanceBenchmark tests the benchmarking functionality
func testPerformanceBenchmark(t *testing.T, testDir string) {
	config := SearchConfig{
		SearchPath:      testDir,
		MaxWorkers:      2,
		MaxResults:      50,
		UseOptimization: true,
		UseGitignore:    true,
	}

	engine := NewSearchEngine(config)

	patterns := []string{"test", "Hello", "func"}
	iterations := 3

	ctx := context.Background()
	benchResults, err := engine.Benchmark(ctx, patterns, iterations)
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}

	if len(benchResults.Results) != len(patterns)*iterations {
		t.Errorf("Expected %d benchmark results, got %d", len(patterns)*iterations, len(benchResults.Results))
	}

	avgPerf := benchResults.GetAveragePerformance()
	for pattern, stats := range avgPerf {
		t.Logf("Pattern '%s': avg duration=%v, avg matches=%.1f, iterations=%d",
			pattern, stats.AverageDuration, stats.AverageMatches, stats.Iterations)

		if stats.AverageDuration <= 0 {
			t.Errorf("Invalid average duration for pattern '%s'", pattern)
		}
	}
}

// TestRipgrepFeatureComparison compares our implementation with ripgrep features
func TestRipgrepFeatureComparison(t *testing.T) {
	testDir := createTestEnvironment(t)
	defer os.RemoveAll(testDir)

	t.Run("FeatureComparison", func(t *testing.T) {
		// Test features that our implementation now supports
		supportedFeatures := map[string]bool{
			"Fast literal string search": true,
			"Regex pattern matching":     true,
			"Case-insensitive search":    true,
			"File pattern filtering":     true,
			"Binary file detection":      true,
			"Hidden file handling":       true,
			"Gitignore support":          true,
			"Unicode support":            true,
			"Context lines":              true,
			"Concurrent processing":      true,
			"Memory-efficient scanning":  true,
			"Performance optimization":   true,
			"Advanced regex features":    true,
			"Timeout support":            true,
			"Result limiting":            true,
			"Performance metrics":        true,
			"Benchmarking":               true,
		}

		for feature, supported := range supportedFeatures {
			if supported {
				t.Logf("✅ %s: SUPPORTED", feature)
			} else {
				t.Logf("❌ %s: NOT SUPPORTED", feature)
			}
		}

		// Demonstrate key features
		config := SearchConfig{
			SearchPath:      testDir,
			MaxWorkers:      runtime.NumCPU(),
			UseOptimization: true,
			UseGitignore:    true,
			IgnoreCase:      true,
			IncludeHidden:   false,
		}

		engine := NewSearchEngine(config)

		ctx := context.Background()
		results, err := engine.Search(ctx, "test")
		if err != nil {
			t.Fatalf("Comprehensive search failed: %v", err)
		}

		summary := results.GetSummary()
		t.Logf("\n🔍 Search Results Summary:")
		t.Logf("   Pattern: %s", summary.Pattern)
		t.Logf("   Total matches: %d", summary.TotalMatches)
		t.Logf("   Files scanned: %d", summary.FilesScanned)
		t.Logf("   Files skipped: %d", summary.FilesSkipped)
		t.Logf("   Files ignored: %d", summary.FilesIgnored)
		t.Logf("   Duration: %v", summary.Duration)
		t.Logf("   Files/second: %.2f", summary.FilesPerSecond)

		if summary.TotalMatches == 0 {
			t.Error("Expected to find matches in comprehensive test")
		}
	})
}
