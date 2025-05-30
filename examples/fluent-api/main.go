// Package main demonstrates advanced usage of the GoRipGrep library.
//
// This example shows how to use the functional options API for
// complex search scenarios with multiple configuration options.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Functional Options API Examples ===")

	// Example 1: Multiple Options Combined
	fmt.Println("1. Multiple Options Combined:")
	results, err := goripgrep.Find("TODO", ".",
		goripgrep.WithIgnoreCase(),
		goripgrep.WithContextLines(2),
		goripgrep.WithFilePattern("*.go"),
		goripgrep.WithGitignore(true),
		goripgrep.WithMaxResults(50),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d TODO items in Go files\n", results.Count())
	for i, match := range results.Matches {
		if i >= 3 { // Show only first 3 matches
			fmt.Printf("... and %d more matches\n", results.Count()-3)
			break
		}
		fmt.Printf("  %s:%d:%d: %s\n", match.File, match.Line, match.Column, match.Content)
		for _, contextLine := range match.Context {
			fmt.Printf("    | %s\n", contextLine)
		}
	}
	fmt.Println()

	// Example 2: Performance-Optimized Search
	fmt.Println("2. Performance-Optimized Search:")
	results, err = goripgrep.Find("func.*main", ".",
		goripgrep.WithWorkers(4),
		goripgrep.WithBufferSize(32*1024),
		goripgrep.WithMaxResults(100),
		goripgrep.WithOptimization(true),
		goripgrep.WithFilePattern("*.{go,md}"),
		goripgrep.WithContextLines(1),
		goripgrep.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d main functions\n", results.Count())
	for i, match := range results.Matches {
		if i >= 2 { // Show only first 2 matches
			fmt.Printf("... and %d more matches\n", results.Count()-2)
			break
		}
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 3: Context-aware search with timeout
	fmt.Println("3. Context-Aware Search with Timeout:")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err = goripgrep.Find("import", ".",
		goripgrep.WithContext(ctx),
		goripgrep.WithWorkers(8),
		goripgrep.WithBufferSize(64*1024),
		goripgrep.WithOptimization(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d import statements\n", results.Count())
	fmt.Printf("Search completed in: %v\n", results.Stats.Duration)
	fmt.Printf("Files scanned: %d\n", results.Stats.FilesScanned)
	fmt.Printf("Bytes scanned: %d\n", results.Stats.BytesScanned)
	fmt.Println()

	// Example 4: Different search types
	fmt.Println("4. Different Search Types:")

	// Case-insensitive Unicode search
	fmt.Println("  Unicode Search:")
	unicodeResults, err := goripgrep.Find("café", ".", goripgrep.WithIgnoreCase())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Found %d Unicode matches\n", unicodeResults.Count())

	// Regex search
	fmt.Println("  Regex Search:")
	regexResults, err := goripgrep.Find(`func\s+\w+\(`, ".")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Found %d function definitions\n", regexResults.Count())

	// Literal search (optimized)
	fmt.Println("  Literal Search:")
	literalResults, err := goripgrep.Find("package main", ".")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("    Found %d 'package main' declarations\n", literalResults.Count())
	fmt.Println()

	// Example 5: File filtering options
	fmt.Println("5. File Filtering Options:")

	// Include hidden files
	hiddenResults, err := goripgrep.Find("config", ".",
		goripgrep.WithHidden(),
		goripgrep.WithIgnoreCase(),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Found %d matches including hidden files\n", hiddenResults.Count())

	// Follow symlinks
	symlinkResults, err := goripgrep.Find("test", ".",
		goripgrep.WithSymlinks(),
		goripgrep.WithFilePattern("*.txt"),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Found %d matches following symlinks\n", symlinkResults.Count())

	// Disable gitignore
	allResults, err := goripgrep.Find("node_modules", ".",
		goripgrep.WithGitignore(false),
		goripgrep.WithMaxResults(10),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Found %d matches ignoring .gitignore\n", allResults.Count())
	fmt.Println()

	// Example 6: Error handling and edge cases
	fmt.Println("6. Error Handling:")

	// Search in non-existent directory
	_, err = goripgrep.Find("test", "/non/existent/path")
	if err != nil {
		fmt.Printf("  Expected error for non-existent path: %v\n", err)
	}

	// Search with invalid regex
	_, err = goripgrep.Find("[invalid", ".")
	if err != nil {
		fmt.Printf("  Expected error for invalid regex: %v\n", err)
	}

	// Search with empty pattern
	emptyResults, err := goripgrep.Find("", ".")
	if err != nil {
		fmt.Printf("  Error with empty pattern: %v\n", err)
	} else {
		fmt.Printf("  Empty pattern returned %d results\n", emptyResults.Count())
	}

	// Example 7: Combining all options
	fmt.Println("\n7. Kitchen Sink - All Options Combined:")
	kitchenSinkResults, err := goripgrep.Find("error", ".",
		goripgrep.WithContext(context.Background()),
		goripgrep.WithWorkers(4),
		goripgrep.WithBufferSize(64*1024),
		goripgrep.WithMaxResults(20),
		goripgrep.WithOptimization(true),
		goripgrep.WithGitignore(true),
		goripgrep.WithIgnoreCase(),
		goripgrep.WithHidden(),
		goripgrep.WithSymlinks(),
		goripgrep.WithFilePattern("*.{go,txt,md}"),
		goripgrep.WithContextLines(2),
		goripgrep.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Kitchen sink search found %d matches in %d files\n",
		kitchenSinkResults.Count(), len(kitchenSinkResults.Files()))
	fmt.Printf("Performance: %v, %d files scanned, %d bytes processed\n",
		kitchenSinkResults.Stats.Duration,
		kitchenSinkResults.Stats.FilesScanned,
		kitchenSinkResults.Stats.BytesScanned)
}
