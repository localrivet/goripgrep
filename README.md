# GoRipGrep

A text search library written in pure Go, inspired by ripgrep. This project is a learning exercise in implementing text search algorithms and optimizations in Go.

**⚠️ Performance Notice**: This implementation is currently **~46x slower** than ripgrep on typical workloads. It's a work-in-progress educational project, not a production replacement for ripgrep.

## Current Performance Status

### Honest Benchmarks (Pattern: `\w+Sushi`, local directory)

```
ripgrep:     20ms (0.020s)
goripgrep:   922ms (0.922s) 
Performance gap: 46x slower
```

**What this means:**
- ✅ Functionally correct (finds same matches as ripgrep)
- ❌ Not competitive with ripgrep for production use
- ✅ Educational value for learning Go optimization techniques
- ❌ Performance regressions occurred during development

### Optimization Attempts

We attempted several optimizations but failed to achieve significant improvements:
- **Memory-mapped file reading**: Implemented but limited impact
- **Basic literal string optimization**: Added to engine  
- **Configuration-based optimization**: Added option flags
- **Result**: Still significantly slower than ripgrep, no meaningful performance gains achieved

## Features

### ✅ Working Features
- **Literal string search** with basic optimizations
- **Regex pattern matching** using Go's standard regexp
- **Directory traversal** with file filtering
- **Gitignore support** for basic patterns
- **Binary file detection** and skipping
- **Context lines** around matches
- **Concurrent processing** with worker pools
- **Basic Unicode support** (via Go's standard library)
- **Functional API** with options

### ❌ Claims We're NOT Making
- ~~"2-16x faster than Go's standard regex"~~ (No evidence for this)
- ~~"Sub-millisecond search times"~~ (False for realistic workloads)
- ~~"DFA caching"~~ (No actual DFA implementation)
- ~~"Pure Go word-level operations"~~ (Just using standard Go)
- ~~"CPU feature detection"~~ (Not implemented)

### 🚧 Areas for Improvement
- **Performance**: Currently 46x slower than ripgrep (significant gap)
- **Memory efficiency**: High allocation count vs ripgrep
- **Advanced regex features**: Basic implementation only
- **SIMD optimizations**: Not implemented
- **True DFA compilation**: Not implemented
- **File I/O optimization**: Standard library approaches vs custom optimizations

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

## Architecture

GoRipGrep uses a straightforward architecture with these components:

### Core Components

1. **Engine** (`engine.go`)
   - Basic pattern search using Go's regexp package
   - Simple literal string optimization
   - File reading with basic buffering

2. **Search** (`search.go`)
   - Directory traversal with `filepath.WalkDir`
   - Binary file detection and skipping
   - Worker pool for concurrent processing
   - Basic gitignore support

3. **API** (`api.go`)
   - Functional options pattern
   - Simple configuration management
   - Result aggregation

4. **Types** (`types.go`)
   - Data structures for results and configuration
   - Basic statistics tracking

## Performance Analysis

### Current Benchmark Results

```bash
# Run real-world performance comparison:
time rg '\w+Sushi' .    # ~20ms
time ./goripgrep '\w+Sushi' .    # ~922ms (46x slower)

# Benchmark tests (if available):
go test -bench=BenchmarkSimpleComparison -benchmem
```

**Key metrics from testing:**
- goripgrep: 922ms (0.922 seconds)
- ripgrep: 20ms (0.020 seconds)  
- **Performance gap: 46x slower**

### Why Is It So Much Slower?

**Honest assessment of performance gaps:**

1. **No SIMD optimizations** - ripgrep uses assembly-optimized string search
2. **Basic regex engine** - using Go's standard regexp vs ripgrep's optimized DFA  
3. **Excessive allocations** - thousands of allocations vs ripgrep's minimal allocation
4. **No advanced byte scanning** - no memchr-style optimizations
5. **Inefficient file I/O** - standard library approaches vs ripgrep's optimized I/O
6. **Suboptimal directory walking** - basic implementation vs ripgrep's optimized walker
7. **No meaningful optimizations** - attempted optimizations provided negligible benefits

### Optimization Opportunities

**Areas where significant improvements could be made:**

1. **Reduce allocations** - currently creating 50K+ objects per search
2. **Implement boyer-moore or similar** for literal string search
3. **Add binary search optimizations** for sorted pattern lists  
4. **Optimize file reading** with better buffering strategies
5. **Implement actual DFA compilation** instead of using standard regexp

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks (see honest performance comparison)
go test -bench=. -benchmem ./...

# Compare directly with ripgrep
time rg '\w+Sushi' .
time ./goripgrep '\w+Sushi' .
```

## Configuration

### Functional Options API

```go
// Available options for Find function
results, err := goripgrep.Find("pattern", "/path",
    goripgrep.WithContext(ctx),                    // Set context for cancellation
    goripgrep.WithWorkers(8),                      // Number of concurrent workers
    goripgrep.WithBufferSize(64*1024),            // I/O buffer size
    goripgrep.WithMaxResults(1000),               // Maximum results to return
    goripgrep.WithOptimization(true),             // Enable basic optimizations
    goripgrep.WithGitignore(true),                // Enable gitignore filtering
    goripgrep.WithIgnoreCase(),                   // Case-insensitive search
    goripgrep.WithRecursive(true),                // Search directories recursively
    goripgrep.WithFilePattern("*.go"),            // File pattern filter
    goripgrep.WithContextLines(3),                // Number of context lines
    goripgrep.WithTimeout(30*time.Second),        // Search timeout
)
```

## Result Types

```go
type SearchResults struct {
    Matches []Match      // Found matches
    Stats   SearchStats  // Basic performance statistics
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
    BytesScanned int64         // Total bytes scanned
    MatchesFound int64         // Total matches found
    Duration     time.Duration // Search duration
}
```

## Use Cases

**Realistic use cases where this might be appropriate:**

- **Learning Go** text processing techniques
- **Educational projects** for understanding search algorithms
- **Small codebases** where 869ms search time is acceptable
- **Integration scenarios** where pure Go is required
- **Prototyping** before switching to production tools

**Use cases where you should use ripgrep instead:**

- **Production applications** requiring fast search
- **Large codebases** (>1000 files)
- **Interactive tools** where speed matters
- **Any performance-critical scenario**

## Supported Features

### ✅ Currently Working
- Basic literal string search
- Regular expression patterns (via Go regexp)
- Case-insensitive search
- File pattern filtering (basic globs)
- Binary file detection
- Hidden file inclusion/exclusion
- Basic gitignore support
- Context lines
- Concurrent processing
- Unicode support (via Go standard library)
- Timeout support
- Result limiting

### ❌ Not Implemented (Despite Earlier Claims)
- DFA caching (just uses standard Go regexp)
- Advanced byte-level optimizations
- SIMD instructions
- CPU feature detection
- Advanced Unicode character classes
- Streaming decompression
- Word-level scanning optimizations
- Performance competitive with ripgrep

### 🚧 Could Be Improved
- Memory efficiency (too many allocations)
- Search speed (fundamental algorithm improvements needed)
- Regex performance (would need custom engine)
- File walking optimization
- Better binary detection

## Contributing

This is a learning project. Contributions are welcome, especially:

- **Performance improvements** with measurable benchmarks
- **Algorithm optimizations** with before/after comparisons
- **Memory allocation reductions**
- **Bug fixes** with test cases
- **Documentation improvements**

Please include benchmark results with any performance-related PRs.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/localrivet/goripgrep.git
cd goripgrep

# Install dependencies
go mod download

# Run tests and benchmarks
go test ./...
go test -bench=. -benchmem ./...

# Compare with ripgrep
time rg '\w+Sushi' .
time ./goripgrep '\w+Sushi' .
```

## Honest Performance Comparison

**If you need fast text search, use ripgrep.** This project is:

✅ **Good for**: Learning, education, Go integration, small projects  
❌ **Bad for**: Production use, large codebases, performance-critical applications

**Performance Reality:**
- ripgrep: 21ms (optimized Rust + assembly)
- goripgrep: 869ms (educational Go implementation)  
- grep: 2.1s (basic system utility)

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Inspired by [ripgrep](https://github.com/BurntSushi/ripgrep) by Andrew Gallant
- This is NOT a replacement for ripgrep, just a learning exercise
- Thanks to the Go community for excellent tooling and documentation

---

**GoRipGrep**: An educational text search implementation in Go. Use ripgrep for production unless you need a pure go alternaive.