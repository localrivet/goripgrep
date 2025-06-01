package goripgrep

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// BenchmarkGoripgrepCurrentDir benchmarks our goripgrep on current directory
func BenchmarkGoripgrepCurrentDir(b *testing.B) {
	pattern := `\w+Sushi`
	dir := "."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Find(pattern, dir,
			WithFastFileFiltering(true),
			WithEarlyBinaryDetection(true),
			WithOptimizedWalking(true),
			WithRecursive(true),
		)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkGoripgrepLargeFile benchmarks our goripgrep on a large single file
func BenchmarkGoripgrepLargeFile(b *testing.B) {
	// Use the large test file we created
	if _, err := os.Stat("large_test.csv"); os.IsNotExist(err) {
		b.Skip("large_test.csv not found, skipping large file benchmark")
	}

	pattern := `\w+Sushi`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Find(pattern, "large_test.csv",
			WithFastFileFiltering(true),
			WithEarlyBinaryDetection(true),
		)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkRipgrep benchmarks ripgrep for comparison
func BenchmarkRipgrep(b *testing.B) {
	// Check if ripgrep is available
	if _, err := exec.LookPath("rg"); err != nil {
		b.Skip("ripgrep not available, skipping benchmark")
	}

	pattern := `\w+Sushi`
	dir := "."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("rg", pattern, dir)
		if err := cmd.Run(); err != nil {
			// Don't fail on exit code 1 (no matches), only on real errors
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 1 {
				b.Fatalf("Ripgrep failed: %v", err)
			}
		}
	}
}

// BenchmarkGrepRecursive benchmarks grep -r for comparison
func BenchmarkGrepRecursive(b *testing.B) {
	pattern := `\w+Sushi`
	dir := "."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := exec.Command("grep", "-rE", pattern, dir)
		cmd.Env = append(os.Environ(), "GREP_OPTIONS=")
		if err := cmd.Run(); err != nil {
			// Don't fail on exit code 1 (no matches), only on real errors
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 1 {
				b.Fatalf("Grep failed: %v", err)
			}
		}
	}
}

// BenchmarkGoripgrepEngine tests just the search engine without file walking
func BenchmarkGoripgrepEngine(b *testing.B) {
	// Create a test file
	testFile := "benchmark_test_content.txt"
	content := strings.Repeat("BurntSushi is a great developer\n", 1000)
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	pattern := `\w+Sushi`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QuickFind(pattern, testFile, false)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkGoripgrepEngineComplex tests the engine with more complex patterns
func BenchmarkGoripgrepEngineComplex(b *testing.B) {
	// Create a test file
	testFile := "benchmark_test_complex.txt"
	content := strings.Repeat("The quick brown fox jumps over the lazy dog. BurntSushi codes in Rust.\n", 1000)
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	pattern := `\b\w+[aeiou]{2,}\w+\b` // Words with 2+ consecutive vowels

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QuickFind(pattern, testFile, false)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkFileWalking tests just the file walking without search
func BenchmarkFileWalking(b *testing.B) {
	dir := "."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var count int
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				count++
			}
			return nil
		})
		if err != nil {
			b.Fatalf("Walk failed: %v", err)
		}
	}
}

// BenchmarkGoripgrepWithoutOptimizations benchmarks without optimizations for comparison
func BenchmarkGoripgrepWithoutOptimizations(b *testing.B) {
	pattern := `\w+Sushi`
	dir := "."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use basic search without optimizations
		_, err := Find(pattern, dir,
			WithFastFileFiltering(false),
			WithEarlyBinaryDetection(false),
			WithOptimizedWalking(false),
			WithRecursive(true),
		)
		if err != nil {
			b.Fatalf("Search failed: %v", err)
		}
	}
}

// BenchmarkPatternCompilation tests regex compilation overhead
func BenchmarkPatternCompilation(b *testing.B) {
	patterns := []string{
		`\w+Sushi`,
		`\b\w+[aeiou]{2,}\w+\b`,
		`(https?://)?([\da-z\.-]+)\.([a-z\.]{2,6})([/\w \.-]*)*/?`,
		`\d{4}-\d{2}-\d{2}`,
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pattern := range patterns {
			_, err := regexp.Compile(pattern)
			if err != nil {
				b.Fatalf("Failed to compile pattern %s: %v", pattern, err)
			}
		}
	}
}
