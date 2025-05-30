package goripgrep

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"
)

// Option represents a functional option for configuring searches
type Option func(*searchOptions)

// searchOptions holds the configuration for a search operation
type searchOptions struct {
	ctx           context.Context
	workers       int
	bufferSize    int
	maxResults    int
	optimization  bool
	gitignore     bool
	ignoreCase    bool
	caseSensitive bool
	hidden        bool
	symlinks      bool
	recursive     bool
	filePattern   string
	contextLines  int
	timeout       time.Duration
}

// defaultOptions returns the default search options
func defaultOptions() *searchOptions {
	return &searchOptions{
		ctx:           context.Background(),
		workers:       4,
		bufferSize:    64 * 1024, // 64KB
		maxResults:    1000,
		optimization:  true,
		gitignore:     true,
		ignoreCase:    false,
		caseSensitive: true,
		hidden:        false,
		symlinks:      false,
		recursive:     false,
		contextLines:  0,
		timeout:       30 * time.Second,
	}
}

// Find performs a search with functional options
func Find(pattern, path string, opts ...Option) (*SearchResults, error) {
	// Validate inputs
	if pattern == "" {
		return nil, fmt.Errorf("pattern cannot be empty")
	}
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}

	// Check if path exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("path error: %w", err)
	}

	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Apply timeout to context if specified
	ctx := options.ctx
	if options.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.timeout)
		defer cancel()
	}

	// Validate regex pattern early
	if !isLiteralPattern(pattern) {
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	// Create SearchConfig from options
	config := SearchConfig{
		SearchPath:      path,
		MaxWorkers:      options.workers,
		BufferSize:      options.bufferSize,
		MaxResults:      options.maxResults,
		UseOptimization: options.optimization,
		UseGitignore:    options.gitignore,
		IgnoreCase:      options.ignoreCase,
		IncludeHidden:   options.hidden,
		FollowSymlinks:  options.symlinks,
		Recursive:       options.recursive,
		FilePattern:     options.filePattern,
		ContextLines:    options.contextLines,
		Timeout:         options.timeout,
	}

	// Create and use SearchEngine
	engine := NewSearchEngine(config)
	return engine.Search(ctx, pattern)
}

// Context and Cancellation Options

// WithContext sets the context for cancellation and timeout control
func WithContext(ctx context.Context) Option {
	return func(opts *searchOptions) {
		opts.ctx = ctx
	}
}

// Performance Options

// WithWorkers sets the number of concurrent workers
func WithWorkers(count int) Option {
	return func(opts *searchOptions) {
		if count > 0 {
			opts.workers = count
		}
	}
}

// WithBufferSize sets the I/O buffer size in bytes
func WithBufferSize(size int) Option {
	return func(opts *searchOptions) {
		if size > 0 {
			opts.bufferSize = size
		}
	}
}

// WithMaxResults sets the maximum number of results to return
func WithMaxResults(max int) Option {
	return func(opts *searchOptions) {
		if max > 0 {
			opts.maxResults = max
		}
	}
}

// WithOptimization enables or disables performance optimizations
func WithOptimization(enabled bool) Option {
	return func(opts *searchOptions) {
		opts.optimization = enabled
	}
}

// Search Behavior Options

// WithIgnoreCase enables case-insensitive search
func WithIgnoreCase() Option {
	return func(opts *searchOptions) {
		opts.ignoreCase = true
		opts.caseSensitive = false
	}
}

// WithCaseSensitive enables case-sensitive search (default)
func WithCaseSensitive() Option {
	return func(opts *searchOptions) {
		opts.ignoreCase = false
		opts.caseSensitive = true
	}
}

// WithContextLines sets the number of context lines around matches
func WithContextLines(lines int) Option {
	return func(opts *searchOptions) {
		if lines >= 0 {
			opts.contextLines = lines
		}
	}
}

// WithTimeout sets the search timeout
func WithTimeout(duration time.Duration) Option {
	return func(opts *searchOptions) {
		if duration > 0 {
			opts.timeout = duration
		}
	}
}

// File Filtering Options

// WithFilePattern sets a file pattern filter (glob-style)
func WithFilePattern(pattern string) Option {
	return func(opts *searchOptions) {
		opts.filePattern = pattern
	}
}

// WithGitignore enables or disables gitignore filtering
func WithGitignore(enabled bool) Option {
	return func(opts *searchOptions) {
		opts.gitignore = enabled
	}
}

// WithHidden includes hidden files in the search
func WithHidden() Option {
	return func(opts *searchOptions) {
		opts.hidden = true
	}
}

// WithSymlinks enables following symbolic links
func WithSymlinks() Option {
	return func(opts *searchOptions) {
		opts.symlinks = true
	}
}

// WithRecursive sets whether to search directories recursively
// By default, search is non-recursive (only immediate directory)
func WithRecursive(recursive bool) Option {
	return func(opts *searchOptions) {
		opts.recursive = recursive
	}
}
