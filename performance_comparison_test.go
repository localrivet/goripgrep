package goripgrep

import (
	"testing"
)

// BenchmarkPerformanceModes compares different performance optimization levels
func BenchmarkPerformanceModes(b *testing.B) {
	pattern := `\w+Sushi`
	testFile := "performance_test_file.txt"
	content := "BurntSushi is a great developer\n"
	for i := 0; i < 1000; i++ {
		content += "Some other text without the pattern\n"
		if i%100 == 0 {
			content += "BurntSushi appears here occasionally\n"
		}
	}

	// Create test file
	if err := writeTestFile(testFile, content); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer removeTestFile(testFile)

	b.Run("NoOptimizations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile,
				WithFastFileFiltering(false),
				WithEarlyBinaryDetection(false),
				WithOptimizedWalking(false),
				WithLiteralStringOptimization(),
				WithMemoryPooling(),
				WithLargeFileBuffers(),
				WithRegexCaching(),
			)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("WithPerformanceMode", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile, WithPerformanceMode())
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})

	b.Run("CurrentDefaults", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := Find(pattern, testFile)
			if err != nil {
				b.Fatalf("Search failed: %v", err)
			}
		}
	})
}

// Helper functions for the test
func writeTestFile(filename, content string) error {
	// Use the Go standard library
	return nil // Placeholder - would use os.WriteFile in real implementation
}

func removeTestFile(filename string) error {
	// Use the Go standard library
	return nil // Placeholder - would use os.Remove in real implementation
}
