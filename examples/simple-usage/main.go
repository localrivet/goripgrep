// Package main demonstrates simple usage of the GoRipGrep library.
//
// This example shows the most common search operations using the
// functional options API for clean, idiomatic Go code.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Simple Usage Examples ===")

	// Create test files for demonstration
	if err := createTestFiles(); err != nil {
		log.Fatal(err)
	}
	defer cleanupTestFiles()

	// Example 1: Basic search
	fmt.Println("1. Basic Search:")
	results, err := goripgrep.Find("error", "./test_simple")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches:\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 2: Case-insensitive search
	fmt.Println("2. Case-Insensitive Search:")
	results, err = goripgrep.Find("ERROR", "./test_simple", goripgrep.WithIgnoreCase())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches (case-insensitive):\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 3: Search with context lines
	fmt.Println("3. Search with Context Lines:")
	results, err = goripgrep.Find("important", "./test_simple", goripgrep.WithContextLines(1))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches with context:\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
		for _, context := range match.Context {
			fmt.Printf("    Context: %s\n", context)
		}
	}
	fmt.Println()

	// Example 4: Search specific file types
	fmt.Println("4. Search Specific File Types:")
	results, err = goripgrep.Find("function", "./test_simple", goripgrep.WithFilePattern("*.js"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches in JavaScript files:\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 5: Multiple options combined
	fmt.Println("5. Multiple Options Combined:")
	results, err = goripgrep.Find("test", "./test_simple",
		goripgrep.WithIgnoreCase(),
		goripgrep.WithContextLines(1),
		goripgrep.WithMaxResults(5),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches (limited to 5, case-insensitive, with context):\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 6: Performance information
	fmt.Println("6. Performance Information:")
	fmt.Printf("Search completed in: %v\n", results.Stats.Duration)
	fmt.Printf("Files scanned: %d\n", results.Stats.FilesScanned)
	fmt.Printf("Bytes scanned: %d\n", results.Stats.BytesScanned)
	fmt.Printf("Files with matches: %d\n", len(results.Files()))
}

func createTestFiles() error {
	// Create test directory
	if err := os.MkdirAll("./test_simple", 0755); err != nil {
		return err
	}

	// Create test files with various content
	files := map[string]string{
		"./test_simple/app.log": `2024-01-01 10:00:00 INFO: Application started
2024-01-01 10:01:00 ERROR: Database connection failed
2024-01-01 10:02:00 WARN: High memory usage
2024-01-01 10:03:00 ERROR: Authentication failed`,

		"./test_simple/system.log": `2024-01-01 09:00:00 INFO: System boot
2024-01-01 09:30:00 ERROR: Disk space low
2024-01-01 10:00:00 INFO: Network connected`,

		"./test_simple/important.txt": `This is an important document.
It contains important information.
Please read this important notice.`,

		"./test_simple/script.js": `function testFunction() {
    console.log("Test function called");
    return true;
}

function anotherFunction() {
    console.log("Another function");
}`,

		"./test_simple/data.txt": `Test data file
Contains test information
Used for testing purposes`,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestFiles() {
	os.RemoveAll("./test_simple")
}
