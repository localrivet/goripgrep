package goripgrep

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// ByteFrequency represents frequency data for byte selection optimization
var ByteFrequency = [256]int{
	// Pre-computed frequency table based on common text analysis
	// Lower values = rarer bytes (better for memchr-style scanning)
	0: 100, 1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1, 7: 1,
	8: 1, 9: 50, 10: 80, 11: 1, 12: 1, 13: 30, 14: 1, 15: 1,
	16: 1, 17: 1, 18: 1, 19: 1, 20: 1, 21: 1, 22: 1, 23: 1,
	24: 1, 25: 1, 26: 1, 27: 1, 28: 1, 29: 1, 30: 1, 31: 1,
	32: 90, 33: 10, 34: 15, 35: 5, 36: 3, 37: 5, 38: 8, 39: 20,
	40: 25, 41: 25, 42: 8, 43: 5, 44: 35, 45: 25, 46: 30, 47: 15,
	48: 40, 49: 35, 50: 30, 51: 25, 52: 25, 53: 25, 54: 25, 55: 25,
	56: 25, 57: 25, 58: 20, 59: 20, 60: 8, 61: 15, 62: 8, 63: 10,
	64: 5, 65: 35, 66: 20, 67: 25, 68: 20, 69: 30, 70: 15, 71: 15,
	72: 20, 73: 30, 74: 5, 75: 8, 76: 20, 77: 15, 78: 25, 79: 20,
	80: 15, 81: 2, 82: 25, 83: 25, 84: 30, 85: 15, 86: 8, 87: 15,
	88: 3, 89: 10, 90: 2, 91: 8, 92: 5, 93: 8, 94: 3, 95: 10,
	96: 5, 97: 70, 98: 20, 99: 30, 100: 35, 101: 85, 102: 20, 103: 20,
	104: 40, 105: 60, 106: 5, 107: 8, 108: 35, 109: 25, 110: 55, 111: 60,
	112: 20, 113: 3, 114: 50, 115: 50, 116: 70, 117: 25, 118: 15, 119: 15,
	120: 5, 121: 15, 122: 3, 123: 8, 124: 3, 125: 8, 126: 2, 127: 1,
	// UTF-8 continuation bytes and common Unicode ranges
	128: 1, 129: 1, 130: 1, 131: 1, 132: 1, 133: 1, 134: 1, 135: 1,
	136: 1, 137: 1, 138: 1, 139: 1, 140: 1, 141: 1, 142: 1, 143: 1,
	144: 1, 145: 1, 146: 1, 147: 1, 148: 1, 149: 1, 150: 1, 151: 1,
	152: 1, 153: 1, 154: 1, 155: 1, 156: 1, 157: 1, 158: 1, 159: 1,
	160: 5, 161: 3, 162: 3, 163: 3, 164: 3, 165: 3, 166: 2, 167: 3,
	168: 3, 169: 3, 170: 3, 171: 3, 172: 2, 173: 3, 174: 3, 175: 3,
	176: 3, 177: 3, 178: 3, 179: 3, 180: 3, 181: 3, 182: 3, 183: 3,
	184: 3, 185: 3, 186: 3, 187: 3, 188: 3, 189: 3, 190: 3, 191: 3,
	192: 5, 193: 5, 194: 10, 195: 15, 196: 10, 197: 8, 198: 5, 199: 8,
	200: 8, 201: 8, 202: 8, 203: 8, 204: 8, 205: 8, 206: 8, 207: 8,
	208: 60, 209: 50, 210: 8, 211: 8, 212: 8, 213: 8, 214: 8, 215: 3,
	216: 5, 217: 5, 218: 5, 219: 5, 220: 5, 221: 5, 222: 5, 223: 8,
	224: 15, 225: 15, 226: 15, 227: 15, 228: 15, 229: 15, 230: 15, 231: 15,
	232: 15, 233: 15, 234: 15, 235: 15, 236: 15, 237: 15, 238: 15, 239: 15,
	240: 8, 241: 8, 242: 8, 243: 8, 244: 8, 245: 5, 246: 5, 247: 5,
	248: 3, 249: 3, 250: 3, 251: 3, 252: 2, 253: 2, 254: 1, 255: 1,
}

// Engine provides high-performance text search with advanced optimizations
type Engine struct {
	pattern      string
	regex        *regexp.Regexp
	isLiteral    bool
	ignoreCase   bool
	searchBytes  []byte
	rareByte     byte
	rareByteIdx  int
	contextLines int

	// Performance settings
	bufferSize   int
	workerCount  int
	prefetchSize int

	// Advanced optimizations
	optimizedEngine *OptimizedEngine
	dfaCache        *DFACache

	// Compression support
	compressionDetector *CompressionDetector
	streamDecompressor  *StreamingDecompressor

	// Statistics
	bytesScanned int64
	filesScanned int64
	matchesFound int64
}

// NewEngine creates a high-performance search engine
func NewEngine(args SearchArgs) (*Engine, error) {
	engine := &Engine{
		pattern:      args.Pattern,
		ignoreCase:   args.IgnoreCase != nil && *args.IgnoreCase,
		bufferSize:   64 * 1024, // 64KB buffer for optimal I/O
		workerCount:  runtime.NumCPU(),
		prefetchSize: 8 * 1024, // 8KB prefetch

		// Initialize optimizations
		optimizedEngine: NewOptimizedEngine(),
		dfaCache:        NewDFACache(1000, 30*time.Minute),

		// Initialize compression support
		compressionDetector: NewCompressionDetector(),
		streamDecompressor:  NewStreamingDecompressor(64 * 1024),
	}

	// Set context lines if provided
	if args.ContextLines != nil {
		engine.contextLines = *args.ContextLines
	}

	// Determine if pattern is literal
	engine.isLiteral = isLiteralPattern(args.Pattern)

	if engine.isLiteral {
		// Optimize literal search
		if engine.ignoreCase {
			engine.searchBytes = []byte(strings.ToLower(args.Pattern))
		} else {
			engine.searchBytes = []byte(args.Pattern)
		}
		engine.findRareByte()
	} else {
		// Compile regex with DFA caching
		var err error
		engine.regex, err = engine.dfaCache.GetOrCompile(args.Pattern, engine.getRegexFlags())
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}

		// Try to extract literals from regex for optimization
		engine.extractLiterals()
	}

	return engine, nil
}

// getRegexFlags returns the appropriate regex flags for compilation
func (e *Engine) getRegexFlags() string {
	if e.ignoreCase {
		return "(?i)"
	}
	return ""
}

// findRareByte selects the rarest byte in the pattern for optimized scanning
func (e *Engine) findRareByte() {
	if len(e.searchBytes) == 0 {
		return
	}

	minFreq := ByteFrequency[e.searchBytes[0]]
	e.rareByte = e.searchBytes[0]
	e.rareByteIdx = 0

	for i, b := range e.searchBytes {
		if ByteFrequency[b] < minFreq {
			minFreq = ByteFrequency[b]
			e.rareByte = b
			e.rareByteIdx = i
		}
	}
}

// extractLiterals attempts to extract literal substrings from regex patterns
func (e *Engine) extractLiterals() {
	// Simple literal extraction for common patterns
	pattern := e.pattern

	// Skip patterns with character classes - they don't have useful literals
	if strings.Contains(pattern, "[") && strings.Contains(pattern, "]") {
		return
	}

	// Skip patterns that are just quantifiers
	if len(pattern) <= 2 && strings.ContainsAny(pattern, "+*?") {
		return
	}

	// Remove common regex anchors and modifiers
	pattern = strings.TrimPrefix(pattern, "^")
	pattern = strings.TrimSuffix(pattern, "$")

	// Look for literal sequences
	if strings.Contains(pattern, "|") {
		// Handle alternations - find common prefix/suffix or use first alternative
		parts := strings.Split(pattern, "|")
		if len(parts) > 1 {
			commonPrefix := findCommonPrefix(parts)
			if len(commonPrefix) >= 2 {
				e.searchBytes = []byte(commonPrefix)
				e.findRareByte()
				return
			}
			// If no common prefix, use the first alternative if it's literal
			firstPart := parts[0]
			if len(firstPart) >= 2 && !strings.ContainsAny(firstPart, ".*+?^$()[]{}\\") {
				e.searchBytes = []byte(firstPart)
				e.findRareByte()
				return
			}
		}
	} else {
		// Look for literal sequences in the pattern
		literals := extractSimpleLiterals(pattern)
		if len(literals) > 0 {
			// Choose the longest literal
			longest := literals[0]
			for _, lit := range literals[1:] {
				if len(lit) > len(longest) {
					longest = lit
				}
			}
			if len(longest) >= 2 {
				e.searchBytes = []byte(longest)
				e.findRareByte()
			}
		}
	}
}

// fastByteScan performs optimized byte scanning using SIMD when available
func (e *Engine) fastByteScan(data []byte, target byte) int {
	// Use SIMD engine for improved performance
	return e.optimizedEngine.FastIndexByte(data, target)
}

// optimizedLiteralSearch performs high-speed literal string search with SIMD
func (e *Engine) optimizedLiteralSearch(data []byte) []int {
	if len(e.searchBytes) == 0 {
		return nil
	}

	var matches []int
	searchLen := len(e.searchBytes)

	// For case-insensitive search, we need to work with lowercase data
	var searchData []byte
	if e.ignoreCase {
		searchData = bytes.ToLower(data)
	} else {
		searchData = data
	}

	if searchLen == 1 {
		// Single byte search using SIMD
		pos := 0
		for {
			idx := e.optimizedEngine.FastIndexByte(searchData[pos:], e.searchBytes[0])
			if idx == -1 {
				break
			}
			matches = append(matches, pos+idx)
			pos += idx + 1
		}
		return matches
	}

	// Multi-byte search using rare byte optimization with SIMD
	pos := 0
	for {
		// Find next occurrence of rare byte using SIMD
		idx := e.optimizedEngine.FastIndexByte(searchData[pos:], e.rareByte)
		if idx == -1 {
			break
		}

		candidatePos := pos + idx - e.rareByteIdx
		if candidatePos >= 0 && candidatePos+searchLen <= len(searchData) {
			// Check if full pattern matches
			if bytes.Equal(searchData[candidatePos:candidatePos+searchLen], e.searchBytes) {
				matches = append(matches, candidatePos)
			}
		}
		pos += idx + 1
	}

	return matches
}

// Search performs optimized search on a file
func (e *Engine) Search(ctx context.Context, filePath string) ([]Match, error) {
	atomic.AddInt64(&e.filesScanned, 1)

	// Check if file is compressed
	isCompressed, compressionType, err := e.compressionDetector.IsCompressed(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check compression: %w", err)
	}

	if isCompressed {
		return e.searchCompressedFile(ctx, filePath, compressionType)
	}

	return e.searchPlainFile(ctx, filePath)
}

// searchPlainFile performs search on uncompressed files
func (e *Engine) searchPlainFile(ctx context.Context, filePath string) ([]Match, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return e.searchFromReader(ctx, filePath, file)
}

// searchCompressedFile performs search on compressed files
func (e *Engine) searchCompressedFile(ctx context.Context, filePath string, compressionType CompressionType) ([]Match, error) {
	var results []Match

	err := e.streamDecompressor.ProcessCompressedFile(filePath, func(reader io.Reader, ct CompressionType) error {
		matches, err := e.searchFromReader(ctx, filePath, reader)
		if err != nil {
			return err
		}
		results = matches
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search compressed file: %w", err)
	}

	return results, nil
}

// searchFromReader performs the actual search logic on any io.Reader
func (e *Engine) searchFromReader(ctx context.Context, filePath string, reader io.Reader) ([]Match, error) {
	var results []Match
	var allLines []string

	// Read all lines first if we need context
	if e.contextLines > 0 {
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, e.bufferSize), e.bufferSize)
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		// For compressed files, we can't seek back, so we'll process from allLines
		for lineNum, line := range allLines {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			default:
			}

			lineBytes := []byte(line)
			atomic.AddInt64(&e.bytesScanned, int64(len(lineBytes)))

			matches := e.findMatches(lineBytes)
			for _, pos := range matches {
				atomic.AddInt64(&e.matchesFound, 1)
				result := Match{
					File:    filePath,
					Line:    lineNum + 1, // 1-indexed
					Content: line,
					Column:  pos + 1, // 1-indexed
				}

				// Add context lines
				result.Context = e.extractContextLines(allLines, lineNum, e.contextLines)
				results = append(results, result)
			}
		}
		return results, nil
	}

	// No context needed, process line by line
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, e.bufferSize), e.bufferSize)

	lineNum := 1
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		line := scanner.Bytes()
		atomic.AddInt64(&e.bytesScanned, int64(len(line)))

		matches := e.findMatches(line)
		for _, pos := range matches {
			atomic.AddInt64(&e.matchesFound, 1)
			result := Match{
				File:    filePath,
				Line:    lineNum,
				Content: string(line),
				Column:  pos + 1, // 1-indexed
			}
			results = append(results, result)
		}

		lineNum++
	}

	return results, scanner.Err()
}

// findMatches extracts the match finding logic
func (e *Engine) findMatches(line []byte) []int {
	var matches []int
	if e.isLiteral {
		matches = e.optimizedLiteralSearch(line)
	} else {
		// Use regex search
		regexMatches := e.regex.FindAllIndex(line, -1)
		for _, match := range regexMatches {
			matches = append(matches, match[0])
		}
	}
	return matches
}

// extractContextLines extracts context lines around a match
func (e *Engine) extractContextLines(allLines []string, matchLineIndex int, contextLines int) []string {
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

// GetStats returns performance statistics including SIMD and cache info
func (e *Engine) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"bytes_scanned": atomic.LoadInt64(&e.bytesScanned),
		"files_scanned": atomic.LoadInt64(&e.filesScanned),
		"matches_found": atomic.LoadInt64(&e.matchesFound),
		"is_literal":    e.isLiteral,
		"rare_byte":     fmt.Sprintf("0x%02x", e.rareByte),
		"worker_count":  e.workerCount,
		"buffer_size":   e.bufferSize,
	}

	// Add SIMD capabilities
	simdCaps := e.optimizedEngine.GetCapabilities()
	for key, value := range simdCaps {
		stats["simd_"+strings.ToLower(key)] = value
	}

	// Add DFA cache statistics
	cacheStats := e.dfaCache.Stats()
	stats["cache_size"] = cacheStats.Size
	stats["cache_hits"] = cacheStats.Hits
	stats["cache_misses"] = cacheStats.Misses
	stats["cache_hit_rate"] = cacheStats.HitRate
	stats["cache_evicted"] = cacheStats.Evicted

	// Add compression support information
	stats["compression_supported"] = true
	stats["compression_formats"] = e.compressionDetector.GetSupportedFormats()
	stats["compression_extensions"] = e.compressionDetector.GetSupportedExtensions()

	return stats
}

// GetAdvancedStats returns detailed performance statistics
func (e *Engine) GetAdvancedStats() AdvancedStats {
	return AdvancedStats{
		BasicStats:       e.GetStats(),
		SIMDCapabilities: e.optimizedEngine.GetCapabilities(),
		CacheStats:       e.dfaCache.Stats(),
		CachedPatterns:   e.dfaCache.GetCachedPatterns(),
	}
}

// AdvancedStats provides comprehensive performance statistics
type AdvancedStats struct {
	BasicStats       map[string]interface{} `json:"basic_stats"`
	SIMDCapabilities map[string]bool        `json:"simd_capabilities"`
	CacheStats       CacheStats             `json:"cache_stats"`
	CachedPatterns   []PatternInfo          `json:"cached_patterns"`
}

// Helper functions

func findCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	prefix := strs[0]
	for _, s := range strs[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if len(prefix) == 0 {
			break
		}
	}
	return prefix
}

func extractSimpleLiterals(pattern string) []string {
	var literals []string
	current := ""

	for i := 0; i < len(pattern); i++ {
		r := rune(pattern[i])
		switch r {
		case '.', '*', '+', '?', '^', '$', '|', '(', ')', '[', ']', '{', '}', '\\':
			if len(current) >= 2 {
				literals = append(literals, current)
			}
			current = ""
			// Skip escaped characters
			if r == '\\' && i+1 < len(pattern) {
				i++ // Skip next character
			}
		default:
			current += string(r)
		}
	}

	if len(current) >= 2 {
		literals = append(literals, current)
	}

	return literals
}
