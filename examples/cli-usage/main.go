// Package main demonstrates CLI usage of the GoRipGrep command-line tool.
//
// This example shows how to use the goripgrep CLI tool with various
// command-line flags and options, matching the examples in the help text.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func main() {
	fmt.Println("=== GoRipGrep CLI Usage Examples ===")
	fmt.Println("This example demonstrates the goripgrep command-line tool.")
	fmt.Println("Make sure you have built the CLI tool first: make cli")
	fmt.Println()

	// Check if CLI tool exists
	cliPath := "../../goripgrep"
	if _, err := os.Stat(cliPath); os.IsNotExist(err) {
		fmt.Println("CLI tool not found. Building it now...")
		if err := buildCLI(); err != nil {
			log.Fatal("Failed to build CLI:", err)
		}
	}

	// Create test files for demonstration
	if err := createTestFiles(); err != nil {
		log.Fatal(err)
	}
	defer cleanupTestFiles()

	fmt.Println("=== BASIC SEARCH EXAMPLES ===")

	// Example 1: Basic literal search
	fmt.Println("1. Basic Literal Search:")
	fmt.Println("   Command: goripgrep \"hello world\" .")
	runCLIExample(cliPath, []string{"hello world", "."})
	fmt.Println()

	// Example 2: Case-insensitive search
	fmt.Println("2. Case-Insensitive Search:")
	fmt.Println("   Command: goripgrep -i \"HELLO\" .")
	runCLIExample(cliPath, []string{"-i", "HELLO", "."})
	fmt.Println()

	// Example 3: Regular expression search
	fmt.Println("3. Regular Expression Search:")
	fmt.Println("   Command: goripgrep \"func.*main\" .")
	runCLIExample(cliPath, []string{"func.*main", "."})
	fmt.Println()

	fmt.Println("=== CONTEXT AND OUTPUT FORMATTING ===")

	// Example 4: Context lines
	fmt.Println("4. Context Lines (2 lines before/after):")
	fmt.Println("   Command: goripgrep -C 2 \"important\" .")
	runCLIExample(cliPath, []string{"-C", "2", "important", "."})
	fmt.Println()

	// Example 5: JSON output
	fmt.Println("5. JSON Output Format:")
	fmt.Println("   Command: goripgrep --json \"error\" .")
	runCLIExample(cliPath, []string{"--json", "error", "."})
	fmt.Println()

	// Example 6: Statistics only
	fmt.Println("6. Statistics Only:")
	fmt.Println("   Command: goripgrep --stats \"test\" .")
	runCLIExample(cliPath, []string{"--stats", "test", "."})
	fmt.Println()

	fmt.Println("=== FILE FILTERING ===")

	// Example 7: File pattern filtering
	fmt.Println("7. File Pattern Filtering (Go files only):")
	fmt.Println("   Command: goripgrep -g \"*.go\" \"func\" .")
	runCLIExample(cliPath, []string{"-g", "*.go", "func", "."})
	fmt.Println()

	// Example 8: Multiple file patterns
	fmt.Println("8. Multiple File Types:")
	fmt.Println("   Command: goripgrep -g \"*.{go,md}\" \"import\" .")
	runCLIExample(cliPath, []string{"-g", "*.{go,md}", "import", "."})
	fmt.Println()

	// Example 9: Include hidden files
	fmt.Println("9. Include Hidden Files:")
	fmt.Println("   Command: goripgrep --hidden \"test\" .")
	runCLIExample(cliPath, []string{"--hidden", "test", "."})
	fmt.Println()

	fmt.Println("=== PERFORMANCE AND LIMITS ===")

	// Example 10: Limit results
	fmt.Println("10. Limit Results (max 5 matches):")
	fmt.Println("    Command: goripgrep -m 5 \"e\" .")
	runCLIExample(cliPath, []string{"-m", "5", "e", "."})
	fmt.Println()

	// Example 11: Adjust workers
	fmt.Println("11. Performance Tuning (8 workers):")
	fmt.Println("    Command: goripgrep --workers 8 \"test\" .")
	runCLIExample(cliPath, []string{"--workers", "8", "test", "."})
	fmt.Println()

	// Example 12: Timeout
	fmt.Println("12. Search Timeout (5 seconds):")
	fmt.Println("    Command: goripgrep --timeout 5s \"test\" .")
	runCLIExample(cliPath, []string{"--timeout", "5s", "test", "."})
	fmt.Println()

	fmt.Println("=== ADVANCED COMBINATIONS ===")

	// Example 13: Multiple flags combined
	fmt.Println("13. Combined Flags:")
	fmt.Println("    Command: goripgrep -i -C 1 -g \"*.txt\" -m 3 \"hello\" .")
	runCLIExample(cliPath, []string{"-i", "-C", "1", "-g", "*.txt", "-m", "3", "hello", "."})
	fmt.Println()

	// Example 14: Complex search with all options
	fmt.Println("14. Kitchen Sink (all options):")
	fmt.Println("    Command: goripgrep -i -C 1 --json --workers 2 --timeout 10s \"error\" .")
	runCLIExample(cliPath, []string{"-i", "-C", "1", "--json", "--workers", "2", "--timeout", "10s", "error", "."})
	fmt.Println()

	fmt.Println("=== GITIGNORE AND SYMLINKS ===")

	// Example 15: Disable gitignore
	fmt.Println("15. Disable Gitignore:")
	fmt.Println("    Command: goripgrep --no-gitignore \"test\" .")
	runCLIExample(cliPath, []string{"--gitignore=false", "test", "."})
	fmt.Println()

	// Example 16: Follow symlinks
	fmt.Println("16. Follow Symlinks:")
	fmt.Println("    Command: goripgrep --follow \"test\" .")
	runCLIExample(cliPath, []string{"--follow", "test", "."})
	fmt.Println()

	fmt.Println("=== UTILITY COMMANDS ===")

	// Example 17: Version information
	fmt.Println("17. Version Information:")
	fmt.Println("    Command: goripgrep version")
	runCLIExample(cliPath, []string{"version"})
	fmt.Println()

	// Example 18: Benchmark command
	fmt.Println("18. Performance Benchmark:")
	fmt.Println("    Command: goripgrep bench \"test\" .")
	runCLIExample(cliPath, []string{"bench", "test", "."})
	fmt.Println()

	// Example 19: Help information
	fmt.Println("19. Help Information:")
	fmt.Println("    Command: goripgrep --help")
	runCLIExample(cliPath, []string{"--help"})
	fmt.Println()

	fmt.Println("=== REAL-WORLD SCENARIOS ===")

	// Example 20: Find TODO comments in code
	fmt.Println("20. Find TODO Comments:")
	fmt.Println("    Command: goripgrep -i -g \"*.{go,js,py}\" \"TODO|FIXME|HACK\" .")
	runCLIExample(cliPath, []string{"-i", "-g", "*.{go,js,py}", "TODO|FIXME|HACK", "."})
	fmt.Println()

	// Example 21: Search log files for errors
	fmt.Println("21. Search Log Files for Errors:")
	fmt.Println("    Command: goripgrep -i -C 2 -g \"*.log\" \"error|fail|exception\" .")
	runCLIExample(cliPath, []string{"-i", "-C", "2", "-g", "*.log", "error|fail|exception", "."})
	fmt.Println()

	// Example 22: Find function definitions
	fmt.Println("22. Find Function Definitions:")
	fmt.Println("    Command: goripgrep -g \"*.go\" \"^func [a-zA-Z]\" .")
	runCLIExample(cliPath, []string{"-g", "*.go", "^func [a-zA-Z]", "."})
	fmt.Println()

	fmt.Println("=== EXAMPLES COMPLETE ===")
	fmt.Println("For more information, run: goripgrep --help")
}

func buildCLI() error {
	cmd := exec.Command("make", "cli")
	cmd.Dir = "../.."
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %w\nOutput: %s", err, output)
	}
	fmt.Println("CLI tool built successfully!")
	return nil
}

func runCLIExample(cliPath string, args []string) {
	cmd := exec.Command(cliPath, args...)
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		if len(output) > 0 {
			fmt.Printf("   Output: %s\n", string(output))
		}
		return
	}

	// Limit output for readability
	lines := strings.Split(string(output), "\n")
	maxLines := 10
	if len(lines) > maxLines {
		for i := 0; i < maxLines-1; i++ {
			if strings.TrimSpace(lines[i]) != "" {
				fmt.Printf("   %s\n", lines[i])
			}
		}
		fmt.Printf("   ... (%d more lines)\n", len(lines)-maxLines+1)
	} else {
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("   %s\n", line)
			}
		}
	}
}

func createTestFiles() error {
	// Create test directory
	if err := os.MkdirAll("./test_cli", 0755); err != nil {
		return err
	}

	// Create test files with various content
	files := map[string]string{
		"./test_cli/hello.txt": `Hello World!
This file contains hello world text.
HELLO in uppercase should match case-insensitive goripgrep.
Some other content here.`,

		"./test_cli/errors.log": `2024-01-01 10:00:00 INFO: Application started
2024-01-01 10:01:00 ERROR: Database connection failed
2024-01-01 10:02:00 WARN: Retrying connection
2024-01-01 10:03:00 ERROR: Authentication error
2024-01-01 10:04:00 INFO: Connection restored`,

		"./test_cli/sample.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func testFunction() {
	// This is a test function
	return
}`,

		"./test_cli/important.md": `# Important Document

This document contains important information.

## Section 1
Some important details here.

## Section 2  
More important content.
This line comes after important content.
Final line of the document.`,

		"./test_cli/data.json": `{
  "name": "test data",
  "values": [1, 2, 3],
  "error": false,
  "message": "hello world"
}`,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestFiles() {
	os.RemoveAll("./test_cli")
}
