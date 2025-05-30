# GoRipGrep API Documentation

This document provides comprehensive API documentation for the GoRipGrep library.

## Table of Contents

- [Quick Start](#quick-start)
- [Core Types](#core-types)
- [Functional Options API](#functional-options-api)
- [Engine APIs](#engine-apis)
- [Configuration](#configuration)
- [Results and Statistics](#results-and-statistics)
- [Advanced Features](#advanced-features)
- [Error Handling](#error-handling)
- [Performance Considerations](#performance-considerations)

## Quick Start

```go
import "github.com/localrivet/goripgrep"

// Simple search with functional options
results, err := goripgrep.Find("pattern", "/path/to/search")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found %d matches in %d files\n", results.Count(), len(results.Files()))
```

## Core Types

### Match

Represents a single search match with context information.

```go
type Match struct {
    File     string   // File path where match was found
    Line     int      // Line number (1-indexed)
    Column   int      // Column number (1-indexed)
    Content  string   // The matching line content
    Context  []string // Context lines (if requested)
}
```

### SearchResults

Container for search results with metadata and statistics.

```go
type SearchResults struct {
    Matches []Match      // Found matches
    Stats   SearchStats  // Performance statistics
    Query   string       // Search pattern
}

// Methods
func (r *SearchResults) HasMatches() bool    // Check if any matches found
func (r *SearchResults) Count() int          // Get total number of matches
func (r *SearchResults) Files() []string     // Get unique files with matches
```

### SearchStats

Performance and execution statistics for search operations.

```go
type SearchStats struct {
    FilesScanned int64         // Number of files scanned
    FilesSkipped int64         // Number of files skipped
    FilesIgnored int64         // Number of files ignored by gitignore
    BytesScanned int64         // Total bytes scanned
    MatchesFound int64         // Total matches found
    Duration     time.Duration // Search duration
    StartTime    time.Time     // Search start time
    EndTime      time.Time     // Search end time
}
```

## Functional Options API

The primary API uses functional options for flexible configuration.

### Find Function

```go
func Find(pattern, path string, opts ...Option) (*SearchResults, error)
```

Basic usage:
```go
// Simple search
results, err := goripgrep.Find("TODO", "/path/to/project")

// Search with options
results, err := goripgrep.Find("pattern", "/path",
    goripgrep.WithIgnoreCase(),
    goripgrep.WithContextLines(2),
    goripgrep.WithFilePattern("*.go"),
)
```

### Available Options

#### Context and Cancellation
```go
func WithContext(ctx context.Context) Option
```
Set context for cancellation and timeout control.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := goripgrep.Find("pattern", "/path",
    goripgrep.WithContext(ctx),
)
```

#### Performance Options
```go
func WithWorkers(count int) Option           // Number of concurrent workers
func WithBufferSize(size int) Option         // I/O buffer size in bytes
func WithMaxResults(max int) Option          // Maximum results to return
func WithOptimization(enabled bool) Option   // Enable performance optimizations
```

Example:
```go
results, err := goripgrep.Find("pattern", "/path",
    goripgrep.WithWorkers(8),
    goripgrep.WithBufferSize(128*1024), // 128KB buffer
    goripgrep.WithMaxResults(1000),
    goripgrep.WithOptimization(true),
)
```

#### Search Behavior Options
```go
func WithIgnoreCase() Option                 // Case-insensitive search
func WithCaseSensitive() Option              // Case-sensitive search (default)
func WithContextLines(lines int) Option      // Number of context lines
func WithTimeout(duration time.Duration) Option // Search timeout
```

Example:
```go
results, err := goripgrep.Find("ERROR", "/var/log",
    goripgrep.WithIgnoreCase(),
    goripgrep.WithContextLines(3),
    goripgrep.WithTimeout(60*time.Second),
)
```

#### File Filtering Options
```go
func WithFilePattern(pattern string) Option  // File pattern filter
func WithGitignore(enabled bool) Option      // Enable gitignore filtering
func WithHidden() Option                     // Include hidden files
func WithSymlinks() Option                   // Follow symbolic links
```

Example:
```go
results, err := goripgrep.Find("func main", "/project",
    goripgrep.WithFilePattern("*.go"),
    goripgrep.WithGitignore(true),
    goripgrep.WithHidden(),
)
```

## Engine APIs

### Engine (Single File Search)

For searching individual files with optimized performance.

```go
type SearchArgs struct {
    Path          string   // Search path
    Pattern       string   // Search pattern
    FilePattern   *string  // File pattern filter
    IgnoreCase    *bool    // Case sensitivity
    MaxResults    *int     // Result limit
    IncludeHidden *bool    // Include hidden files
    ContextLines  *int     // Context lines
    TimeoutMs     *int     // Timeout in milliseconds
}

func NewEngine(args SearchArgs) (*Engine, error)
func (e *Engine) Search(ctx context.Context, filePath string) ([]Match, error)
func (e *Engine) GetStats() map[string]interface{}
```

Example:
```go
args := goripgrep.SearchArgs{
    Pattern:      "func.*main",
    IgnoreCase:   &[]bool{true}[0],
    ContextLines: &[]int{2}[0],
}

engine, err := goripgrep.NewEngine(args)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
matches, err := engine.Search(ctx, "/path/to/file.go")
if err != nil {
    log.Fatal(err)
}

// Get performance statistics
stats := engine.GetStats()
fmt.Printf("Scanned %d bytes\n", stats["bytes_scanned"])
```

### SearchEngine (Directory Traversal)

For searching across directories with full feature support.

```go
type SearchConfig struct {
    SearchPath      string        // Root path to search
    MaxWorkers      int          // Number of concurrent workers
    BufferSize      int          // I/O buffer size
    MaxResults      int          // Maximum number of results
    UseOptimization bool         // Enable performance optimizations
    UseGitignore    bool         // Enable gitignore filtering
    IgnoreCase      bool         // Case-insensitive search
    IncludeHidden   bool         // Include hidden files
    FollowSymlinks  bool         // Follow symbolic links
    FilePattern     string       // File pattern filter
    ContextLines    int          // Number of context lines
    Timeout         time.Duration // Search timeout
}

func NewSearchEngine(config SearchConfig) *SearchEngine
func (e *SearchEngine) Search(ctx context.Context, pattern string) (*SearchResults, error)
```

Example:
```go
config := goripgrep.SearchConfig{
    SearchPath:      "/path/to/project",
    MaxWorkers:      8,
    BufferSize:      64 * 1024,
    MaxResults:      1000,
    UseOptimization: true,
    UseGitignore:    true,
    IgnoreCase:      true,
    FilePattern:     "*.{go,js,py}",
    ContextLines:    3,
    Timeout:         30 * time.Second,
}

searchEngine := goripgrep.NewSearchEngine(config)
results, err := searchEngine.Search(context.Background(), "pattern")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Search completed in %v\n", results.Stats.Duration)
```

### UnicodeSearchEngine

For advanced Unicode-aware searching.

```go
func NewUnicodeSearchEngine(pattern string, ignoreCase bool) (*UnicodeSearchEngine, error)
func (e *UnicodeSearchEngine) Search(text string) []UnicodeMatch
func (e *UnicodeSearchEngine) SearchFile(filePath string) ([]UnicodeMatch, error)
```

Example:
```go
engine, err := goripgrep.NewUnicodeSearchEngine(`\p{Greek}+`, false)
if err != nil {
    log.Fatal(err)
}

matches := engine.Search("Hello Κόσμος World")
for _, match := range matches {
    fmt.Printf("Found Greek text: %s\n", match.Text)
}
```

### RegexEngine

For advanced regex features with capture groups.

```go
func NewRegex(pattern string, ignoreCase bool) (*RegexEngine, error)
func (e *RegexEngine) FindAll(text string) []RegexMatch
func (e *RegexEngine) Matches(text string) bool
func (e *RegexEngine) ReplaceAll(text, replacement string) string
```

Example:
```go
engine, err := goripgrep.NewRegex(`(\w+)@(\w+\.\w+)`, false)
if err != nil {
    log.Fatal(err)
}

matches := engine.FindAll("Contact: john@example.com or jane@test.org")
for _, match := range matches {
    fmt.Printf("Email: %s, User: %s, Domain: %s\n", 
        match.Text, match.Groups[0], match.Groups[1])
}
```

### GitignoreEngine

For gitignore pattern matching.

```go
func NewGitignoreEngine(basePath string) *GitignoreEngine
func (e *GitignoreEngine) ShouldIgnore(filePath string) bool
func (e *GitignoreEngine) LoadPatterns(patterns []string)
```

Example:
```go
gitignore := goripgrep.NewGitignoreEngine("/project/root")
if gitignore.ShouldIgnore("/project/root/node_modules/file.js") {
    fmt.Println("File should be ignored")
}
```

## Configuration

### Default Values

```go
// Default configuration values
MaxWorkers:      4
BufferSize:      64 * 1024  // 64KB
MaxResults:      1000
UseOptimization: true
UseGitignore:    true
IgnoreCase:      false
IncludeHidden:   false
ContextLines:    0
Timeout:         30 * time.Second
```

### File Pattern Syntax

File patterns support glob-style matching:
- `*.go` - All Go files
- `*.{go,js,py}` - Go, JavaScript, and Python files
- `**/*.test.go` - Test files in any subdirectory
- `src/**/*` - All files under src directory

### Gitignore Support

Full .gitignore specification support:
- Standard patterns: `*.log`, `build/`
- Negation: `!important.log`
- Directory-only: `cache/`
- Absolute paths: `/root-only`
- Wildcards: `**/*.tmp`

## Results and Statistics

### Processing Results

```go
results, err := goripgrep.Find("pattern", "/path")
if err != nil {
    log.Fatal(err)
}

// Check if matches found
if !results.HasMatches() {
    fmt.Println("No matches found")
    return
}

// Process all matches
for _, match := range results.Matches {
    fmt.Printf("%s:%d:%d: %s\n", 
        match.File, match.Line, match.Column, match.Content)
    
    // Print context lines
    for _, context := range match.Context {
        fmt.Printf("  | %s\n", context)
    }
}

// Get unique files
files := results.Files()
fmt.Printf("Found matches in %d files\n", len(files))
```

### Performance Statistics

```go
stats := results.Stats
fmt.Printf("Performance Statistics:\n")
fmt.Printf("  Files scanned: %d\n", stats.FilesScanned)
fmt.Printf("  Files skipped: %d\n", stats.FilesSkipped)
fmt.Printf("  Files ignored: %d\n", stats.FilesIgnored)
fmt.Printf("  Bytes scanned: %d\n", stats.BytesScanned)
fmt.Printf("  Matches found: %d\n", stats.MatchesFound)
fmt.Printf("  Duration: %v\n", stats.Duration)
fmt.Printf("  Throughput: %.2f MB/s\n", 
    float64(stats.BytesScanned)/1024/1024/stats.Duration.Seconds())
```

### Engine Statistics

```go
engine, _ := goripgrep.NewEngine(args)
// ... perform searches ...

stats := engine.GetStats()
fmt.Printf("Engine Statistics:\n")
fmt.Printf("  Bytes scanned: %d\n", stats["bytes_scanned"])
fmt.Printf("  Files scanned: %d\n", stats["files_scanned"])
fmt.Printf("  Matches found: %d\n", stats["matches_found"])
fmt.Printf("  Is literal: %v\n", stats["is_literal"])
fmt.Printf("  Rare byte: 0x%02x\n", stats["rare_byte"])
fmt.Printf("  Worker count: %d\n", stats["worker_count"])
fmt.Printf("  Buffer size: %d\n", stats["buffer_size"])

// SIMD capabilities
fmt.Printf("  Pure Go optimization: %v\n", stats["simd_pure_go"])
fmt.Printf("  Word optimization: %v\n", stats["simd_word_optimized"])

// Cache statistics
fmt.Printf("  Cache size: %d\n", stats["cache_size"])
fmt.Printf("  Cache hits: %d\n", stats["cache_hits"])
fmt.Printf("  Cache misses: %d\n", stats["cache_misses"])
fmt.Printf("  Cache hit rate: %.2f%%\n", stats["cache_hit_rate"].(float64)*100)
```

## Advanced Features

### Compressed File Search

GoRipGrep automatically detects and searches compressed files:

```go
// Automatically handles .gz and .bz2 files
results, err := goripgrep.Find("error", "/var/log",
    goripgrep.WithFilePattern("*.{log,gz,bz2}"),
)
```

Supported compression formats:
- **gzip** (.gz, .gzip) - Using Go's `compress/gzip`
- **bzip2** (.bz2, .bzip2) - Using Go's `compress/bzip2`

### Unicode Character Classes

```go
// Search for Greek characters
results, err := goripgrep.Find(`\p{Greek}+`, "/documents")

// Search for any letter in any script
results, err := goripgrep.Find(`\p{L}+`, "/text")

// Supported character classes:
// \p{Greek}, \p{Latin}, \p{Cyrillic}, \p{Arabic}, \p{Hebrew}
// \p{Han}, \p{Hiragana}, \p{Katakana}, \p{Thai}, \p{Devanagari}
```

### Context Lines

```go
// Get 3 lines of context around each match
results, err := goripgrep.Find("error", "/logs",
    goripgrep.WithContextLines(3),
)

for _, match := range results.Matches {
    fmt.Printf("Match: %s\n", match.Content)
    fmt.Println("Context:")
    for _, line := range match.Context {
        fmt.Printf("  %s\n", line)
    }
}
```

### Performance Optimization

```go
// Enable all optimizations
results, err := goripgrep.Find("pattern", "/path",
    goripgrep.WithOptimization(true),
    goripgrep.WithWorkers(runtime.NumCPU()),
    goripgrep.WithBufferSize(128*1024),
)

// Check optimization status
engine, _ := goripgrep.NewEngine(args)
stats := engine.GetStats()
if stats["simd_pure_go"].(bool) {
    fmt.Println("Pure Go optimizations enabled")
}
```

## Error Handling

### Common Error Types

```go
results, err := goripgrep.Find("pattern", "/path")
if err != nil {
    switch {
    case os.IsNotExist(err):
        fmt.Println("Path does not exist")
    case os.IsPermission(err):
        fmt.Println("Permission denied")
    case strings.Contains(err.Error(), "invalid regex"):
        fmt.Println("Invalid regex pattern")
    case err == context.DeadlineExceeded:
        fmt.Println("Search timed out")
    case err == context.Canceled:
        fmt.Println("Search was canceled")
    default:
        fmt.Printf("Search error: %v\n", err)
    }
}
```

### Timeout Handling

```go
// Set timeout via context
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := goripgrep.Find("pattern", "/large/directory",
    goripgrep.WithContext(ctx),
)
if err == context.DeadlineExceeded {
    fmt.Println("Search timed out after 30 seconds")
}

// Or use timeout option
results, err = goripgrep.Find("pattern", "/large/directory",
    goripgrep.WithTimeout(30*time.Second),
)
```

### Graceful Degradation

```go
// Search continues even if some files can't be read
results, err := goripgrep.Find("pattern", "/mixed/permissions")
if err != nil {
    log.Printf("Search completed with errors: %v", err)
}

// Check statistics for skipped files
fmt.Printf("Files skipped due to errors: %d\n", results.Stats.FilesSkipped)
```

## Performance Considerations

### Optimization Tips

1. **Use literal patterns when possible** - They're much faster than regex
2. **Set appropriate buffer sizes** - Larger buffers for large files
3. **Limit results** - Use `WithMaxResults()` for better performance
4. **Use file patterns** - Avoid scanning unnecessary files
5. **Enable gitignore** - Skip irrelevant files automatically

### Memory Usage

```go
// For large directories, limit memory usage
results, err := goripgrep.Find("pattern", "/huge/directory",
    goripgrep.WithMaxResults(1000),        // Limit results
    goripgrep.WithBufferSize(32*1024),     // Smaller buffer
    goripgrep.WithWorkers(2),              // Fewer workers
)
```

### Benchmarking

```go
// Use the performance benchmarking example
// go run examples/performance-benchmarking/main.go

// Or create custom benchmarks
start := time.Now()
results, err := goripgrep.Find("pattern", "/path")
duration := time.Since(start)

if err == nil {
    throughput := float64(results.Stats.BytesScanned) / 1024 / 1024 / duration.Seconds()
    fmt.Printf("Throughput: %.2f MB/s\n", throughput)
}
```

### Best Practices

1. **Reuse engines** for multiple searches with the same pattern
2. **Use appropriate timeouts** for long-running searches
3. **Monitor statistics** to identify performance bottlenecks
4. **Test with representative data** to optimize configuration
5. **Profile memory usage** for large-scale applications

---

For more examples and advanced usage patterns, see the [examples directory](../examples/) and [Tutorial](TUTORIAL.md). 