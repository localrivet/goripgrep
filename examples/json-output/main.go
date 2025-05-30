// Package main demonstrates JSON output capabilities of GoRipGrep.
//
// This example shows how to get structured JSON output from both
// the library API and the CLI tool for integration with other tools.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep JSON Output Examples ===")

	// Create test files for demonstration
	if err := createTestFiles(); err != nil {
		log.Fatal(err)
	}
	defer cleanupTestFiles()

	// Example 1: Library API with manual JSON marshaling
	fmt.Println("1. Library API - Manual JSON Output:")
	results, err := goripgrep.Find("error", "./test_json")
	if err != nil {
		log.Fatal(err)
	}

	// Create JSON output structure
	output := map[string]interface{}{
		"query":   "error",
		"matches": results.Matches,
		"stats":   results.Stats,
		"summary": map[string]interface{}{
			"total_matches": results.Count(),
			"files_found":   len(results.Files()),
			"has_matches":   results.HasMatches(),
		},
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonData))
	fmt.Println()

	// Example 2: Structured search results
	fmt.Println("2. Structured Search Results:")
	contextResults, err := goripgrep.Find("important", "./test_json",
		goripgrep.WithIgnoreCase(),
		goripgrep.WithContextLines(1),
		goripgrep.WithMaxResults(50))
	if err != nil {
		log.Fatal(err)
	}

	// Create detailed JSON structure
	detailedOutput := map[string]interface{}{
		"search_config": map[string]interface{}{
			"pattern":       "important",
			"ignore_case":   true,
			"context_lines": 1,
			"max_results":   50,
		},
		"results": map[string]interface{}{
			"matches": contextResults.Matches,
			"count":   contextResults.Count(),
			"files":   contextResults.Files(),
		},
		"performance": map[string]interface{}{
			"duration":      contextResults.Stats.Duration.String(),
			"files_scanned": contextResults.Stats.FilesScanned,
			"files_skipped": contextResults.Stats.FilesSkipped,
			"files_ignored": contextResults.Stats.FilesIgnored,
			"bytes_scanned": contextResults.Stats.BytesScanned,
			"matches_found": contextResults.Stats.MatchesFound,
		},
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-01T12:00:00Z",
			"version":   "1.0.0",
			"engine":    "goripgrep",
		},
	}

	detailedJSON, err := json.MarshalIndent(detailedOutput, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(detailedJSON))
	fmt.Println()

	// Example 3: Error handling in JSON format
	fmt.Println("3. Error Handling in JSON Format:")
	_, err = goripgrep.Find("[invalid", "./test_json")
	if err != nil {
		errorOutput := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"message": err.Error(),
				"type":    "regex_error",
				"code":    "INVALID_PATTERN",
			},
			"query":   "[invalid",
			"results": nil,
		}

		errorJSON, _ := json.MarshalIndent(errorOutput, "", "  ")
		fmt.Println(string(errorJSON))
	}
	fmt.Println()

	// Example 4: Batch search results
	fmt.Println("4. Batch Search Results:")
	patterns := []string{"error", "info", "warn"}
	batchResults := make(map[string]interface{})

	for _, pattern := range patterns {
		results, err := goripgrep.Find(pattern, "./test_json", goripgrep.WithIgnoreCase())
		if err != nil {
			batchResults[pattern] = map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			}
			continue
		}

		batchResults[pattern] = map[string]interface{}{
			"success":       true,
			"matches":       results.Matches,
			"match_count":   results.Count(),
			"files":         results.Files(),
			"duration":      results.Stats.Duration.String(),
			"files_scanned": results.Stats.FilesScanned,
		}
	}

	batchOutput := map[string]interface{}{
		"batch_search": true,
		"patterns":     patterns,
		"results":      batchResults,
		"summary": map[string]interface{}{
			"total_patterns": len(patterns),
			"successful":     len(batchResults),
		},
	}

	batchJSON, err := json.MarshalIndent(batchOutput, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(batchJSON))
	fmt.Println()

	// Example 5: Performance metrics in JSON
	fmt.Println("5. Performance Metrics JSON:")
	perfResults, err := goripgrep.Find("ERROR", "./test_json",
		goripgrep.WithWorkers(4),
		goripgrep.WithMaxResults(100),
		goripgrep.WithGitignore(true),
		goripgrep.WithFilePattern("*.log"))
	if err != nil {
		log.Fatal(err)
	}

	perfOutput := map[string]interface{}{
		"performance_analysis": map[string]interface{}{
			"search_config": map[string]interface{}{
				"workers":      4,
				"max_results":  100,
				"gitignore":    true,
				"file_pattern": "*.log",
			},
			"results": map[string]interface{}{
				"pattern":        "ERROR",
				"matches_found":  len(perfResults.Matches),
				"files_searched": len(perfResults.Files()),
			},
			"metrics": map[string]interface{}{
				"total_duration":   perfResults.Stats.Duration.String(),
				"duration_ns":      perfResults.Stats.Duration.Nanoseconds(),
				"files_scanned":    perfResults.Stats.FilesScanned,
				"files_per_second": float64(perfResults.Stats.FilesScanned) / perfResults.Stats.Duration.Seconds(),
				"bytes_scanned":    perfResults.Stats.BytesScanned,
				"bytes_per_second": float64(perfResults.Stats.BytesScanned) / perfResults.Stats.Duration.Seconds(),
				"matches_per_file": float64(len(perfResults.Matches)) / float64(perfResults.Stats.FilesScanned),
			},
		},
	}

	perfJSON, err := json.MarshalIndent(perfOutput, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(perfJSON))
	fmt.Println()

	// Example 6: Advanced search with multiple options
	fmt.Println("6. Advanced Search Configuration JSON:")
	advancedResults, err := goripgrep.Find("important", "./test_json",
		goripgrep.WithIgnoreCase(),
		goripgrep.WithContextLines(2),
		goripgrep.WithMaxResults(20),
		goripgrep.WithWorkers(2),
		goripgrep.WithHidden(),
		goripgrep.WithFilePattern("*.{txt,md,log}"))
	if err != nil {
		log.Fatal(err)
	}

	advancedOutput := map[string]interface{}{
		"advanced_search": map[string]interface{}{
			"configuration": map[string]interface{}{
				"pattern":        "important",
				"ignore_case":    true,
				"context_lines":  2,
				"max_results":    20,
				"workers":        2,
				"include_hidden": true,
				"file_pattern":   "*.{txt,md,log}",
			},
			"results": map[string]interface{}{
				"total_matches": advancedResults.Count(),
				"files_found":   len(advancedResults.Files()),
				"has_context":   len(advancedResults.Matches) > 0 && len(advancedResults.Matches[0].Context) > 0,
				"matches":       advancedResults.Matches,
			},
			"performance": advancedResults.Stats,
		},
	}

	advancedJSON, err := json.MarshalIndent(advancedOutput, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(advancedJSON))
	fmt.Println()

	// Example 7: CLI JSON output comparison
	fmt.Println("7. CLI Tool JSON Output:")
	fmt.Println("   To get JSON output from the CLI tool, use:")
	fmt.Println("   goripgrep --json \"pattern\" /path/to/search")
	fmt.Println("   goripgrep --json --stats \"pattern\" /path/to/search")
	fmt.Println("   goripgrep --json -C 2 -i \"pattern\" /path/to/search")
	fmt.Println()
	fmt.Println("   This produces the same structured JSON format as shown above.")
	fmt.Println("   The CLI tool automatically includes search configuration,")
	fmt.Println("   results, and performance metrics in the JSON output.")
}

func createTestFiles() error {
	// Create test directory
	if err := os.MkdirAll("./test_json", 0755); err != nil {
		return err
	}

	// Create test files with various content
	files := map[string]string{
		"./test_json/application.log": `2024-01-01 10:00:00 INFO: Application started successfully
2024-01-01 10:01:00 ERROR: Database connection failed - retrying
2024-01-01 10:02:00 WARN: High memory usage detected
2024-01-01 10:03:00 ERROR: Authentication service unavailable
2024-01-01 10:04:00 INFO: Connection restored
2024-01-01 10:05:00 ERROR: Timeout occurred during request processing`,

		"./test_json/system.log": `2024-01-01 09:00:00 INFO: System boot completed
2024-01-01 09:30:00 WARN: Disk space running low
2024-01-01 10:00:00 ERROR: Failed to mount network drive
2024-01-01 10:15:00 INFO: Network drive mounted successfully`,

		"./test_json/important.txt": `This is an important document.
It contains important information about the system.
Please read this important notice carefully.
Important: All users must update their passwords.`,

		"./test_json/config.json": `{
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "important_db"
  },
  "logging": {
    "level": "ERROR",
    "file": "/var/log/app.log"
  }
}`,

		"./test_json/readme.md": `# Important Project

This project contains important functionality.

## Error Handling
- All errors should be logged
- Important errors require immediate attention
- INFO level logging for normal operations`,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestFiles() {
	os.RemoveAll("./test_json")
}
