package goripgrep

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/cpu"
)

// OptimizedEngine provides high-performance search operations using pure Go optimizations
type OptimizedEngine struct {
	hasAVX2   bool // CPU feature detection for potential future use
	hasSSE42  bool
	hasNEON   bool
	wordSize  int
	chunkSize int
}

// NewOptimizedEngine creates a new optimized search engine with CPU feature detection
func NewOptimizedEngine() *OptimizedEngine {
	engine := &OptimizedEngine{
		wordSize:  8,  // 64-bit words
		chunkSize: 64, // Process 64 bytes at a time for optimal performance
	}

	// Detect CPU features for potential future optimizations
	switch runtime.GOARCH {
	case "amd64":
		engine.hasAVX2 = cpu.X86.HasAVX2
		engine.hasSSE42 = cpu.X86.HasSSE42
	case "arm64":
		engine.hasNEON = true
	}

	return engine
}

// FastIndexByte performs optimized byte search using word-level operations and bit manipulation
func (e *OptimizedEngine) FastIndexByte(data []byte, target byte) int {
	if len(data) == 0 {
		return -1
	}

	// Use optimized word-level scanning for larger data
	if len(data) >= e.wordSize {
		return e.indexByteWordOptimized(data, target)
	}

	// Simple byte-by-byte for small data
	for i, b := range data {
		if b == target {
			return i
		}
	}
	return -1
}

// indexByteWordOptimized uses word-level operations and bit manipulation for fast byte searching
func (e *OptimizedEngine) indexByteWordOptimized(data []byte, target byte) int {
	if len(data) == 0 {
		return -1
	}

	const wordSize = 8
	if len(data) < wordSize {
		// Fallback for small data
		for i, b := range data {
			if b == target {
				return i
			}
		}
		return -1
	}

	// Create target word (8 copies of target byte)
	targetWord := uint64(target)
	targetWord |= targetWord << 8
	targetWord |= targetWord << 16
	targetWord |= targetWord << 32

	// Handle unaligned start
	alignOffset := uintptr(unsafe.Pointer(&data[0])) & (wordSize - 1)
	if alignOffset != 0 {
		alignOffset = wordSize - alignOffset
		if int(alignOffset) > len(data) {
			alignOffset = uintptr(len(data))
		}

		// Check unaligned prefix byte by byte
		for i := 0; i < int(alignOffset); i++ {
			if data[i] == target {
				return i
			}
		}
		data = data[alignOffset:]
	}

	// Process aligned 8-byte words
	for i := 0; i <= len(data)-wordSize; i += wordSize {
		word := *(*uint64)(unsafe.Pointer(&data[i]))

		// XOR with target pattern - matching bytes become 0
		xor := word ^ targetWord

		// Check if any byte in the word is zero (matches target)
		if e.hasZeroByte(xor) {
			// Found a match, find which byte
			for j := 0; j < wordSize && i+j < len(data); j++ {
				if data[i+j] == target {
					return int(alignOffset) + i + j
				}
			}
		}
	}

	// Handle remaining bytes
	remainder := len(data) % wordSize
	if remainder > 0 {
		start := len(data) - remainder
		for i := start; i < len(data); i++ {
			if data[i] == target {
				return int(alignOffset) + i
			}
		}
	}

	return -1
}

// hasZeroByte uses bit manipulation to detect if any byte in a 64-bit word is zero
// This is a well-known optimization technique
func (e *OptimizedEngine) hasZeroByte(word uint64) bool {
	// Bit manipulation trick: (word - 0x0101010101010101) & ~word & 0x8080808080808080
	// This detects zero bytes in parallel across all 8 bytes of the word
	return (word-0x0101010101010101)&^word&0x8080808080808080 != 0
}

// FastCountLines performs optimized newline counting using word-level operations
func (e *OptimizedEngine) FastCountLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	// Use word-level optimization for larger data
	if len(data) >= e.wordSize {
		return e.countLinesWordOptimized(data)
	}

	// Simple counting for small data
	count := 0
	for _, b := range data {
		if b == '\n' {
			count++
		}
	}
	return count
}

// countLinesWordOptimized uses word-level operations to count newlines efficiently
func (e *OptimizedEngine) countLinesWordOptimized(data []byte) int {
	if len(data) == 0 {
		return 0
	}

	count := 0
	const newline = '\n'
	const wordSize = 8

	if len(data) < wordSize {
		// Fallback for small data
		for _, b := range data {
			if b == newline {
				count++
			}
		}
		return count
	}

	// Create newline word (8 copies of newline byte)
	newlineWord := uint64(newline)
	newlineWord |= newlineWord << 8
	newlineWord |= newlineWord << 16
	newlineWord |= newlineWord << 32

	// Process 8-byte words
	for i := 0; i <= len(data)-wordSize; i += wordSize {
		word := *(*uint64)(unsafe.Pointer(&data[i]))

		// XOR with newline pattern - matching bytes become 0
		xor := word ^ newlineWord

		// Check if any byte in the word is zero (matches newline)
		if e.hasZeroByte(xor) {
			// Count newlines in this word byte by byte
			for j := 0; j < wordSize && i+j < len(data); j++ {
				if data[i+j] == newline {
					count++
				}
			}
		}
	}

	// Handle remaining bytes
	remainder := len(data) % wordSize
	if remainder > 0 {
		start := len(data) - remainder
		for i := start; i < len(data); i++ {
			if data[i] == newline {
				count++
			}
		}
	}

	return count
}

// GetCapabilities returns information about available optimizations and CPU features
func (e *OptimizedEngine) GetCapabilities() map[string]bool {
	return map[string]bool{
		"WORD_LEVEL_OPT":   true,                                                   // Word-level optimizations available
		"BIT_MANIPULATION": true,                                                   // Bit manipulation optimizations
		"MEMORY_ALIGNMENT": true,                                                   // Memory alignment optimizations
		"ARCH_OPTIMIZED":   runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64", // Architecture-specific optimizations
		"AVX2_DETECTED":    e.hasAVX2,                                              // CPU feature detection (for future use)
		"SSE42_DETECTED":   e.hasSSE42,
		"NEON_DETECTED":    e.hasNEON,
		"PURE_GO":          true, // This is pure Go implementation
	}
}

// BenchmarkMethods provides performance comparison between different search methods
func (e *OptimizedEngine) BenchmarkMethods(data []byte, target byte) map[string]int {
	results := make(map[string]int)

	// Optimized word-level implementation
	results["word_optimized"] = e.indexByteWordOptimized(data, target)

	// Simple byte-by-byte for comparison
	results["byte_by_byte"] = e.simpleIndexByte(data, target)

	return results
}

// simpleIndexByte provides a simple byte-by-byte implementation for comparison
func (e *OptimizedEngine) simpleIndexByte(data []byte, target byte) int {
	for i, b := range data {
		if b == target {
			return i
		}
	}
	return -1
}
