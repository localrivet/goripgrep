package goripgrep

import (
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	t.Run("LiteralPattern", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "hello",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		if !engine.isLiteral {
			t.Error("Expected literal pattern to be detected")
		}

		if string(engine.searchBytes) != "hello" {
			t.Errorf("Expected searchBytes to be 'hello', got %q", string(engine.searchBytes))
		}
	})

	t.Run("RegexPattern", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "hello.*world",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		if engine.isLiteral {
			t.Error("Expected regex pattern to be detected")
		}

		if engine.regex == nil {
			t.Error("Expected regex to be compiled")
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		ignoreCase := true
		args := SearchArgs{
			Pattern:    "Hello",
			IgnoreCase: &ignoreCase,
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		if !engine.ignoreCase {
			t.Error("Expected case insensitive mode")
		}

		if string(engine.searchBytes) != "hello" {
			t.Errorf("Expected searchBytes to be lowercase 'hello', got %q", string(engine.searchBytes))
		}
	})

	t.Run("InvalidRegex", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "[invalid",
		}

		_, err := NewEngine(args)
		if err == nil {
			t.Error("Expected error for invalid regex pattern")
		}
	})
}

func TestEngineFindRareByte(t *testing.T) {
	args := SearchArgs{
		Pattern: "hello",
	}

	engine, err := NewEngine(args)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// The rare byte should be one of the characters in "hello"
	found := false
	for _, b := range []byte("hello") {
		if engine.rareByte == b {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Rare byte %c not found in pattern 'hello'", engine.rareByte)
	}
}

func TestEngineSearch(t *testing.T) {
	// Create a test file
	testDir, err := os.MkdirTemp("", "engine_test_*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	content := "Hello world\nThis is a test\nAnother test line\nHello again"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
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

		ctx := context.Background()
		results, err := engine.Search(ctx, testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(results))
		}

		// Check first match
		if results[0].Line != 2 {
			t.Errorf("Expected first match on line 2, got %d", results[0].Line)
		}

		if !strings.Contains(results[0].Content, "test") {
			t.Errorf("Expected match content to contain 'test', got %q", results[0].Content)
		}
	})

	t.Run("CaseInsensitiveSearch", func(t *testing.T) {
		ignoreCase := true
		args := SearchArgs{
			Pattern:    "HELLO",
			IgnoreCase: &ignoreCase,
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		results, err := engine.Search(ctx, testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(results))
		}
	})

	t.Run("RegexSearch", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "H.*o",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		results, err := engine.Search(ctx, testFile)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 matches, got %d", len(results))
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		args := SearchArgs{
			Pattern: "test",
		}

		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err = engine.Search(ctx, testFile)
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	})
}

func TestEngineOptimizedLiteralSearch(t *testing.T) {
	args := SearchArgs{
		Pattern: "test",
		Path:    ".",
	}

	engine, err := NewEngine(args)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	text := "This is a test file with test content and more test data"
	ctx := context.Background()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(text); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	results, err := engine.Search(ctx, tmpFile.Name())
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify we found the expected number of matches
	if len(results) < 2 {
		t.Errorf("Expected at least 2 matches, got %d", len(results))
	}

	// Check that matches contain the pattern
	for i, result := range results {
		if !strings.Contains(result.Content, "test") {
			t.Errorf("Match %d does not contain 'test': %s", i, result.Content)
		}
	}
}

func TestEngineFastByteScan(t *testing.T) {
	args := SearchArgs{
		Pattern: "x",
	}

	engine, err := NewEngine(args)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	testData := []byte("hello world x test x another x")
	target := byte('x')

	pos := engine.fastByteScan(testData, target)
	if pos != 12 { // First 'x' is at position 12
		t.Errorf("Expected first 'x' at position 12, got %d", pos)
	}

	// Test with no matches
	noMatchData := []byte("hello world")
	pos = engine.fastByteScan(noMatchData, target)
	if pos != -1 {
		t.Errorf("Expected -1 for no match, got %d", pos)
	}

	// Test with empty data
	emptyData := []byte("")
	pos = engine.fastByteScan(emptyData, target)
	if pos != -1 {
		t.Errorf("Expected -1 for empty data, got %d", pos)
	}
}

func TestEngineGetStats(t *testing.T) {
	args := SearchArgs{
		Pattern: "test",
	}

	engine, err := NewEngine(args)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	stats := engine.GetStats()

	// Check that all expected stats are present
	expectedKeys := []string{
		"bytes_scanned",
		"files_scanned",
		"matches_found",
		"is_literal",
		"rare_byte",
		"worker_count",
		"buffer_size",
	}

	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stat key %q not found", key)
		}
	}

	// Check specific values
	if stats["is_literal"] != true {
		t.Error("Expected is_literal to be true for literal pattern")
	}

	if stats["worker_count"].(int) <= 0 {
		t.Error("Expected worker_count to be positive")
	}

	if stats["buffer_size"].(int) <= 0 {
		t.Error("Expected buffer_size to be positive")
	}
}

func BenchmarkEngineSearch(b *testing.B) {
	// Create a test file with substantial content
	testDir, err := os.MkdirTemp("", "engine_bench_*")
	if err != nil {
		b.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	content := strings.Repeat("This is a test line with some content to search through.\n", 1000)
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.Run("LiteralSearch", func(b *testing.B) {
		args := SearchArgs{
			Pattern: "test",
		}

		engine, err := NewEngine(args)
		if err != nil {
			b.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := engine.Search(ctx, testFile)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("RegexSearch", func(b *testing.B) {
		args := SearchArgs{
			Pattern: "test.*line",
		}

		engine, err := NewEngine(args)
		if err != nil {
			b.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := engine.Search(ctx, testFile)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})
}

func TestEngineExtractLiterals(t *testing.T) {
	tests := []struct {
		pattern  string
		hasBytes bool
	}{
		{"hello", true},
		{"hello|world", true}, // Should extract common prefix or one of the alternatives
		{"^hello$", true},     // Should extract "hello"
		{".*", false},         // No useful literals
		{"[abc]", false},      // Character class, no literals
		{"a+", false},         // Quantifier, no useful literals
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			args := SearchArgs{
				Pattern: test.pattern,
			}

			engine, err := NewEngine(args)
			if err != nil {
				t.Fatalf("Failed to create engine: %v", err)
			}

			hasBytes := len(engine.searchBytes) > 0
			if hasBytes != test.hasBytes {
				t.Errorf("Pattern %q: expected hasBytes=%v, got %v", test.pattern, test.hasBytes, hasBytes)
			}
		})
	}
}

func TestEngineCompressedFileSearch(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "compressed_search_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testContent := "Hello, World!\nThis is a test file for compression search.\nLine 3 with pattern\nLine 4\n"
	pattern := "pattern"

	t.Run("GzipFileSearch", func(t *testing.T) {
		// Create gzip compressed file
		gzipFile := filepath.Join(tempDir, "test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test search
		args := SearchArgs{Pattern: pattern}
		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		results, err := engine.Search(ctx, gzipFile)
		if err != nil {
			t.Fatalf("Failed to search gzip file: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 match, got %d", len(results))
		}

		if len(results) > 0 {
			result := results[0]
			if result.Line != 3 {
				t.Errorf("Expected match on line 3, got line %d", result.Line)
			}
			if !strings.Contains(result.Content, pattern) {
				t.Errorf("Expected content to contain '%s', got '%s'", pattern, result.Content)
			}
		}
	})

	t.Run("PlainFileSearch", func(t *testing.T) {
		// Create plain text file for comparison
		plainFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(plainFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create plain test file: %v", err)
		}

		// Test search
		args := SearchArgs{Pattern: pattern}
		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		results, err := engine.Search(ctx, plainFile)
		if err != nil {
			t.Fatalf("Failed to search plain file: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 match, got %d", len(results))
		}

		if len(results) > 0 {
			result := results[0]
			if result.Line != 3 {
				t.Errorf("Expected match on line 3, got line %d", result.Line)
			}
			if !strings.Contains(result.Content, pattern) {
				t.Errorf("Expected content to contain '%s', got '%s'", pattern, result.Content)
			}
		}
	})

	t.Run("CompressedFileWithContext", func(t *testing.T) {
		// Create gzip compressed file
		gzipFile := filepath.Join(tempDir, "test_context.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test search with context lines
		contextLines := 1
		args := SearchArgs{
			Pattern:      pattern,
			ContextLines: &contextLines,
		}
		engine, err := NewEngine(args)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		ctx := context.Background()
		results, err := engine.Search(ctx, gzipFile)
		if err != nil {
			t.Fatalf("Failed to search gzip file with context: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 match, got %d", len(results))
		}

		if len(results) > 0 {
			result := results[0]
			if result.Line != 3 {
				t.Errorf("Expected match on line 3, got line %d", result.Line)
			}
			if len(result.Context) == 0 {
				t.Error("Expected context lines, got none")
			}
		}
	})
}
