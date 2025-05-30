// Package main demonstrates context lines functionality of GoRipGrep.
//
// This example shows how to use context lines to get surrounding text
// around matches for better understanding of search results.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Context Lines Examples ===")

	// Create test files for demonstration
	if err := createTestFiles(); err != nil {
		log.Fatal(err)
	}
	defer cleanupTestFiles()

	// Example 1: Basic context lines
	fmt.Println("1. Basic Context Lines (2 lines before/after):")
	results, err := goripgrep.Find("important", "./test_context", goripgrep.WithContextLines(2))
	if err != nil {
		log.Fatal(err)
	}

	for _, match := range results.Matches {
		fmt.Printf("\n%s:%d: %s\n", match.File, match.Line, match.Content)

		// Display context lines
		contextLines := len(match.Context)
		if contextLines > 0 {
			beforeLines := contextLines / 2

			// Before context
			for i := 0; i < beforeLines; i++ {
				lineNum := match.Line - beforeLines + i
				fmt.Printf("%s:%d-: %s\n", match.File, lineNum, match.Context[i])
			}

			// The match line (highlighted)
			fmt.Printf("%s:%d:> %s\n", match.File, match.Line, match.Content)

			// After context
			for i := beforeLines + 1; i < contextLines; i++ {
				lineNum := match.Line + i - beforeLines
				fmt.Printf("%s:%d+: %s\n", match.File, lineNum, match.Context[i])
			}
		}
	}
	fmt.Println()

	// Example 2: Different context sizes
	fmt.Println("2. Different Context Sizes:")

	contextSizes := []int{0, 1, 3, 5}
	for _, size := range contextSizes {
		fmt.Printf("   Context size %d:\n", size)
		results, err := goripgrep.Find("error", "./test_context", goripgrep.WithContextLines(size))
		if err != nil {
			log.Fatal(err)
		}

		if len(results.Matches) > 0 {
			match := results.Matches[0] // Show first match
			fmt.Printf("     %s:%d: %s\n", match.File, match.Line, match.Content)
			fmt.Printf("     Context lines: %d\n", len(match.Context))
		}
	}
	fmt.Println()

	// Example 3: Multiple options with context
	fmt.Println("3. Multiple Options with Context:")
	results, err = goripgrep.Find("WARNING", "./test_context",
		goripgrep.WithContextLines(3),
		goripgrep.WithIgnoreCase(),
		goripgrep.WithFilePattern("*.log"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches with 3 lines of context:\n", results.Count())
	for i, match := range results.Matches {
		if i >= 2 { // Show only first 2 matches
			fmt.Printf("... and %d more matches\n", results.Count()-2)
			break
		}

		fmt.Printf("\n%s:%d:\n", match.File, match.Line)

		// Show context with line numbers
		if len(match.Context) > 0 {
			contextBefore := len(match.Context) / 2

			for j, contextLine := range match.Context {
				var lineNum int
				var marker string

				if j < contextBefore {
					lineNum = match.Line - contextBefore + j
					marker = "-"
				} else if j == contextBefore {
					lineNum = match.Line
					marker = ":"
				} else {
					lineNum = match.Line + j - contextBefore
					marker = "+"
				}

				fmt.Printf("  %d%s %s\n", lineNum, marker, contextLine)
			}
		}
	}
	fmt.Println()

	// Example 4: Context with regex patterns
	fmt.Println("4. Context with Regex Patterns:")
	regexResults, err := goripgrep.Find(`\b(ERROR|FATAL)\b`, "./test_context", goripgrep.WithContextLines(2))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d error/fatal matches with context:\n", regexResults.Count())
	for _, match := range regexResults.Matches {
		fmt.Printf("\n%s:%d: %s\n", match.File, match.Line, match.Content)

		// Show simplified context
		for i, contextLine := range match.Context {
			if i < len(match.Context)/2 {
				fmt.Printf("  | %s\n", contextLine)
			} else if i > len(match.Context)/2 {
				fmt.Printf("  | %s\n", contextLine)
			}
		}
	}
	fmt.Println()

	// Example 5: Performance comparison with/without context
	fmt.Println("5. Performance Impact of Context Lines:")

	// Without context
	start := time.Now()
	noContextResults, err := goripgrep.Find("info", "./test_context")
	if err != nil {
		log.Fatal(err)
	}
	noContextDuration := time.Since(start)

	// With context
	start = time.Now()
	withContextResults, err := goripgrep.Find("info", "./test_context", goripgrep.WithContextLines(3))
	if err != nil {
		log.Fatal(err)
	}
	withContextDuration := time.Since(start)

	fmt.Printf("   Without context: %d matches in %v\n",
		noContextResults.Count(), noContextDuration)
	fmt.Printf("   With context (3 lines): %d matches in %v\n",
		withContextResults.Count(), withContextDuration)

	if noContextDuration > 0 {
		overhead := float64(withContextDuration-noContextDuration) / float64(noContextDuration) * 100
		fmt.Printf("   Context overhead: %.1f%%\n", overhead)
	}
	fmt.Println()

	// Example 6: Large context windows
	fmt.Println("6. Large Context Windows:")
	largeContextResults, err := goripgrep.Find("function", "./test_context", goripgrep.WithContextLines(10))
	if err != nil {
		log.Fatal(err)
	}

	if len(largeContextResults.Matches) > 0 {
		match := largeContextResults.Matches[0]
		fmt.Printf("Match with 10 lines of context:\n")
		fmt.Printf("%s:%d: %s\n", match.File, match.Line, match.Content)
		fmt.Printf("Context lines provided: %d\n", len(match.Context))

		// Show first few and last few context lines
		if len(match.Context) > 6 {
			fmt.Println("First 3 context lines:")
			for i := 0; i < 3; i++ {
				fmt.Printf("  %s\n", match.Context[i])
			}
			fmt.Println("  ...")
			fmt.Println("Last 3 context lines:")
			for i := len(match.Context) - 3; i < len(match.Context); i++ {
				fmt.Printf("  %s\n", match.Context[i])
			}
		}
	}
	fmt.Println()

	// Example 7: Advanced context combinations
	fmt.Println("7. Advanced Context Combinations:")
	advancedResults, err := goripgrep.Find("important", "./test_context",
		goripgrep.WithContextLines(2),
		goripgrep.WithIgnoreCase(),
		goripgrep.WithMaxResults(5),
		goripgrep.WithFilePattern("*.{txt,go}"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d matches with advanced options:\n", advancedResults.Count())
	for _, match := range advancedResults.Matches {
		fmt.Printf("\n%s:%d: %s\n", match.File, match.Line, match.Content)
		if len(match.Context) > 0 {
			fmt.Printf("  Context: %d lines\n", len(match.Context))
		}
	}
}

func createTestFiles() error {
	// Create test directory
	if err := os.MkdirAll("./test_context", 0755); err != nil {
		return err
	}

	// Create test files with various content
	files := map[string]string{
		"./test_context/application.log": `2024-01-01 10:00:00 INFO: Application starting up
2024-01-01 10:00:01 INFO: Loading configuration files
2024-01-01 10:00:02 INFO: Connecting to database
2024-01-01 10:00:03 ERROR: Database connection failed
2024-01-01 10:00:04 INFO: Retrying database connection
2024-01-01 10:00:05 INFO: Database connected successfully
2024-01-01 10:00:06 WARNING: High memory usage detected
2024-01-01 10:00:07 INFO: Memory usage normalized
2024-01-01 10:00:08 ERROR: Authentication service timeout
2024-01-01 10:00:09 FATAL: Critical system failure
2024-01-01 10:00:10 INFO: System recovery initiated
2024-01-01 10:00:11 INFO: Recovery completed successfully`,

		"./test_context/system.log": `System boot sequence started
Loading kernel modules
Initializing hardware drivers
ERROR: Failed to load network driver
Falling back to generic driver
Network interface configured
Starting system services
WARNING: Service startup delayed
All services started successfully
System ready for user login
INFO: User session started
System running normally`,

		"./test_context/important.txt": `This document contains important information.

Section 1: Overview
This is an important section that describes the system.
It contains critical details about configuration.

Section 2: Important Notes
Please read this important notice carefully.
All users must follow these important guidelines.
Failure to comply may result in system issues.

Section 3: Configuration
The following settings are important:
- Database connection string
- API endpoint URLs
- Security certificates

Section 4: Troubleshooting
If you encounter important errors, check the logs.
Most important issues are documented here.
Contact support for important system failures.`,

		"./test_context/code.go": `package main

import (
	"fmt"
	"log"
)

// This is an important function
func importantFunction() {
	fmt.Println("Executing important logic")
	
	// Important error handling
	if err := doSomething(); err != nil {
		log.Printf("Important error occurred: %v", err)
		return
	}
	
	fmt.Println("Function completed successfully")
}

func doSomething() error {
	// Simulate some work
	return nil
}

func main() {
	fmt.Println("Starting application")
	importantFunction()
	fmt.Println("Application finished")
}`,

		"./test_context/config.json": `{
  "database": {
    "host": "localhost",
    "port": 5432,
    "important": true
  },
  "api": {
    "endpoint": "https://api.example.com",
    "timeout": 30,
    "important_headers": [
      "Authorization",
      "Content-Type"
    ]
  },
  "logging": {
    "level": "INFO",
    "important_events": true,
    "file": "/var/log/app.log"
  }
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
	os.RemoveAll("./test_context")
}
