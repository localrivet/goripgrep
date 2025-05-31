package goripgrep

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// MmapSearcher provides memory-mapped file search capabilities
type MmapSearcher struct {
	file     *os.File
	data     []byte
	fileSize int64
	mapped   bool
}

// MmapSearchOptions configures memory-mapped search behavior
type MmapSearchOptions struct {
	MinFileSize     int64 // Minimum file size to use memory mapping (default: 64MB)
	MaxMappingSize  int64 // Maximum size to memory map at once (default: 1GB)
	FallbackEnabled bool  // Enable fallback to regular file reading
}

// DefaultMmapOptions returns sensible defaults for memory-mapped search
func DefaultMmapOptions() MmapSearchOptions {
	return MmapSearchOptions{
		MinFileSize:     64 * 1024 * 1024,   // 64MB
		MaxMappingSize:  1024 * 1024 * 1024, // 1GB
		FallbackEnabled: true,
	}
}

// NewMmapSearcher creates a new memory-mapped file searcher
func NewMmapSearcher(filepath string, options MmapSearchOptions) (*MmapSearcher, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := stat.Size()
	searcher := &MmapSearcher{
		file:     file,
		fileSize: fileSize,
		mapped:   false,
	}

	// Decide whether to use memory mapping
	if fileSize >= options.MinFileSize && fileSize <= options.MaxMappingSize {
		if err := searcher.mapFile(); err != nil {
			if !options.FallbackEnabled {
				file.Close()
				return nil, fmt.Errorf("memory mapping failed and fallback disabled: %w", err)
			}
			// Continue without mapping - will use regular file reading
		}
	}

	return searcher, nil
}

// mapFile creates a memory mapping of the entire file
func (m *MmapSearcher) mapFile() error {
	if m.fileSize == 0 {
		return fmt.Errorf("cannot map empty file")
	}

	// Use platform-specific memory mapping
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return m.mapFileUnix()
	case "windows":
		return m.mapFileWindows()
	default:
		return fmt.Errorf("memory mapping not supported on %s", runtime.GOOS)
	}
}

// mapFileUnix implements memory mapping for Unix-like systems
func (m *MmapSearcher) mapFileUnix() error {
	fd := int(m.file.Fd())

	// Map the file into memory with read-only access
	data, err := syscall.Mmap(fd, 0, int(m.fileSize), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return fmt.Errorf("mmap failed: %w", err)
	}

	m.data = data
	m.mapped = true
	return nil
}

// mapFileWindows implements memory mapping for Windows
func (m *MmapSearcher) mapFileWindows() error {
	// For Windows, we'll use a simplified approach
	// In a production implementation, you'd use Windows-specific APIs
	return fmt.Errorf("Windows memory mapping not implemented in this version")
}

// Search performs a search on the memory-mapped file
func (m *MmapSearcher) Search(ctx context.Context, pattern string, matcher PatternMatcher) ([]Match, error) {
	if m.mapped {
		return m.searchMapped(ctx, pattern, matcher)
	}
	return m.searchFallback(ctx, pattern, matcher)
}

// searchMapped performs search on memory-mapped data
func (m *MmapSearcher) searchMapped(ctx context.Context, pattern string, matcher PatternMatcher) ([]Match, error) {
	if len(m.data) == 0 {
		return nil, nil
	}

	var matches []Match
	lineNum := 1
	lineStart := 0

	for i := 0; i < len(m.data); i++ {
		// Check for context cancellation periodically
		if i%10000 == 0 {
			select {
			case <-ctx.Done():
				return matches, ctx.Err()
			default:
			}
		}

		// Track line numbers
		if m.data[i] == '\n' {
			// Check if this line contains a match
			lineData := m.data[lineStart:i]
			if match := matcher.Match(lineData, pattern); match != nil {
				matches = append(matches, Match{
					File:    m.file.Name(),
					Line:    lineNum,
					Column:  match.Column,
					Content: string(lineData),
				})
			}

			lineNum++
			lineStart = i + 1
		}
	}

	// Handle last line if file doesn't end with newline
	if lineStart < len(m.data) {
		lineData := m.data[lineStart:]
		if match := matcher.Match(lineData, pattern); match != nil {
			matches = append(matches, Match{
				File:    m.file.Name(),
				Line:    lineNum,
				Column:  match.Column,
				Content: string(lineData),
			})
		}
	}

	return matches, nil
}

// searchFallback performs search using regular file reading
func (m *MmapSearcher) searchFallback(ctx context.Context, pattern string, matcher PatternMatcher) ([]Match, error) {
	// Reset file position
	if _, err := m.file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to beginning: %w", err)
	}

	// Use existing engine for fallback
	args := SearchArgs{
		Pattern: pattern,
	}

	engine, err := NewEngine(args)
	if err != nil {
		return nil, fmt.Errorf("failed to create fallback engine: %w", err)
	}

	return engine.Search(ctx, m.file.Name())
}

// Close releases resources and unmaps the file
func (m *MmapSearcher) Close() error {
	var err error

	if m.mapped && m.data != nil {
		// Unmap the memory
		if unmapErr := m.unmapFile(); unmapErr != nil {
			err = unmapErr
		}
	}

	if m.file != nil {
		if closeErr := m.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	return err
}

// unmapFile releases the memory mapping
func (m *MmapSearcher) unmapFile() error {
	if !m.mapped || m.data == nil {
		return nil
	}

	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return m.unmapFileUnix()
	case "windows":
		return m.unmapFileWindows()
	default:
		return fmt.Errorf("memory unmapping not supported on %s", runtime.GOOS)
	}
}

// unmapFileUnix releases memory mapping on Unix-like systems
func (m *MmapSearcher) unmapFileUnix() error {
	if err := syscall.Munmap(m.data); err != nil {
		return fmt.Errorf("munmap failed: %w", err)
	}

	m.data = nil
	m.mapped = false
	return nil
}

// unmapFileWindows releases memory mapping on Windows
func (m *MmapSearcher) unmapFileWindows() error {
	// Windows-specific unmapping would go here
	m.data = nil
	m.mapped = false
	return nil
}

// IsMapped returns whether the file is currently memory-mapped
func (m *MmapSearcher) IsMapped() bool {
	return m.mapped
}

// Size returns the size of the file
func (m *MmapSearcher) Size() int64 {
	return m.fileSize
}

// GetMemoryUsage returns current memory usage statistics
func (m *MmapSearcher) GetMemoryUsage() MemoryUsage {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	usage := MemoryUsage{
		MappedSize: 0,
		HeapAlloc:  memStats.HeapAlloc,
		HeapSys:    memStats.HeapSys,
		TotalAlloc: memStats.TotalAlloc,
		NumGC:      memStats.NumGC,
	}

	if m.mapped && m.data != nil {
		usage.MappedSize = uint64(len(m.data))
	}

	return usage
}

// MemoryUsage provides memory usage statistics
type MemoryUsage struct {
	MappedSize uint64 // Size of memory-mapped region
	HeapAlloc  uint64 // Bytes allocated on heap
	HeapSys    uint64 // Bytes obtained from system for heap
	TotalAlloc uint64 // Cumulative bytes allocated
	NumGC      uint32 // Number of GC cycles
}

// PatternMatcher interface for different matching strategies
type PatternMatcher interface {
	Match(data []byte, pattern string) *MatchResult
	FindAllMatches(data []byte, pattern string) []MatchResult
}

// MatchResult represents a single match result
type MatchResult struct {
	Column int
	Length int
}

// LiteralMatcher implements literal string matching
type LiteralMatcher struct{}

// Match performs literal string matching on byte data (returns first match)
func (lm *LiteralMatcher) Match(data []byte, pattern string) *MatchResult {
	matches := lm.FindAllMatches(data, pattern)
	if len(matches) > 0 {
		return &matches[0]
	}
	return nil
}

// FindAllMatches finds all occurrences of the pattern in the data
func (lm *LiteralMatcher) FindAllMatches(data []byte, pattern string) []MatchResult {
	var matches []MatchResult
	patternBytes := []byte(pattern)

	for i := 0; i <= len(data)-len(patternBytes); i++ {
		if bytesEqual(data[i:i+len(patternBytes)], patternBytes) {
			matches = append(matches, MatchResult{
				Column: i + 1, // 1-indexed
				Length: len(patternBytes),
			})
			// Continue searching after this match for overlapping patterns
		}
	}

	return matches
}

// bytesEqual compares two byte slices for equality
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// AdviseSequential hints to the OS that we'll read the file sequentially
func (m *MmapSearcher) AdviseSequential() error {
	if !m.mapped || m.data == nil {
		return nil
	}

	// Use madvise to hint sequential access pattern
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		return m.adviseSequentialUnix()
	default:
		// Not supported on this platform
		return nil
	}
}

// adviseSequentialUnix provides sequential access hints on Unix systems
func (m *MmapSearcher) adviseSequentialUnix() error {
	const MADV_SEQUENTIAL = 2 // Sequential access hint

	// Use unsafe to call madvise
	dataPtr := uintptr(unsafe.Pointer(&m.data[0]))
	dataLen := uintptr(len(m.data))

	_, _, errno := syscall.Syscall(syscall.SYS_MADVISE, dataPtr, dataLen, MADV_SEQUENTIAL)
	if errno != 0 {
		return fmt.Errorf("madvise failed: %v", errno)
	}

	return nil
}
