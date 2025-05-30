package goripgrep

import (
	"testing"
)

func TestIsLiteralPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		expected bool
	}{
		{"hello", true},
		{"hello world", true},
		{"hello.world", false},
		{"hello*", false},
		{"hello+", false},
		{"hello?", false},
		{"^hello", false},
		{"hello$", false},
		{"hello|world", false},
		{"hello(world)", false},
		{"hello[world]", false},
		{"hello{1,2}", false},
		{"hello\\world", false},
		{"", true},
		{"123", true},
		{"test_file", true},
		{"test-file", true},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			result := isLiteralPattern(test.pattern)
			if result != test.expected {
				t.Errorf("isLiteralPattern(%q) = %v, expected %v", test.pattern, result, test.expected)
			}
		})
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		filePath string
		expected bool
	}{
		{"test.txt", false},
		{"test.go", false},
		{"test.js", false},
		{"test.py", false},
		{"test.md", false},
		{"test.json", false},
		{"test.xml", false},
		{"test.exe", true},
		{"test.dll", true},
		{"test.so", true},
		{"test.dylib", true},
		{"test.bin", true},
		{"test.dat", true},
		{"test.db", true},
		{"test.sqlite", true},
		{"test.jpg", true},
		{"test.jpeg", true},
		{"test.png", true},
		{"test.gif", true},
		{"test.pdf", true},
		{"test.doc", true},
		{"test.docx", true},
		{"test.xls", true},
		{"test.zip", true},
		{"test.tar", true},
		{"test.gz", true},
		{"test.rar", true},
		{"test.mp3", true},
		{"test.mp4", true},
		{"test.avi", true},
		{"test.mov", true},
		{"test.o", true},
		{"test.a", true},
		{"test.lib", true},
		{"/path/to/test.TXT", false}, // Case insensitive
		{"/path/to/test.EXE", true},  // Case insensitive
		{"test", false},              // No extension
		{".hidden", false},           // Hidden file without extension
	}

	for _, test := range tests {
		t.Run(test.filePath, func(t *testing.T) {
			result := isBinaryFile(test.filePath)
			if result != test.expected {
				t.Errorf("isBinaryFile(%q) = %v, expected %v", test.filePath, result, test.expected)
			}
		})
	}
}

func TestSearchArgs(t *testing.T) {
	// Test SearchArgs struct creation and field access
	args := SearchArgs{
		Path:        "/test/path",
		Pattern:     "test pattern",
		FilePattern: stringPtr("*.go"),
		IgnoreCase:  boolPtr(true),
		MaxResults:  intPtr(100),
	}

	if args.Path != "/test/path" {
		t.Errorf("Expected Path to be '/test/path', got %q", args.Path)
	}

	if args.Pattern != "test pattern" {
		t.Errorf("Expected Pattern to be 'test pattern', got %q", args.Pattern)
	}

	if args.FilePattern == nil || *args.FilePattern != "*.go" {
		t.Error("Expected FilePattern to be '*.go'")
	}

	if args.IgnoreCase == nil || !*args.IgnoreCase {
		t.Error("Expected IgnoreCase to be true")
	}

	if args.MaxResults == nil || *args.MaxResults != 100 {
		t.Error("Expected MaxResults to be 100")
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}
