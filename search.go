package goripgrep

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SearchConfig holds configuration for the search engine
type SearchConfig struct {
	SearchPath      string
	MaxWorkers      int
	BufferSize      int
	MaxResults      int
	UseOptimization bool
	UseGitignore    bool
	IgnoreCase      bool
	IncludeHidden   bool
	FollowSymlinks  bool
	Recursive       bool
	FilePattern     string
	ContextLines    int
	Timeout         time.Duration

	// Streaming search configuration for large files
	StreamingSearch    bool                 // Enable streaming search for large files
	StreamingOptions   SlidingWindowOptions // Configuration for streaming search
	LargeSizeThreshold int64                // File size threshold to trigger streaming search

	// Performance optimization options
	FastFileFiltering         bool // Enable fast extension-based filtering before content checks
	EarlyBinaryDetection      bool // Enable early binary detection (first 512 bytes)
	OptimizedWalking          bool // Use filepath.WalkDir instead of filepath.Walk
	SkipKnownBinary           bool // Skip known binary extensions immediately
	LiteralStringOptimization bool // Use fast string search for literal patterns
	MemoryPooling             bool // Use object pools to reduce allocations
	LargeFileBuffers          bool // Use larger I/O buffers for better performance
	RegexCaching              bool // Cache compiled regex patterns
	MemoryMappedFiles         bool // Use memory-mapped files for large files
}

// SearchEngine provides integrated search functionality
type SearchEngine struct {
	config          SearchConfig
	gitignoreEngine *GitignoreEngine
	stats           SearchStats
}

// SearchStats tracks search performance metrics
type SearchStats struct {
	FilesScanned int64
	FilesSkipped int64
	FilesIgnored int64
	BytesScanned int64
	MatchesFound int64
	Duration     time.Duration
	StartTime    time.Time
	EndTime      time.Time
}

// SearchResults contains search results and metadata
type SearchResults struct {
	Matches []Match
	Stats   SearchStats
	Query   string
}

// HasMatches returns true if any matches were found
func (r *SearchResults) HasMatches() bool {
	return len(r.Matches) > 0
}

// Count returns the number of matches
func (r *SearchResults) Count() int {
	return len(r.Matches)
}

// Files returns the unique files that contain matches
func (r *SearchResults) Files() []string {
	fileSet := make(map[string]bool)
	for _, match := range r.Matches {
		fileSet[match.File] = true
	}

	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}

// NewSearchEngine creates a new integrated search engine
func NewSearchEngine(config SearchConfig) *SearchEngine {
	engine := &SearchEngine{
		config: config,
	}

	// Initialize engines - ignore errors and continue without optimization if initialization fails
	_ = engine.initializeEngines()

	return engine
}

// initializeEngines initializes the various search engines
func (e *SearchEngine) initializeEngines() error {
	// Note: Engine will be created per-search with the actual pattern
	// since it needs the pattern for optimization

	// Initialize gitignore engine if enabled
	if e.config.UseGitignore {
		e.gitignoreEngine = NewGitignoreEngine(e.config.SearchPath)
	}

	return nil
}

// Search performs an integrated search with all enabled features
func (e *SearchEngine) Search(ctx context.Context, pattern string) (*SearchResults, error) {
	startTime := time.Now()

	// Reset stats for this search
	e.stats = SearchStats{StartTime: startTime}

	// Initialize results
	results := &SearchResults{
		Query: pattern,
		Stats: SearchStats{StartTime: startTime},
	}

	// Initialize engines for this specific pattern
	_ = e.initializeEngines()

	// Perform the search
	if err := e.performSearch(ctx, pattern, results); err != nil {
		return nil, err
	}

	// Copy accumulated stats from engine to results
	results.Stats.FilesScanned = e.stats.FilesScanned
	results.Stats.FilesSkipped = e.stats.FilesSkipped
	results.Stats.FilesIgnored = e.stats.FilesIgnored
	results.Stats.BytesScanned = e.stats.BytesScanned
	results.Stats.MatchesFound = int64(len(results.Matches))

	// Update final stats
	results.Stats.EndTime = time.Now()
	results.Stats.Duration = results.Stats.EndTime.Sub(results.Stats.StartTime)

	return results, nil
}

// performSearch executes the actual search using the configured engines
func (e *SearchEngine) performSearch(ctx context.Context, pattern string, results *SearchResults) error {
	// Create channels for communication
	filesChan := make(chan string, e.config.MaxWorkers*2)
	resultsChan := make(chan []Match, e.config.MaxWorkers)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < e.config.MaxWorkers; i++ {
		wg.Add(1)
		go e.searchWorker(ctx, pattern, filesChan, resultsChan, &wg)
	}

	// Start file walker
	go e.walkFiles(ctx, filesChan)

	// Collect results
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Process results
	for workerResults := range resultsChan {
		results.Matches = append(results.Matches, workerResults...)
		e.stats.MatchesFound += int64(len(workerResults))

		// Check if we've hit the max results limit
		if len(results.Matches) >= e.config.MaxResults {
			break
		}
	}

	return nil
}

// searchWorker processes files from the files channel
func (e *SearchEngine) searchWorker(ctx context.Context, pattern string, filesChan <-chan string, resultsChan chan<- []Match, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range filesChan {
		select {
		case <-ctx.Done():
			return
		default:
			// Track file size for bytes scanned
			if info, err := os.Stat(filePath); err == nil {
				e.stats.BytesScanned += info.Size()
			}

			fileResults, err := e.searchFile(ctx, pattern, filePath)
			if err != nil {
				// Log error but continue processing
				continue
			}

			if len(fileResults) > 0 {
				resultsChan <- fileResults
			}

			e.stats.FilesScanned++
		}
	}
}

// searchFile processes an individual file (updated to support memory mapping)
func (e *SearchEngine) searchFile(ctx context.Context, pattern string, filePath string) ([]Match, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Get file info for size-based decisions
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Track stats
	e.stats.FilesScanned++
	e.stats.BytesScanned += info.Size()

	// Use memory-mapped files for large files if enabled
	if e.config.MemoryMappedFiles && info.Size() > 1024*1024 { // 1MB threshold
		return e.mmapSearch(ctx, pattern, filePath, info.Size())
	}

	// Use streaming search for large files if enabled and file is above threshold
	if e.config.StreamingSearch && info.Size() > e.config.LargeSizeThreshold {
		return e.streamingSearch(ctx, pattern, filePath)
	}

	// For smaller files, use simple search
	return e.simpleSearch(ctx, pattern, filePath)
}

// mmapSearch performs memory-mapped file search for large files
func (e *SearchEngine) mmapSearch(ctx context.Context, pattern string, filePath string, fileSize int64) ([]Match, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if fileSize == 0 {
		return nil, nil
	}

	// Memory map the file
	data, err := syscall.Mmap(int(file.Fd()), 0, int(fileSize), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		// Fallback to regular search if mmap fails
		return e.simpleSearch(ctx, pattern, filePath)
	}
	defer func() {
		if unmapErr := syscall.Munmap(data); unmapErr != nil {
			// Log error but don't fail the search
			_ = unmapErr
		}
	}()

	// Convert bytes to string safely
	content := string(data)

	// Split into lines efficiently
	lines := strings.Split(content, "\n")

	// Compile regex
	var regex *regexp.Regexp
	if e.config.IgnoreCase {
		regex, err = regexp.Compile(`(?i)` + pattern)
	} else {
		regex, err = regexp.Compile(pattern)
	}
	if err != nil {
		return nil, err
	}

	var matches []Match

	// Search each line
	for lineNum, line := range lines {
		// Check for context cancellation periodically
		if lineNum%1000 == 0 {
			select {
			case <-ctx.Done():
				return matches, ctx.Err()
			default:
			}
		}

		// Find all matches in this line
		indices := regex.FindAllStringIndex(line, -1)
		for _, match := range indices {
			matchObj := Match{
				File:    filePath,
				Line:    lineNum + 1,
				Column:  match[0] + 1,
				Content: line,
			}

			// Add context lines if requested
			if e.config.ContextLines > 0 {
				matchObj.Context = e.extractContextLinesFromSlice(lines, lineNum, e.config.ContextLines)
			}

			matches = append(matches, matchObj)
		}
	}

	return matches, nil
}

// extractContextLinesFromSlice extracts context lines from a string slice
func (e *SearchEngine) extractContextLinesFromSlice(lines []string, centerLine int, contextLines int) []string {
	var contextResult []string

	// Add lines before the match
	for i := contextLines; i > 0; i-- {
		lineIndex := centerLine - i
		if lineIndex >= 0 && lineIndex < len(lines) {
			contextResult = append(contextResult, lines[lineIndex])
		}
	}

	// Add lines after the match
	for i := 1; i <= contextLines; i++ {
		lineIndex := centerLine + i
		if lineIndex >= 0 && lineIndex < len(lines) {
			contextResult = append(contextResult, lines[lineIndex])
		}
	}

	return contextResult
}

// streamingSearch performs streaming search on large files using the sliding window approach
func (e *SearchEngine) streamingSearch(ctx context.Context, pattern string, filePath string) ([]Match, error) {
	// Create a sliding window searcher with the configured options
	searcher, err := NewSlidingWindowSearcher(filePath, pattern, e.config.StreamingOptions)
	if err != nil {
		// Fall back to simple search if streaming search fails to initialize
		return e.simpleSearch(ctx, pattern, filePath)
	}
	defer searcher.Close()

	// Perform the streaming search
	matches, err := searcher.Search(ctx)
	if err != nil {
		// Fall back to simple search if streaming search fails
		return e.simpleSearch(ctx, pattern, filePath)
	}

	return matches, nil
}

// simpleSearch performs a basic search without optimization
func (e *SearchEngine) simpleSearch(ctx context.Context, pattern string, filePath string) ([]Match, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read all lines first if we need context
	var allLines []string
	if e.config.ContextLines > 0 {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	var results []Match
	scanner := bufio.NewScanner(file)

	// Reset file position if we read it for context
	if e.config.ContextLines > 0 {
		if _, err := file.Seek(0, 0); err != nil {
			return nil, err
		}
		scanner = bufio.NewScanner(file)
	}

	lineNum := 1

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		line := scanner.Text()

		// Simple pattern matching
		var matched bool
		if e.config.IgnoreCase {
			matched = strings.Contains(strings.ToLower(line), strings.ToLower(pattern))
		} else {
			matched = strings.Contains(line, pattern)
		}

		if matched {
			result := Match{
				File:    filePath,
				Line:    lineNum,
				Content: line,
			}

			// Add context lines if requested
			if e.config.ContextLines > 0 && len(allLines) > 0 {
				result.Context = e.extractContextLines(allLines, lineNum-1, e.config.ContextLines)
			}

			results = append(results, result)
		}

		lineNum++
	}

	return results, scanner.Err()
}

// extractContextLines extracts context lines around a match
func (e *SearchEngine) extractContextLines(allLines []string, matchLineIndex int, contextLines int) []string {
	var context []string

	// Add lines before the match
	start := matchLineIndex - contextLines
	if start < 0 {
		start = 0
	}

	// Add lines after the match
	end := matchLineIndex + contextLines + 1
	if end > len(allLines) {
		end = len(allLines)
	}

	for i := start; i < end; i++ {
		if i != matchLineIndex {
			context = append(context, allLines[i])
		}
	}

	return context
}

// walkFiles walks the directory tree and sends files to the channel
func (e *SearchEngine) walkFiles(ctx context.Context, filesChan chan<- string) {
	defer close(filesChan)

	// Clean the search path for consistent comparison
	searchPath, err := filepath.Abs(e.config.SearchPath)
	if err != nil {
		searchPath = e.config.SearchPath
	}

	// Phase 2 optimization: Use optimized walking if enabled
	if e.config.OptimizedWalking {
		err = e.optimizedWalk(ctx, searchPath, filesChan)
	} else {
		// Original logic
		if e.config.Recursive {
			// Recursive mode: walk the entire directory tree
			visited := make(map[string]bool)
			err = e.walkPath(ctx, searchPath, visited, filesChan)
		} else {
			// Non-recursive mode: only process files in the immediate directory
			err = e.processDirectory(ctx, searchPath, filesChan)
		}
	}

	// Silently continue on walk errors (no logging)
	_ = err
}

// walkPath recursively walks a path (for recursive mode)
func (e *SearchEngine) walkPath(ctx context.Context, path string, visited map[string]bool, filesChan chan<- string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get file info using Lstat to detect symlinks
	info, err := os.Lstat(path)
	if err != nil {
		return nil // Continue on errors
	}

	// Handle symlinks
	if info.Mode()&os.ModeSymlink != 0 {
		if !e.config.FollowSymlinks {
			// Skip symlinks if not following them
			return nil
		}

		// Resolve the symlink target
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil // Continue on errors
		}

		// Check for cycles using the resolved path
		if visited[target] {
			// Cycle detected, skip
			return nil
		}

		// Mark as visited and continue with the target
		visited[target] = true
		defer delete(visited, target)

		return e.walkPath(ctx, target, visited, filesChan)
	}

	// Handle regular files
	if !info.IsDir() {
		// Check if we should ignore this file
		if e.shouldIgnoreFile(path, info) {
			e.stats.FilesSkipped++
			return nil
		}

		filesChan <- path
		return nil
	}

	// Handle directories - recurse into them
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil // Continue on errors
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())
		if err := e.walkPath(ctx, entryPath, visited, filesChan); err != nil {
			return err
		}
	}

	return nil
}

// processDirectory processes only files in the immediate directory (for non-recursive mode)
func (e *SearchEngine) processDirectory(ctx context.Context, dirPath string, filesChan chan<- string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Get directory info
	info, err := os.Stat(dirPath)
	if err != nil {
		return err
	}

	// If it's a single file, process it
	if !info.IsDir() {
		if !e.shouldIgnoreFile(dirPath, info) {
			filesChan <- dirPath
		} else {
			e.stats.FilesSkipped++
		}
		return nil
	}

	// Read directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	// Process only files (not subdirectories)
	for _, entry := range entries {
		// Skip directories entirely in non-recursive mode
		if entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(dirPath, entry.Name())
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}

		if !e.shouldIgnoreFile(entryPath, entryInfo) {
			filesChan <- entryPath
		} else {
			e.stats.FilesSkipped++
		}
	}

	return nil
}

// shouldIgnoreFile determines if a file should be ignored based on various criteria
func (e *SearchEngine) shouldIgnoreFile(path string, info os.FileInfo) bool {
	// Fast extension-based binary filtering (Phase 1 optimization)
	if e.config.SkipKnownBinary && e.isKnownBinaryExtension(path) {
		return true
	}

	// Apply gitignore filtering if enabled
	if e.config.UseGitignore && e.gitignoreEngine != nil {
		if e.gitignoreEngine.ShouldIgnore(path) {
			e.stats.FilesIgnored++
			return true
		}
	}

	// Apply file pattern filtering
	if e.config.FilePattern != "" {
		matched, err := filepath.Match(e.config.FilePattern, info.Name())
		if err != nil || !matched {
			return true
		}
	}

	// Skip hidden files if not included
	if !e.config.IncludeHidden && strings.HasPrefix(info.Name(), ".") {
		return true
	}

	// Fast file filtering with early text detection
	if e.config.FastFileFiltering && !e.isLikelyTextFile(path) {
		return true
	}

	// Enhanced binary detection
	if e.config.EarlyBinaryDetection {
		if e.isBinaryFileOptimized(path) {
			return true
		}
	} else {
		// Fallback to existing binary detection
		if isBinaryFile(path) {
			return true
		}
	}

	return false
}

// isKnownBinaryExtension performs fast extension-based binary detection
func (e *SearchEngine) isKnownBinaryExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	binaryExtensions := map[string]bool{
		// Images
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
		".tiff": true, ".tif": true, ".webp": true, ".ico": true,
		".psd": true, ".ai": true, ".eps": true, ".raw": true, ".cr2": true,
		// Videos
		".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
		".mkv": true, ".webm": true, ".m4v": true, ".3gp": true, ".mpg": true,
		// Audio
		".mp3": true, ".wav": true, ".flac": true, ".aac": true, ".ogg": true,
		".wma": true, ".m4a": true, ".opus": true, ".aiff": true,
		// Archives
		".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true,
		".rar": true, ".7z": true, ".dmg": true, ".iso": true, ".deb": true,
		// Executables
		".exe": true, ".dll": true, ".so": true, ".dylib": true, ".a": true,
		".lib": true, ".o": true, ".obj": true, ".bin": true, ".class": true,
		".jar": true, ".war": true, ".ear": true, ".pyc": true, ".pyo": true,
		// Documents (binary formats)
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,
		// Fonts
		".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
		// Database files
		".db": true, ".sqlite": true, ".sqlite3": true, ".mdb": true,
	}
	return binaryExtensions[ext]
}

// isLikelyTextFile performs fast text file detection based on extension
func (e *SearchEngine) isLikelyTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	textExtensions := map[string]bool{
		// Programming languages
		".go": true, ".py": true, ".js": true, ".ts": true, ".java": true,
		".c": true, ".cpp": true, ".cxx": true, ".cc": true, ".h": true, ".hpp": true,
		".rs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true,
		".scala": true, ".clj": true, ".hs": true, ".ml": true, ".fs": true,
		".vb": true, ".cs": true, ".pas": true, ".pl": true, ".pm": true,
		".lua": true, ".r": true, ".m": true, ".asm": true, ".s": true,
		// Web technologies
		".html": true, ".htm": true, ".xhtml": true, ".xml": true, ".xsl": true,
		".css": true, ".scss": true, ".sass": true, ".less": true,
		".json": true, ".yaml": true, ".yml": true, ".toml": true,
		".vue": true, ".jsx": true, ".tsx": true, ".svelte": true,
		// Documentation and text
		".txt": true, ".md": true, ".markdown": true, ".rst": true,
		".tex": true, ".ltx": true, ".org": true, ".adoc": true,
		".rtf": true, ".man": true, ".1": true, ".2": true, ".3": true,
		// Configuration and data
		".cfg": true, ".conf": true, ".config": true, ".ini": true,
		".env": true, ".properties": true, ".plist": true,
		".csv": true, ".tsv": true, ".log": true, ".sql": true,
		// Build and project files
		".mk": true, ".makefile": true, ".cmake": true, ".gradle": true,
		".sbt": true, ".cabal": true, ".gemspec": true, ".podspec": true,
		".dockerfile": true, ".dockerignore": true, ".gitignore": true,
		// Scripts and shells
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".ps1": true, ".psm1": true, ".bat": true, ".cmd": true,
	}

	// If we know it's a text extension, return true
	if textExtensions[ext] {
		return true
	}

	// Files without extensions might be text (e.g., Makefile, README)
	if ext == "" {
		name := strings.ToLower(filepath.Base(filePath))
		textFiles := map[string]bool{
			"makefile": true, "dockerfile": true, "readme": true,
			"changelog": true, "license": true, "authors": true,
			"contributors": true, "copying": true, "install": true,
			"news": true, "todo": true, "version": true,
		}
		return textFiles[name]
	}

	// Unknown extensions - let binary detection decide
	return true
}

// isBinaryFileOptimized uses optimized binary detection (first 512 bytes)
func (e *SearchEngine) isBinaryFileOptimized(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return true // If we can't read it, treat as binary
	}
	defer file.Close()

	// Read first 512 bytes (same as Git's binary detection)
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return true
	}

	// Check for null bytes (strong binary indicator)
	nullCount := 0
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			nullCount++
		}
	}

	// If more than 0.1% are null bytes, consider it binary
	if n > 0 && float64(nullCount)/float64(n) > 0.001 {
		return true
	}

	// Check for high proportion of non-printable characters
	nonPrintable := 0
	for i := 0; i < n; i++ {
		b := buffer[i]
		// Count non-printable characters (excluding common whitespace)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
		if b > 126 {
			nonPrintable++
		}
	}

	// If more than 5% are non-printable, likely binary
	if n > 0 && float64(nonPrintable)/float64(n) > 0.05 {
		return true
	}

	return false
}

// optimizedWalk performs fast directory walking using filepath.WalkDir (Phase 2 optimization)
func (e *SearchEngine) optimizedWalk(ctx context.Context, searchPath string, filesChan chan<- string) error {
	// For non-recursive mode, use processDirectory instead
	if !e.config.Recursive {
		return e.processDirectory(ctx, searchPath, filesChan)
	}

	return filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Continue on errors
			return nil
		}

		// Skip directories
		if d.IsDir() {
			// Skip hidden directories if not including hidden files
			if !e.config.IncludeHidden && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}

			// Skip known directories to ignore for performance
			if e.shouldSkipDirectory(d.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		// Fast file filtering using DirEntry (no need to call os.Stat)
		info, err := d.Info()
		if err != nil {
			return nil // Skip files we can't stat
		}

		// Apply all file filters
		if !e.shouldIgnoreFile(path, info) {
			select {
			case filesChan <- path:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	})
}

// shouldSkipDirectory determines if a directory should be skipped entirely
func (e *SearchEngine) shouldSkipDirectory(dirName string) bool {
	// Skip common binary/build directories for performance
	skipDirs := map[string]bool{
		"node_modules":  true,
		".git":          true,
		".svn":          true,
		".hg":           true,
		"target":        true,
		"build":         true,
		"dist":          true,
		"out":           true,
		"bin":           true,
		"pkg":           true,
		"vendor":        true,
		".vscode":       true,
		".idea":         true,
		"__pycache__":   true,
		".pytest_cache": true,
		".cache":        true,
		".tmp":          true,
		".DS_Store":     true,
	}

	return skipDirs[dirName]
}

// GetSummary returns a summary of the search results
func (r *SearchResults) GetSummary() SearchSummary {
	return SearchSummary{
		Pattern:        r.Query,
		TotalMatches:   len(r.Matches),
		FilesScanned:   int(r.Stats.FilesScanned),
		FilesSkipped:   int(r.Stats.FilesSkipped),
		FilesIgnored:   int(r.Stats.FilesIgnored),
		Duration:       r.Stats.Duration,
		FilesPerSecond: float64(r.Stats.FilesScanned) / r.Stats.Duration.Seconds(),
	}
}

// SearchSummary provides a concise summary of search results
type SearchSummary struct {
	Pattern        string
	TotalMatches   int
	FilesScanned   int
	FilesSkipped   int
	FilesIgnored   int
	Duration       time.Duration
	FilesPerSecond float64
}

// GetPerformanceReport generates a detailed performance report
func (e *SearchEngine) GetPerformanceReport() PerformanceReport {
	return PerformanceReport{
		Config: e.config,
		Stats:  e.stats,
		Engines: EngineStatus{
			OptimizedEngine: e.config.UseOptimization,
			GitignoreEngine: e.gitignoreEngine != nil,
		},
	}
}

// PerformanceReport provides detailed performance information
type PerformanceReport struct {
	Config  SearchConfig
	Stats   SearchStats
	Engines EngineStatus
}

// EngineStatus shows which engines are active
type EngineStatus struct {
	OptimizedEngine bool
	GitignoreEngine bool
}

// Benchmark runs a performance benchmark
func (e *SearchEngine) Benchmark(ctx context.Context, patterns []string, iterations int) (*BenchmarkResults, error) {
	results := &BenchmarkResults{
		Patterns:   patterns,
		Iterations: iterations,
		Results:    make([]BenchmarkResult, 0, len(patterns)*iterations),
	}

	for _, pattern := range patterns {
		for i := 0; i < iterations; i++ {
			start := time.Now()
			searchResults, err := e.Search(ctx, pattern)
			duration := time.Since(start)

			result := BenchmarkResult{
				Pattern:      pattern,
				Iteration:    i + 1,
				Duration:     duration,
				MatchesFound: len(searchResults.Matches),
				Error:        err,
			}

			results.Results = append(results.Results, result)
		}
	}

	return results, nil
}

// BenchmarkResults holds benchmark test results
type BenchmarkResults struct {
	Patterns   []string
	Iterations int
	Results    []BenchmarkResult
}

// BenchmarkResult represents a single benchmark run
type BenchmarkResult struct {
	Pattern      string
	Iteration    int
	Duration     time.Duration
	MatchesFound int
	Error        error
}

// GetAveragePerformance calculates average performance metrics
func (br *BenchmarkResults) GetAveragePerformance() map[string]BenchmarkStats {
	stats := make(map[string]BenchmarkStats)

	for _, pattern := range br.Patterns {
		var totalDuration time.Duration
		var totalMatches int
		var count int

		for _, result := range br.Results {
			if result.Pattern == pattern && result.Error == nil {
				totalDuration += result.Duration
				totalMatches += result.MatchesFound
				count++
			}
		}

		if count > 0 {
			stats[pattern] = BenchmarkStats{
				AverageDuration: totalDuration / time.Duration(count),
				AverageMatches:  float64(totalMatches) / float64(count),
				Iterations:      count,
			}
		}
	}

	return stats
}

// BenchmarkStats holds statistical information about benchmark results
type BenchmarkStats struct {
	AverageDuration time.Duration
	AverageMatches  float64
	Iterations      int
}
