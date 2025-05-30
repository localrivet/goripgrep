# GoRipGrep Tutorial

This tutorial will guide you through using GoRipGrep, from basic searches to advanced configurations.

## Table of Contents

1. [Installation](#installation)
2. [Your First Search](#your-first-search)
3. [Understanding Results](#understanding-results)
4. [Case-Insensitive Search](#case-insensitive-search)
5. [Context Lines](#context-lines)
6. [File Filtering](#file-filtering)
7. [Functional Options API](#functional-options-api)
8. [Regular Expressions](#regular-expressions)
9. [Performance Optimization](#performance-optimization)
10. [Error Handling](#error-handling)
11. [Advanced Features](#advanced-features)

## Installation

Add GoRipGrep to your Go project:

```bash
go get github.com/localrivet/goripgrep
```

Import it in your Go code:

```go
import "github.com/localrivet/goripgrep"
```

## Your First Search

Let's start with the simplest possible search:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/localrivet/goripgrep"
)

func main() {
    // Search for "func" in the current directory
    results, err := goripgrep.Find("func", ".")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d matches\n", results.Count())
}
```

This searches for the literal string "func" in all files in the current directory and subdirectories.

## Understanding Results

The `SearchResults` type contains all the information about your search:

```go
results, err := goripgrep.Find("package", ".")
if err != nil {
    log.Fatal(err)
}

// Basic information
fmt.Printf("Found %d matches\n", results.Count())
fmt.Printf("Has matches: %v\n", results.HasMatches())
fmt.Printf("Files with matches: %v\n", results.Files())

// Iterate through matches
for _, match := range results.Matches {
    fmt.Printf("%s:%d:%d: %s\n", 
        match.File,    // File path
        match.Line,    // Line number (1-indexed)
        match.Column,  // Column number (1-indexed)
        match.Content) // The matching line
}

// Performance statistics
stats := results.Stats
fmt.Printf("Search took: %v\n", stats.Duration)
fmt.Printf("Files scanned: %d\n", stats.FilesScanned)
fmt.Printf("Bytes scanned: %d\n", stats.BytesScanned)
```

## Case-Insensitive Search

Use the `WithIgnoreCase()` option for case-insensitive searches:

```go
// Find "TODO" in any case: todo, Todo, TODO, etc.
results, err := goripgrep.Find("todo", ".", goripgrep.WithIgnoreCase())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d TODO items\n", results.Count())
for _, match := range results.Matches {
    fmt.Printf("%s:%d: %s\n", match.File, match.Line, match.Content)
}
```

## Context Lines

Context lines show you the surrounding code for better understanding:

```go
// Get 2 lines before and after each match
results, err := goripgrep.Find("error", ".", goripgrep.WithContextLines(2))
if err != nil {
    log.Fatal(err)
}

for _, match := range results.Matches {
    fmt.Printf("\n%s:%d: %s\n", match.File, match.Line, match.Content)
    
    // Print context lines
    for _, contextLine := range match.Context {
        fmt.Printf("  | %s\n", contextLine)
    }
}
```

## File Filtering

Search only specific file types using the `WithFilePattern()` option:

```go
// Search only in Go files
results, err := goripgrep.Find("func", ".", goripgrep.WithFilePattern("*.go"))
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d functions in Go files\n", results.Count())

// Search in multiple file types
results, err = goripgrep.Find("TODO", ".", 
    goripgrep.WithFilePattern("*.{go,js,py}"))
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d TODOs in source files\n", results.Count())
```

## Functional Options API

The functional options API provides maximum flexibility by combining multiple options:

```go
// Build a complex search configuration
results, err := goripgrep.Find("TODO", "./src",
    goripgrep.WithIgnoreCase(),              // Case-insensitive
    goripgrep.WithContextLines(1),           // 1 line of context
    goripgrep.WithFilePattern("*.{go,js,py}"), // Multiple file types
    goripgrep.WithGitignore(true),           // Respect .gitignore
    goripgrep.WithMaxResults(100),           // Limit results
    goripgrep.WithWorkers(4),                // Use 4 worker threads
    goripgrep.WithHidden(),                  // Include hidden files
    goripgrep.WithTimeout(30*time.Second),   // Set timeout
)
if err != nil {
    log.Fatal(err)
}

// Process results
for _, match := range results.Matches {
    fmt.Printf("%s:%d: %s\n", match.File, match.Line, match.Content)
    for _, ctx := range match.Context {
        fmt.Printf("  | %s\n", ctx)
    }
}
```

## Regular Expressions

GoRipGrep supports full Go regex syntax:

```go
// Find function definitions
results, err := goripgrep.Find(`func\s+\w+\(`, ".")
if err != nil {
    log.Fatal(err)
}

// Find email addresses with case-insensitive search
results, err = goripgrep.Find(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, ".",
    goripgrep.WithIgnoreCase())
if err != nil {
    log.Fatal(err)
}

// Find TODO comments with context
results, err = goripgrep.Find(`//\s*TODO:.*`, ".",
    goripgrep.WithContextLines(2),
    goripgrep.WithFilePattern("*.go"))
if err != nil {
    log.Fatal(err)
}
```

## Performance Optimization

GoRipGrep includes several performance optimization options:

```go
// High-performance search configuration
results, err := goripgrep.Find("pattern", "/large/directory",
    goripgrep.WithWorkers(8),                // More workers for large datasets
    goripgrep.WithBufferSize(128*1024),      // Larger buffer for I/O
    goripgrep.WithOptimization(true),        // Enable all optimizations
    goripgrep.WithMaxResults(1000),          // Limit results to avoid memory issues
    goripgrep.WithTimeout(60*time.Second),   // Prevent long-running searches
)
if err != nil {
    log.Fatal(err)
}

// Check performance statistics
stats := results.Stats
fmt.Printf("Performance Report:\n")
fmt.Printf("  Duration: %v\n", stats.Duration)
fmt.Printf("  Files scanned: %d\n", stats.FilesScanned)
fmt.Printf("  Bytes scanned: %d\n", stats.BytesScanned)
fmt.Printf("  Throughput: %.2f MB/s\n", 
    float64(stats.BytesScanned)/1024/1024/stats.Duration.Seconds())
```

## Error Handling

GoRipGrep provides comprehensive error handling:

```go
results, err := goripgrep.Find("pattern", "/path/to/search")
if err != nil {
    // Handle different types of errors
    switch {
    case strings.Contains(err.Error(), "no such file"):
        fmt.Println("Search path does not exist")
    case strings.Contains(err.Error(), "invalid regex"):
        fmt.Println("Invalid regular expression pattern")
    case strings.Contains(err.Error(), "context"):
        fmt.Println("Search was cancelled or timed out")
    default:
        fmt.Printf("Search error: %v\n", err)
    }
    return
}

// Check if any matches were found
if !results.HasMatches() {
    fmt.Println("No matches found")
    return
}

// Process results...
```

## Advanced Features

### Context Cancellation

```go
// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := goripgrep.Find("pattern", "/large/directory",
    goripgrep.WithContext(ctx))
if err != nil {
    if err == context.DeadlineExceeded {
        fmt.Println("Search timed out")
    }
    return
}
```

### Compressed File Search

```go
// Search in compressed files (gzip, bzip2)
results, err := goripgrep.Find("error", "/var/log",
    goripgrep.WithFilePattern("*.{log,gz,bz2}"))
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d errors in log files\n", results.Count())
```

### Unicode Support

```go
// Search for Unicode text
results, err := goripgrep.Find("世界", ".",
    goripgrep.WithIgnoreCase())
if err != nil {
    log.Fatal(err)
}

// Search with Unicode character classes (requires regex)
results, err = goripgrep.Find(`\p{Greek}+`, ".",
    goripgrep.WithIgnoreCase())
if err != nil {
    log.Fatal(err)
}
```

### Direct Engine Usage

For more control, you can use the engines directly:

```go
// Create a search engine with specific configuration
config := goripgrep.SearchConfig{
    SearchPath:      "/path/to/search",
    MaxWorkers:      4,
    BufferSize:      64 * 1024,
    MaxResults:      1000,
    UseOptimization: true,
    UseGitignore:    true,
    IgnoreCase:      true,
    FilePattern:     "*.go",
    ContextLines:    2,
}

engine := goripgrep.NewSearchEngine(config)
results, err := engine.Search(context.Background(), "pattern")
if err != nil {
    log.Fatal(err)
}

// Get detailed performance report
report := engine.GetPerformanceReport()
fmt.Printf("Search completed in %v\n", report.Stats.Duration)
```

### Engine-Specific Features

```go
// Use the regex engine for advanced patterns
regexEngine, err := goripgrep.NewRegex(`func\s+(\w+)\(`, false)
if err != nil {
    log.Fatal(err)
}

matches := regexEngine.FindAllMatches("func main() { ... }")
for _, match := range matches {
    fmt.Printf("Function: %s\n", match.Groups[1]) // Capture group
}

// Use Unicode engine for international text
unicodeEngine, err := goripgrep.NewUnicodeSearchEngine("café", true)
if err != nil {
    log.Fatal(err)
}

// Search with Unicode normalization and case folding
matches, err = unicodeEngine.Search("Café CAFÉ café")
if err != nil {
    log.Fatal(err)
}
```

## Best Practices

1. **Use appropriate options**: Don't enable features you don't need
2. **Set reasonable timeouts**: Prevent runaway searches
3. **Limit results**: Use `WithMaxResults()` for large datasets
4. **Choose the right worker count**: 4-8 workers for most cases
5. **Handle errors gracefully**: Check for context cancellation and timeouts
6. **Use file patterns**: Filter files early to improve performance
7. **Monitor performance**: Check `SearchStats` for optimization opportunities

This tutorial covers the essential features of GoRipGrep. For more advanced usage and API details, see the [API Documentation](API.md).

## Next Steps

- Explore the [API Documentation](API.md) for complete reference
- Check out the [examples/](../examples/) directory for more use cases
- Read the [Performance Analysis](PERFORMANCE_ANALYSIS.md) for optimization tips

## Getting Help

If you encounter issues or have questions:

1. Check the [API Documentation](API.md)
2. Look at the examples in the `examples/` directory
3. Review the test files for usage patterns
4. Open an issue on the project repository 