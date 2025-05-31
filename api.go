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

	// Streaming search options for large files
	streamingSearch    bool                 // Enable streaming search for large files
	streamingOptions   SlidingWindowOptions // Configuration for streaming search
	largeSizeThreshold int64                // File size threshold to trigger streaming search
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

		// Streaming search defaults
		streamingSearch:    true,                          // Enable streaming search by default
		streamingOptions:   DefaultSlidingWindowOptions(), // Use default sliding window options
		largeSizeThreshold: 100 * 1024 * 1024,             // 100MB threshold for streaming search
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

		// Streaming search configuration
		StreamingSearch:    options.streamingSearch,
		StreamingOptions:   options.streamingOptions,
		LargeSizeThreshold: options.largeSizeThreshold,
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

// Streaming Search Configuration Options

// WithStreamingSearch enables or disables streaming search for large files
func WithStreamingSearch(enabled bool) Option {
	return func(opts *searchOptions) {
		opts.streamingSearch = enabled
	}
}

// WithLargeSizeThreshold sets the file size threshold (in bytes) that triggers streaming search
func WithLargeSizeThreshold(sizeBytes int64) Option {
	return func(opts *searchOptions) {
		if sizeBytes > 0 {
			opts.largeSizeThreshold = sizeBytes
		}
	}
}

// WithChunkSize sets the chunk size for streaming search operations
func WithChunkSize(chunkSize int64) Option {
	return func(opts *searchOptions) {
		if chunkSize > 0 {
			opts.streamingOptions.ChunkSize = chunkSize
		}
	}
}

// WithOverlapSize sets the overlap size between chunks in streaming search
func WithOverlapSize(overlapSize int64) Option {
	return func(opts *searchOptions) {
		if overlapSize >= 0 {
			opts.streamingOptions.OverlapSize = overlapSize
		}
	}
}

// WithMemoryThreshold sets the memory threshold for adaptive chunk sizing
func WithMemoryThreshold(threshold int64) Option {
	return func(opts *searchOptions) {
		if threshold > 0 {
			opts.streamingOptions.MemoryThreshold = threshold
		}
	}
}

// WithMaxChunkSize sets the maximum allowed chunk size for streaming search
func WithMaxChunkSize(maxSize int64) Option {
	return func(opts *searchOptions) {
		if maxSize > 0 {
			opts.streamingOptions.MaxChunkSize = maxSize
		}
	}
}

// WithMinChunkSize sets the minimum allowed chunk size for streaming search
func WithMinChunkSize(minSize int64) Option {
	return func(opts *searchOptions) {
		if minSize > 0 {
			opts.streamingOptions.MinChunkSize = minSize
		}
	}
}

// WithAdaptiveResize enables or disables adaptive chunk resizing based on memory pressure
func WithAdaptiveResize(enabled bool) Option {
	return func(opts *searchOptions) {
		opts.streamingOptions.AdaptiveResize = enabled
	}
}

// WithMemoryMapping enables or disables memory mapping for large files when available
func WithMemoryMapping(enabled bool) Option {
	return func(opts *searchOptions) {
		opts.streamingOptions.UseMemoryMap = enabled
	}
}

// WithMaxPatternLength sets the maximum expected pattern length for overlap calculation
func WithMaxPatternLength(maxLength int) Option {
	return func(opts *searchOptions) {
		if maxLength > 0 {
			opts.streamingOptions.MaxPatternLength = maxLength
		}
	}
}

// WithProgressCallback sets a callback function for progress reporting during streaming search
func WithProgressCallback(callback func(bytesProcessed, totalBytes int64, percentage float64)) Option {
	return func(opts *searchOptions) {
		opts.streamingOptions.ProgressCallback = callback
	}
}

// WithStreamingOptions sets complete streaming search options (advanced usage)
func WithStreamingOptions(options SlidingWindowOptions) Option {
	return func(opts *searchOptions) {
		opts.streamingOptions = options
	}
}
