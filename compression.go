package goripgrep

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CompressionType represents the type of compression detected
type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionGzip
	CompressionBzip2
)

// String returns the string representation of the compression type
func (ct CompressionType) String() string {
	switch ct {
	case CompressionNone:
		return "none"
	case CompressionGzip:
		return "gzip"
	case CompressionBzip2:
		return "bzip2"
	default:
		return "unknown"
	}
}

// CompressionDetector provides methods to detect and handle compressed files
type CompressionDetector struct {
	// Magic bytes for different compression formats
	magicBytes map[CompressionType][]byte
	// File extensions for different compression formats
	extensions map[string]CompressionType
}

// NewCompressionDetector creates a new compression detector
func NewCompressionDetector() *CompressionDetector {
	detector := &CompressionDetector{
		magicBytes: make(map[CompressionType][]byte),
		extensions: make(map[string]CompressionType),
	}

	// Initialize magic bytes for different formats
	detector.magicBytes[CompressionGzip] = []byte{0x1f, 0x8b}
	detector.magicBytes[CompressionBzip2] = []byte{0x42, 0x5a, 0x68}

	// Initialize file extensions
	detector.extensions[".gz"] = CompressionGzip
	detector.extensions[".gzip"] = CompressionGzip
	detector.extensions[".bz2"] = CompressionBzip2
	detector.extensions[".bzip2"] = CompressionBzip2

	return detector
}

// DetectCompressionByExtension detects compression type based on file extension
func (cd *CompressionDetector) DetectCompressionByExtension(filename string) CompressionType {
	ext := strings.ToLower(filepath.Ext(filename))
	if compressionType, exists := cd.extensions[ext]; exists {
		return compressionType
	}
	return CompressionNone
}

// DetectCompressionByMagicBytes detects compression type by reading magic bytes from file
func (cd *CompressionDetector) DetectCompressionByMagicBytes(filename string) (CompressionType, error) {
	file, err := os.Open(filename)
	if err != nil {
		return CompressionNone, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	// Read first 16 bytes to check magic numbers
	buffer := make([]byte, 16)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return CompressionNone, fmt.Errorf("failed to read magic bytes from %s: %w", filename, err)
	}

	buffer = buffer[:n]
	return cd.DetectCompressionByBytes(buffer), nil
}

// DetectCompressionByBytes detects compression type from byte slice
func (cd *CompressionDetector) DetectCompressionByBytes(data []byte) CompressionType {
	for compressionType, magic := range cd.magicBytes {
		if len(data) >= len(magic) && bytes.HasPrefix(data, magic) {
			return compressionType
		}
	}
	return CompressionNone
}

// DetectCompression detects compression using both extension and magic bytes
func (cd *CompressionDetector) DetectCompression(filename string) (CompressionType, error) {
	// First try by extension (faster)
	if compressionType := cd.DetectCompressionByExtension(filename); compressionType != CompressionNone {
		// Verify with magic bytes if possible
		if magicType, err := cd.DetectCompressionByMagicBytes(filename); err == nil {
			if magicType == compressionType || magicType == CompressionNone {
				return compressionType, nil
			}
			// Magic bytes disagree with extension, trust magic bytes
			return magicType, nil
		} else {
			// If we can't read magic bytes due to file access error, return the error
			return CompressionNone, err
		}
	}

	// If extension doesn't indicate compression, check magic bytes
	return cd.DetectCompressionByMagicBytes(filename)
}

// IsCompressed checks if a file is compressed
func (cd *CompressionDetector) IsCompressed(filename string) (bool, CompressionType, error) {
	compressionType, err := cd.DetectCompression(filename)
	if err != nil {
		return false, CompressionNone, err
	}
	return compressionType != CompressionNone, compressionType, nil
}

// DecompressReader creates a decompressing reader for the given compression type
func (cd *CompressionDetector) DecompressReader(reader io.Reader, compressionType CompressionType) (io.Reader, error) {
	switch compressionType {
	case CompressionGzip:
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return gzipReader, nil

	case CompressionBzip2:
		// bzip2.NewReader doesn't return an error
		return bzip2.NewReader(reader), nil

	case CompressionNone:
		return reader, nil

	default:
		return nil, fmt.Errorf("unsupported compression type: %s", compressionType.String())
	}
}

// DecompressFile decompresses a file and returns the decompressed content
func (cd *CompressionDetector) DecompressFile(filename string) ([]byte, CompressionType, error) {
	compressionType, err := cd.DetectCompression(filename)
	if err != nil {
		return nil, CompressionNone, fmt.Errorf("failed to detect compression: %w", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, compressionType, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	if compressionType == CompressionNone {
		// File is not compressed, read directly
		content, err := io.ReadAll(file)
		return content, compressionType, err
	}

	// Decompress the file
	decompressedReader, err := cd.DecompressReader(file, compressionType)
	if err != nil {
		return nil, compressionType, fmt.Errorf("failed to create decompressed reader: %w", err)
	}

	// Handle gzip reader closing
	if gzipReader, ok := decompressedReader.(*gzip.Reader); ok {
		defer gzipReader.Close()
	}

	content, err := io.ReadAll(decompressedReader)
	if err != nil {
		return nil, compressionType, fmt.Errorf("failed to read decompressed content: %w", err)
	}

	return content, compressionType, nil
}

// GetSupportedFormats returns a list of supported compression formats
func (cd *CompressionDetector) GetSupportedFormats() []CompressionType {
	return []CompressionType{
		CompressionGzip,
		CompressionBzip2,
	}
}

// GetSupportedExtensions returns a map of supported file extensions
func (cd *CompressionDetector) GetSupportedExtensions() map[string]CompressionType {
	supported := make(map[string]CompressionType)
	supportedTypes := cd.GetSupportedFormats()

	for ext, compressionType := range cd.extensions {
		for _, supportedType := range supportedTypes {
			if compressionType == supportedType {
				supported[ext] = compressionType
				break
			}
		}
	}

	return supported
}

// StreamingDecompressor provides memory-efficient streaming decompression
type StreamingDecompressor struct {
	detector   *CompressionDetector
	bufferSize int
}

// NewStreamingDecompressor creates a new streaming decompressor
func NewStreamingDecompressor(bufferSize int) *StreamingDecompressor {
	if bufferSize <= 0 {
		bufferSize = 64 * 1024 // Default 64KB buffer
	}
	return &StreamingDecompressor{
		detector:   NewCompressionDetector(),
		bufferSize: bufferSize,
	}
}

// DecompressStream decompresses a file and provides a streaming reader
func (sd *StreamingDecompressor) DecompressStream(filename string) (io.ReadCloser, CompressionType, error) {
	compressionType, err := sd.detector.DetectCompression(filename)
	if err != nil {
		return nil, CompressionNone, fmt.Errorf("failed to detect compression: %w", err)
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, compressionType, fmt.Errorf("failed to open file: %w", err)
	}

	if compressionType == CompressionNone {
		// File is not compressed, return file directly
		return file, compressionType, nil
	}

	// Create decompressed reader
	decompressedReader, err := sd.detector.DecompressReader(file, compressionType)
	if err != nil {
		file.Close()
		return nil, compressionType, fmt.Errorf("failed to create decompressed reader: %w", err)
	}

	// Wrap in a composite closer that closes both the decompressed reader and the file
	return &compositeReadCloser{
		reader:     decompressedReader,
		closers:    []io.Closer{file},
		gzipReader: nil,
		closed:     false,
	}, compressionType, nil
}

// compositeReadCloser combines a reader with multiple closers
type compositeReadCloser struct {
	reader     io.Reader
	closers    []io.Closer
	gzipReader *gzip.Reader
	closed     bool
}

func (crc *compositeReadCloser) Read(p []byte) (n int, err error) {
	return crc.reader.Read(p)
}

func (crc *compositeReadCloser) Close() error {
	if crc.closed {
		return nil // Already closed, return nil to be idempotent
	}
	crc.closed = true

	var lastErr error

	// Close gzip reader if present
	if gzipReader, ok := crc.reader.(*gzip.Reader); ok {
		if err := gzipReader.Close(); err != nil {
			lastErr = err
		}
	}

	// Close all other closers
	for _, closer := range crc.closers {
		if err := closer.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// ProcessCompressedFile processes a compressed file with a callback function
func (sd *StreamingDecompressor) ProcessCompressedFile(filename string, processor func(io.Reader, CompressionType) error) error {
	stream, compressionType, err := sd.DecompressStream(filename)
	if err != nil {
		return fmt.Errorf("failed to create decompression stream: %w", err)
	}
	defer stream.Close()

	return processor(stream, compressionType)
}
