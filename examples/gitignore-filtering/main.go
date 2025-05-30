// Package main demonstrates gitignore filtering capabilities of GoRipGrep.
//
// This example shows how GoRipGrep respects .gitignore patterns and
// provides options for including or excluding ignored files.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Gitignore Filtering Examples ===")

	// Create test directory structure with .gitignore
	if err := createTestStructure(); err != nil {
		log.Fatal(err)
	}
	defer cleanupTestStructure()

	// Example 1: Search with gitignore enabled (default)
	fmt.Println("1. Search with Gitignore Enabled (Default):")
	results, err := goripgrep.Find("test content", "./test_gitignore", goripgrep.WithGitignore(true))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches (gitignore enabled)\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 2: Search with gitignore disabled
	fmt.Println("2. Search with Gitignore Disabled:")
	resultsNoGitignore, err := goripgrep.Find("test content", "./test_gitignore", goripgrep.WithGitignore(false))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches (gitignore disabled)\n", resultsNoGitignore.Count())
	for _, match := range resultsNoGitignore.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 3: Compare the difference
	fmt.Println("3. Gitignore Impact Analysis:")
	fmt.Printf("Files found with gitignore: %d\n", len(results.Files()))
	fmt.Printf("Files found without gitignore: %d\n", len(resultsNoGitignore.Files()))

	ignoredFiles := make(map[string]bool)
	for _, file := range resultsNoGitignore.Files() {
		ignoredFiles[file] = true
	}
	for _, file := range results.Files() {
		delete(ignoredFiles, file)
	}

	fmt.Printf("Files ignored by .gitignore: %d\n", len(ignoredFiles))
	for file := range ignoredFiles {
		fmt.Printf("  %s\n", file)
	}
	fmt.Println()

	// Example 4: Hidden files handling
	fmt.Println("4. Hidden Files Handling:")
	hiddenResults, err := goripgrep.Find("hidden content", "./test_gitignore",
		goripgrep.WithGitignore(true),
		goripgrep.WithHidden())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches in hidden files\n", hiddenResults.Count())
	for _, match := range hiddenResults.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 5: File pattern filtering with gitignore
	fmt.Println("5. File Pattern + Gitignore Filtering:")
	goResults, err := goripgrep.Find("package", "./test_gitignore",
		goripgrep.WithGitignore(true),
		goripgrep.WithFilePattern("*.go"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches in Go files (gitignore enabled)\n", goResults.Count())
	for _, match := range goResults.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 6: Performance impact of gitignore
	fmt.Println("6. Performance Impact of Gitignore:")
	perfResults, err := goripgrep.Find("content", "./test_gitignore", goripgrep.WithGitignore(true))
	if err != nil {
		log.Fatal(err)
	}

	perfResultsNo, err := goripgrep.Find("content", "./test_gitignore", goripgrep.WithGitignore(false))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("With gitignore: %v (%d files scanned)\n",
		perfResults.Stats.Duration, perfResults.Stats.FilesScanned)
	fmt.Printf("Without gitignore: %v (%d files scanned)\n",
		perfResultsNo.Stats.Duration, perfResultsNo.Stats.FilesScanned)

	if perfResults.Stats.FilesScanned > 0 && perfResultsNo.Stats.FilesScanned > 0 {
		reduction := float64(perfResultsNo.Stats.FilesScanned-perfResults.Stats.FilesScanned) /
			float64(perfResultsNo.Stats.FilesScanned) * 100
		fmt.Printf("Gitignore reduced files scanned by %.1f%%\n", reduction)
	}
	fmt.Println()

	// Example 7: Advanced gitignore scenarios
	fmt.Println("7. Advanced Gitignore Scenarios:")

	// Search for exception files (files that should be found despite being in ignored directories)
	exceptionResults, err := goripgrep.Find("important", "./test_gitignore",
		goripgrep.WithGitignore(true),
		goripgrep.WithFilePattern("*.log"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches in exception files\n", exceptionResults.Count())
	for _, match := range exceptionResults.Matches {
		fmt.Printf("  %s:%d: %s (exception to .gitignore)\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 8: Combining multiple options with gitignore
	fmt.Println("8. Multiple Options with Gitignore:")
	combinedResults, err := goripgrep.Find("test", "./test_gitignore",
		goripgrep.WithGitignore(true),
		goripgrep.WithIgnoreCase(),
		goripgrep.WithContextLines(1),
		goripgrep.WithMaxResults(10),
		goripgrep.WithFilePattern("*.{go,md}"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches with combined options\n", combinedResults.Count())
	for _, match := range combinedResults.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
		if len(match.Context) > 0 {
			fmt.Printf("    Context: %v\n", match.Context)
		}
	}
}

func createTestStructure() error {
	// Create test directory
	if err := os.MkdirAll("./test_gitignore", 0755); err != nil {
		return err
	}

	// Create .gitignore file
	gitignoreContent := `# Ignore build artifacts
*.o
*.so
*.exe
build/
dist/

# Ignore logs
*.log
logs/

# Ignore temporary files
*.tmp
*.temp
temp/

# Ignore IDE files
.vscode/
.idea/
*.swp

# Ignore specific files
secret.txt
config.local.json

# Ignore node_modules
node_modules/

# But don't ignore important.log
!important.log
`

	if err := os.WriteFile("./test_gitignore/.gitignore", []byte(gitignoreContent), 0644); err != nil {
		return err
	}

	// Create directory structure
	dirs := []string{
		"./test_gitignore/src",
		"./test_gitignore/build",
		"./test_gitignore/logs",
		"./test_gitignore/temp",
		"./test_gitignore/.vscode",
		"./test_gitignore/node_modules",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create test files
	files := map[string]string{
		// Regular files (should be found)
		"./test_gitignore/main.go": `package main

import "fmt"

func main() {
	fmt.Println("test content")
}`,

		"./test_gitignore/src/utils.go": `package src

// test content in utils
func Helper() string {
	return "test content"
}`,

		"./test_gitignore/README.md": `# Test Project

This file contains test content for demonstration.`,

		// Files that should be ignored
		"./test_gitignore/build/app.exe":     "binary test content",
		"./test_gitignore/logs/app.log":      "log test content",
		"./test_gitignore/temp/cache.tmp":    "temp test content",
		"./test_gitignore/secret.txt":        "secret test content",
		"./test_gitignore/config.local.json": `{"test": "content"}`,

		// IDE files (should be ignored)
		"./test_gitignore/.vscode/settings.json": `{"test": "content"}`,

		// Hidden files
		"./test_gitignore/.hidden": "hidden content",
		"./test_gitignore/.env":    "ENV=hidden content",

		// Exception file (should be found despite being in logs/)
		"./test_gitignore/important.log": "important test content",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	// Create node_modules file separately (needs subdirectory)
	if err := os.MkdirAll("./test_gitignore/node_modules/package", 0755); err != nil {
		return err
	}
	if err := os.WriteFile("./test_gitignore/node_modules/package/index.js", []byte("module test content"), 0644); err != nil {
		return err
	}

	return nil
}

func cleanupTestStructure() {
	os.RemoveAll("./test_gitignore")
}
