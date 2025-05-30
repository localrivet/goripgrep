# GoRipGrep

A high-performance, pure Go text search library that provides ripgrep-like functionality without external dependencies. GoRipGrep combines the speed of optimized algorithms with the convenience of native Go integration.

## Features

### Core Capabilities
- **Fast literal string search** with byte-level optimizations and rare byte scanning
- **Advanced regex support** with DFA caching and Unicode character classes
- **Full Unicode support** including emojis, multi-byte characters, and encoding detection
- **Gitignore pattern matching** with full .gitignore specification support
- **Concurrent processing** with configurable worker pools
- **Memory-efficient scanning** with buffered I/O and streaming
- **Performance metrics** and comprehensive benchmarking tools
- **Compressed file search** with on-the-fly decompression (gzip, bzip2)

### Advanced Features
- **Pure Go optimizations** using unsafe pointer arithmetic for 8-byte word scanning
- **Rare byte optimization** for multi-byte pattern matching
- **Unicode character class support** (Greek, Latin, Cyrillic, Arabic, Hebrew, Han, etc.)
- **Advanced regex features** with named capture groups and backreferences
- **Binary file detection** and filtering
- **Context lines** support with configurable line counts
- **Timeout handling** for long-running searches
- **DFA cache** for regex compilation optimization
- **Streaming decompression** for large compressed files

## Requirements

- **Go 1.21+**
- No external dependencies (pure Go implementation)

## Installation

```bash
go get github.com/localrivet/goripgrep
```

## Quick Start

### Simple Functional API

```go
package main

import (
    "fmt"
    "log"
    "github.com/localrivet/goripgrep"
)

func main() {
    // Basic search (non-recursive by default)
    results, err := goripgrep.Find("hello", ".")
    if err != nil {
        log.Fatal(err)
    }
    
    // Recursive search with options
    results, err = goripgrep.Find("hello", ".", 
        goripgrep.WithRecursive(true),
        goripgrep.WithIgnoreCase(),
        goripgrep.WithContextLines(2))
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d matches in %d files\n", 
        results.Count(), len(results.Files()))
}
```

### Advanced Search with Options

```go
// Search with multiple options
results, err := goripgrep.Find("TODO", "/path/to/project",
    goripgrep.WithIgnoreCase(),
    goripgrep.WithContextLines(2),
    goripgrep.WithFilePattern("*.go"),
    goripgrep.WithMaxResults(100),
    goripgrep.WithGitignore(true),
    goripgrep.WithTimeout(30*time.Second),
)
if err != nil {
    log.Fatal(err)
}

// Process results with context
for _, match := range results.Matches {
    fmt.Printf("%s:%d:%d: %s\n", match.File, match.Line, match.Column, match.Content)
    
    // Print context lines if available
    for _, contextLine := range match.Context {
        fmt.Printf("  | %s\n", contextLine)
    }
}
```

### Direct Engine Usage

```go
// Create engine with specific configuration
args := goripgrep.SearchArgs{
    Pattern:      "func.*main",
    IgnoreCase:   &[]bool{true}[0],
    ContextLines: &[]int{2}[0],
}

engine, err := goripgrep.NewEngine(args)
if err != nil {
    log.Fatal(err)
}

// Search a specific file
ctx := context.Background()
matches, err := engine.Search(ctx, "/path/to/file.go")
if err != nil {
    log.Fatal(err)
}

// Get performance statistics
stats := engine.GetStats()
fmt.Printf("Scanned %d bytes in %d files\n", 
    stats["bytes_scanned"], stats["files_scanned"])
```

### SearchEngine for Directory Traversal

```go
// Configure search engine for directory traversal
config := goripgrep.SearchConfig{
    SearchPath:      "/path/to/project",
    MaxWorkers:      8,
    BufferSize:      64 * 1024,
    MaxResults:      1000,
    UseOptimization: true,
    UseGitignore:    true,
    IgnoreCase:      true,
    IncludeHidden:   false,
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
fmt.Printf("Files scanned: %d, Matches found: %d\n", 
    results.Stats.FilesScanned, results.Stats.MatchesFound)
```

## Architecture

GoRipGrep uses a modular architecture with specialized engines:

### Core Engines

1. **Engine** (`engine.go`)
   - Fast literal string search with rare byte optimization
   - Pure Go word-level scanning (8-byte operations)
   - Regex optimization with literal extraction
   - Compressed file support with streaming decompression

2. **UnicodeSearchEngine** (`unicode.go`)
   - Full Unicode character class support
   - Unicode-aware case folding and normalization
   - Encoding detection (UTF-8, UTF-16, Latin-1, etc.)
   - Rune-based position tracking

3. **RegexEngine** (`regex.go`)
   - Advanced regex features with capture groups
   - Named capture groups and backreferences
   - Unicode property classes (\p{Greek}, \p{Latin}, etc.)
   - DFA caching for performance

4. **GitignoreEngine** (`gitignore.go`)
   - Full .gitignore specification support
   - Negation patterns (!) and directory patterns (/)
   - Wildcard expansion (*, **, ?, [abc])
   - Hierarchical pattern matching

5. **SearchEngine** (`search.go`)
   - Combines all engines with intelligent selection
   - Directory traversal with worker pools
   - Performance monitoring and metrics
   - Configurable feature flags

6. **OptimizedEngine** (`optimized_search.go`)
   - Pure Go performance optimizations
   - Word-level byte scanning
   - CPU feature detection
   - Benchmarking and performance analysis

## Performance

### Benchmarks
- **2-16x faster** than Go's standard regex for literal patterns
- **Sub-millisecond** search times for typical patterns
- **Memory-efficient** with 64KB buffers and streaming
- **Concurrent processing** scaling with CPU cores

### Memory Usage
- Streaming processing to minimize memory footprint
- Garbage collection friendly design
- Configurable buffer sizes for different workloads
- Compressed file support without full decompression

### Optimization Features
- **Rare byte scanning** for multi-byte patterns
- **DFA caching** for regex compilation
- **Pure Go word-level operations** (no CGO dependencies)
- **CPU feature detection** for optimal performance

## Testing

The library includes comprehensive tests covering:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. -benchmem ./...

# Run performance benchmarks
go run examples/performance-benchmarking/main.go
```

### Test Coverage
- **100% test pass rate** with comprehensive functionality coverage
- Unicode test cases with emojis and special characters
- Performance benchmarking with statistical analysis
- Edge case handling (empty files, binary detection, etc.)
- Compressed file search testing

## Configuration

### Functional Options API

```go
// Available options for Find function
results, err := goripgrep.Find("pattern", "/path",
    goripgrep.WithContext(ctx),                    // Set context for cancellation
    goripgrep.WithWorkers(8),                      // Number of concurrent workers
    goripgrep.WithBufferSize(64*1024),            // I/O buffer size
    goripgrep.WithMaxResults(1000),               // Maximum results to return
    goripgrep.WithOptimization(true),             // Enable performance optimizations
    goripgrep.WithGitignore(true),                // Enable gitignore filtering
    goripgrep.WithIgnoreCase(),                   // Case-insensitive search
    goripgrep.WithCaseSensitive(),                // Case-sensitive search (default)
    goripgrep.WithHidden(),                       // Include hidden files
    goripgrep.WithSymlinks(),                     // Follow symbolic links
    goripgrep.WithRecursive(true),                // Search directories recursively (default: false)
    goripgrep.WithFilePattern("*.go"),            // File pattern filter
    goripgrep.WithContextLines(3),                // Number of context lines
    goripgrep.WithTimeout(30*time.Second),        // Search timeout
)
```

### SearchConfig Structure

```go
type SearchConfig struct {
    SearchPath      string        // Root path to search
    MaxWorkers      int          // Number of concurrent workers
    BufferSize      int          // I/O buffer size (default: 64KB)
    MaxResults      int          // Maximum number of results
    UseOptimization bool         // Enable performance optimizations
    UseGitignore    bool         // Enable gitignore filtering
    IgnoreCase      bool         // Case-insensitive search
    IncludeHidden   bool         // Include hidden files
    FollowSymlinks  bool         // Follow symbolic links
    Recursive       bool         // Search directories recursively (default: false)
    FilePattern     string       // File pattern filter
    ContextLines    int          // Number of context lines
    Timeout         time.Duration // Search timeout
}
```

### SearchArgs for Engine

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
```

## Use Cases

GoRipGrep is ideal for:

- **Code search tools** and IDEs
- **Log analysis** and monitoring systems
- **Documentation search** with Unicode support
- **Build systems** requiring gitignore compliance
- **Text processing pipelines** with performance requirements
- **CLI tools** needing fast search capabilities
- **Compressed file analysis** without extraction

## API Reference

### Core Functions

```go
// Simple search with functional options
func Find(pattern, path string, opts ...Option) (*SearchResults, error)

// Available options
func WithContext(ctx context.Context) Option
func WithWorkers(count int) Option
func WithBufferSize(size int) Option
func WithMaxResults(max int) Option
func WithOptimization(enabled bool) Option
func WithGitignore(enabled bool) Option
func WithIgnoreCase() Option
func WithCaseSensitive() Option
func WithHidden() Option
func WithSymlinks() Option
func WithRecursive(recursive bool) Option
func WithFilePattern(pattern string) Option
func WithContextLines(lines int) Option
func WithTimeout(duration time.Duration) Option
```

### Engine Creation

```go
// Create optimized search engine
func NewEngine(args SearchArgs) (*Engine, error)

// Create search engine for directory traversal
func NewSearchEngine(config SearchConfig) *SearchEngine

// Create Unicode-aware engine
func NewUnicodeSearchEngine(pattern string, ignoreCase bool) (*UnicodeSearchEngine, error)

// Create regex engine
func NewRegex(pattern string, ignoreCase bool) (*RegexEngine, error)

// Create gitignore engine
func NewGitignoreEngine(basePath string) *GitignoreEngine
```

### Result Types

```go
type SearchResults struct {
    Matches []Match      // Found matches
    Stats   SearchStats  // Performance statistics
    Query   string       // Search pattern
}

type Match struct {
    File    string   // Path to the file containing the match
    Line    int      // Line number (1-indexed)
    Column  int      // Column number (1-indexed)
    Content string   // Content of the matching line
    Context []string // Context lines (if requested)
}

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

### Result Methods

```go
// Check if any matches were found
func (r *SearchResults) HasMatches() bool

// Get total number of matches
func (r *SearchResults) Count() int

// Get unique files containing matches
func (r *SearchResults) Files() []string
```

### Supported Features

### ✅ Fully Implemented
- Fast literal string search with rare byte optimization
- Advanced regex pattern matching with DFA caching
- Case-insensitive search with Unicode support
- File pattern filtering with glob patterns
- Binary file detection and filtering
- Hidden file handling
- Gitignore support with full specification
- Unicode support with character classes
- Context lines with configurable counts
- Concurrent processing with worker pools
- Memory-efficient scanning with streaming
- Performance optimization with pure Go techniques
- Compressed file search (gzip, bzip2)
- Timeout support with context cancellation
- Result limiting and pagination
- Performance metrics and statistics
- Comprehensive benchmarking tools

### 🚧 Planned Features
- Streaming search for very large files (Task #28)
- Plugin architecture for custom filters (Task #29)
- Advanced caching mechanisms (Task #30)
- Distributed search capabilities (Task #31)

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- **[Tutorial](docs/TUTORIAL.md)** - Step-by-step guide from basic to advanced usage
- **[API Reference](docs/API.md)** - Complete API documentation with examples
- **[Performance Analysis](docs/PERFORMANCE_ANALYSIS.md)** - Detailed performance benchmarks and optimization guide

### Examples

The `examples/` directory contains practical usage examples:

- **[Simple Usage](examples/simple-usage/)** - Basic search operations
- **[Fluent API](examples/fluent-api/)** - Advanced functional options usage
- **[Unicode Search](examples/unicode-search/)** - International text search
- **[Regex Patterns](examples/regex-patterns/)** - Complex regex examples
- **[Gitignore Filtering](examples/gitignore-filtering/)** - File filtering examples
- **[Performance Benchmarking](examples/performance-benchmarking/)** - Performance analysis tools

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/localrivet/goripgrep.git
cd goripgrep

# Install development dependencies
go mod download

# Run tests and linting
go test ./...
go vet ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by [ripgrep](https://github.com/BurntSushi/ripgrep) by Andrew Gallant
- Built with Go's excellent standard library
- Performance optimizations inspired by modern text search algorithms

---

**GoRipGrep**: Fast, reliable, pure Go text search. No external dependencies, maximum performance.