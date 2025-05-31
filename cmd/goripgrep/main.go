package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/localrivet/goripgrep"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	ignoreCase     bool
	contextLines   int
	maxResults     int
	workers        int
	timeout        time.Duration
	includeHidden  bool
	followSymlinks bool
	useGitignore   bool
	recursive      bool
	filePattern    string
	jsonOutput     bool
	statsOnly      bool
	version        = "dev" // Will be set during build
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "goripgrep [flags] PATTERN [PATH...]",
	Short: "A fast text search tool written in Go",
	Long: `GoRipGrep is a high-performance text search tool that provides ripgrep-like 
functionality with native Go performance optimizations. It supports literal string 
search, regular expressions, Unicode handling, and various output formats.

By default, GoRipGrep searches only the immediate directory. Use -r/--recursive 
to search subdirectories recursively.

BASIC USAGE:
  goripgrep "hello world" .                               # Search current directory only
  goripgrep -r "hello world" .                            # Search recursively
  goripgrep "func.*main" src/                             # Search src/ directory only
  goripgrep -r "func.*main" src/                          # Search src/ and subdirectories
  goripgrep "TODO" /path/to/project                       # Search specific directory

RECURSIVE SEARCH:
  goripgrep -r "pattern" .                                # Search all subdirectories
  goripgrep -r -g "*.go" "func" .                         # Recursive search in Go files
  goripgrep -r -C 2 "error" logs/                         # Recursive with context

CASE SENSITIVITY:
  goripgrep -i "Hello" .                                  # Case-insensitive search
  goripgrep -r -i "ERROR" logs/                           # Recursive case-insensitive

CONTEXT LINES:
  goripgrep -C 2 "error" .                                # Show 2 lines before/after match
  goripgrep -r -C 5 "func main" src/                      # Recursive with 5 lines context
  goripgrep -C 1 "import" *.go                            # Context for imports

FILE FILTERING:
  goripgrep -g "*.go" "func" .                            # Search only Go files
  goripgrep -r -g "*.{js,ts}" "export" .                  # Recursive search JS/TS files
  goripgrep -g "*.log" "ERROR" /var/log/                  # Search log files only
  goripgrep -r --hidden "config" .                        # Recursive including hidden files
  goripgrep -r --follow "test" .                          # Recursive following symlinks

OUTPUT FORMATS:
  goripgrep --json "error" .                              # JSON output format
  goripgrep --stats "pattern" .                           # Show only statistics
  goripgrep -r -m 10 "TODO" .                             # Recursive with 10 result limit

PERFORMANCE TUNING:
  goripgrep -r --workers 8 "pattern" .                    # Recursive with 8 workers
  goripgrep --timeout 30s "pattern" .                     # Set 30 second timeout
  goripgrep --workers 1 "complex.*regex" .                # Single worker for complex regex

GITIGNORE HANDLING:
  goripgrep -r --gitignore=false "test" .                 # Ignore .gitignore files
  goripgrep -r "secret" .                                 # Respects .gitignore by default

REAL-WORLD EXAMPLES:
  goripgrep -r -i -g "*.{go,js,py}" "TODO|FIXME" .        # Find TODO comments recursively
  goripgrep -r -C 3 -g "*.log" "ERROR|FATAL" /var/log/    # Search logs recursively
  goripgrep -r -i "password|secret|key" --hidden .        # Recursive security audit
  goripgrep -r "^func [A-Z]" -g "*.go" .                  # Find exported functions
  goripgrep -r --json -m 100 "import.*react" src/         # Find React imports recursively
  goripgrep -r -C 2 "panic\|fatal" -g "*.go" .            # Find Go panics/fatals

COMBINING FLAGS:
  goripgrep -r -i -C 2 -g "*.txt" -m 5 "hello" .          # Recursive with multiple options
  goripgrep -r --json --workers 4 --timeout 10s "error" . # Recursive performance + format
  goripgrep -r -i --hidden --follow "config" /etc/        # Comprehensive recursive search

UTILITY COMMANDS:
  goripgrep version                                       # Show version information
  goripgrep bench "pattern" .                             # Run performance benchmark
  goripgrep --help                                        # Show this help message`,
	Args: func(cmd *cobra.Command, args []string) error {
		// If no arguments, that's fine - we'll show help
		if len(args) == 0 {
			return nil
		}
		// If first argument is a known subcommand, let cobra handle it
		if args[0] == "version" || args[0] == "bench" || args[0] == "help" || args[0] == "completion" {
			return nil
		}
		// Otherwise, we need at least one argument (the pattern)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no arguments provided, show help
		if len(args) == 0 {
			return cmd.Help()
		}
		return runSearch(cmd, args)
	},
}

func init() {
	// Search behavior flags
	rootCmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "Case-insensitive search")
	rootCmd.Flags().IntVarP(&contextLines, "context", "C", 0, "Show NUM lines before and after each match")
	rootCmd.Flags().IntVarP(&maxResults, "max-count", "m", 1000, "Maximum number of results to return")
	rootCmd.Flags().IntVar(&workers, "workers", 4, "Number of concurrent workers")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Search timeout")

	// File filtering flags
	rootCmd.Flags().BoolVarP(&includeHidden, "hidden", ".", false, "Include hidden files and directories")
	rootCmd.Flags().BoolVarP(&followSymlinks, "follow", "L", false, "Follow symbolic links")
	rootCmd.Flags().BoolVar(&useGitignore, "gitignore", true, "Respect .gitignore files")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Search directories recursively")
	rootCmd.Flags().StringVarP(&filePattern, "glob", "g", "", "Only search files matching this glob pattern")

	// Output format flags
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
	rootCmd.Flags().BoolVar(&statsOnly, "stats", false, "Show only search statistics")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(benchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	// Default to current directory if no paths specified
	paths := []string{"."}
	if len(args) > 1 {
		paths = args[1:]
	}

	// Build search options
	var opts []goripgrep.Option

	if workers > 0 {
		opts = append(opts, goripgrep.WithWorkers(workers))
	}
	if maxResults > 0 {
		opts = append(opts, goripgrep.WithMaxResults(maxResults))
	}
	if ignoreCase {
		opts = append(opts, goripgrep.WithIgnoreCase())
	}
	if contextLines > 0 {
		opts = append(opts, goripgrep.WithContextLines(contextLines))
	}
	if filePattern != "" {
		opts = append(opts, goripgrep.WithFilePattern(filePattern))
	}
	if !useGitignore {
		opts = append(opts, goripgrep.WithGitignore(false))
	}
	if includeHidden {
		opts = append(opts, goripgrep.WithHidden())
	}
	if followSymlinks {
		opts = append(opts, goripgrep.WithSymlinks())
	}
	if recursive {
		opts = append(opts, goripgrep.WithRecursive(true))
	}

	// Add context for timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	opts = append(opts, goripgrep.WithContext(ctx))

	var allResults []*goripgrep.SearchResults
	var totalStats goripgrep.SearchStats

	// Search each path
	for _, path := range paths {
		results, err := goripgrep.Find(pattern, path, opts...)
		if err != nil {
			return fmt.Errorf("search failed for path %s: %w", path, err)
		}

		allResults = append(allResults, results)

		// Accumulate stats
		totalStats.FilesScanned += results.Stats.FilesScanned
		totalStats.FilesSkipped += results.Stats.FilesSkipped
		totalStats.FilesIgnored += results.Stats.FilesIgnored
		totalStats.BytesScanned += results.Stats.BytesScanned
		totalStats.MatchesFound += results.Stats.MatchesFound
		if totalStats.Duration < results.Stats.Duration {
			totalStats.Duration = results.Stats.Duration
		}
	}

	// Output results
	if statsOnly {
		return outputStats(totalStats)
	}

	if jsonOutput {
		return outputJSON(allResults, totalStats)
	}

	return outputText(allResults, totalStats)
}

func outputText(results []*goripgrep.SearchResults, stats goripgrep.SearchStats) error {
	totalMatches := 0

	for _, result := range results {
		for _, match := range result.Matches {
			totalMatches++

			// Format: file:line:column:content
			fmt.Printf("%s:%d:%d:%s\n",
				match.File,
				match.Line,
				match.Column,
				strings.TrimSpace(match.Content))

			// Show context lines if requested
			for i, contextLine := range match.Context {
				if i < contextLines { // Before context
					fmt.Printf("%s:%d-:%s\n",
						match.File,
						match.Line-contextLines+i,
						strings.TrimSpace(contextLine))
				} else if i >= contextLines+1 { // After context
					fmt.Printf("%s:%d+:%s\n",
						match.File,
						match.Line+i-contextLines,
						strings.TrimSpace(contextLine))
				}
			}
		}
	}

	// Show summary if multiple files or verbose
	if len(results) > 1 || totalMatches > 10 {
		fmt.Fprintf(os.Stderr, "\nFound %d matches in %d files (searched %d files in %v)\n",
			stats.MatchesFound,
			len(getUniqueFiles(results)),
			stats.FilesScanned,
			stats.Duration)
	}

	return nil
}

func outputJSON(results []*goripgrep.SearchResults, stats goripgrep.SearchStats) error {
	output := map[string]interface{}{
		"query":   results[0].Query, // Assuming same query for all
		"matches": getAllMatches(results),
		"stats":   stats,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputStats(stats goripgrep.SearchStats) error {
	fmt.Printf("Files scanned: %d\n", stats.FilesScanned)
	fmt.Printf("Files skipped: %d\n", stats.FilesSkipped)
	fmt.Printf("Files ignored: %d\n", stats.FilesIgnored)
	fmt.Printf("Bytes scanned: %d\n", stats.BytesScanned)
	fmt.Printf("Matches found: %d\n", stats.MatchesFound)
	fmt.Printf("Duration: %v\n", stats.Duration)
	return nil
}

func getAllMatches(results []*goripgrep.SearchResults) []goripgrep.Match {
	var allMatches []goripgrep.Match
	for _, result := range results {
		allMatches = append(allMatches, result.Matches...)
	}
	return allMatches
}

func getUniqueFiles(results []*goripgrep.SearchResults) []string {
	fileSet := make(map[string]bool)
	for _, result := range results {
		for _, file := range result.Files() {
			fileSet[file] = true
		}
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("goripgrep %s\n", version)
		fmt.Println("A fast text search tool written in Go")
		fmt.Println("https://github.com/localrivet/goripgrep")
	},
}

var benchCmd = &cobra.Command{
	Use:   "bench [flags] PATTERN [PATH...]",
	Short: "Run performance benchmarks",
	Long: `Run performance benchmarks comparing GoRipGrep against Go's standard regex.
This helps evaluate the performance characteristics of different search patterns.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runBenchmark,
}

func init() {
	benchCmd.Flags().IntVar(&workers, "workers", 4, "Number of concurrent workers")
	benchCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Benchmark timeout")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	pattern := args[0]

	paths := []string{"."}
	if len(args) > 1 {
		paths = args[1:]
	}

	fmt.Printf("Benchmarking pattern: %q\n", pattern)
	fmt.Printf("Paths: %v\n", paths)
	fmt.Printf("Workers: %d\n", workers)
	fmt.Println()

	// Run benchmark with GoRipGrep
	start := time.Now()

	// Build options for benchmark
	var opts []goripgrep.Option
	opts = append(opts, goripgrep.WithWorkers(workers))
	opts = append(opts, goripgrep.WithGitignore(useGitignore))
	opts = append(opts, goripgrep.WithTimeout(timeout))

	var totalMatches int
	var totalFiles int

	for _, path := range paths {
		results, err := goripgrep.Find(pattern, path, opts...)
		if err != nil {
			return fmt.Errorf("benchmark failed: %w", err)
		}

		totalMatches += len(results.Matches)
		totalFiles += len(results.Files())
	}

	duration := time.Since(start)

	fmt.Printf("GoRipGrep Results:\n")
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Matches: %d\n", totalMatches)
	fmt.Printf("  Files: %d\n", totalFiles)
	fmt.Printf("  Rate: %.2f matches/sec\n", float64(totalMatches)/duration.Seconds())

	return nil
}
