package goripgrep

import (
	"os"
	"testing"
)

// BenchmarkSimpleComparison compares basic vs optimized search
func BenchmarkSimpleComparison(b *testing.B) {
	// Create test file
	testFile := "bench_test.txt"
	content := ""
	for i := 0; i < 1000; i++ {
		content += "This is a test line with BurntSushi content\n"
		content += "Another line without the target\n"
		content += "More content here\n"
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	pattern := `\\w+Sushi`

	b.Run("BasicMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile,
				WithRecursive(false),
				WithOptimization(false),
			)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("OptimizedMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile,
				WithRecursive(false),
				WithOptimization(true),
			)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("PerformanceMode", func(b *testing.B) {
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
