package goripgrep

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestCompressionDetector(t *testing.T) {
	detector := NewCompressionDetector()

	t.Run("DetectCompressionByExtension", func(t *testing.T) {
		testCases := []struct {
			filename string
			expected CompressionType
		}{
			{"test.txt", CompressionNone},
			{"test.gz", CompressionGzip},
			{"test.gzip", CompressionGzip},
			{"test.bz2", CompressionBzip2},
			{"test.bzip2", CompressionBzip2},
			{"test.GZ", CompressionGzip}, // Test case insensitive
			{"test.BZ2", CompressionBzip2},
			// Unsupported formats should return CompressionNone
			{"test.xz", CompressionNone},
			{"test.lzma", CompressionNone},
			{"test.lz4", CompressionNone},
			{"test.zst", CompressionNone},
		}

		for _, tc := range testCases {
			result := detector.DetectCompressionByExtension(tc.filename)
			if result != tc.expected {
				t.Errorf("DetectCompressionByExtension(%s) = %v, expected %v",
					tc.filename, result, tc.expected)
			}
		}
	})

	t.Run("DetectCompressionByBytes", func(t *testing.T) {
		testCases := []struct {
			name     string
			data     []byte
			expected CompressionType
		}{
			{"Empty", []byte{}, CompressionNone},
			{"Plain text", []byte("Hello, World!"), CompressionNone},
			{"Gzip magic", []byte{0x1f, 0x8b, 0x08, 0x00}, CompressionGzip},
			{"Bzip2 magic", []byte{0x42, 0x5a, 0x68, 0x39}, CompressionBzip2},
			// Unsupported formats should return CompressionNone
			{"XZ magic", []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00}, CompressionNone},
			{"LZMA magic", []byte{0x5d, 0x00, 0x00, 0x80, 0x00}, CompressionNone},
			{"LZ4 magic", []byte{0x04, 0x22, 0x4d, 0x18}, CompressionNone},
			{"Zstd magic", []byte{0x28, 0xb5, 0x2f, 0xfd}, CompressionNone},
		}

		for _, tc := range testCases {
			result := detector.DetectCompressionByBytes(tc.data)
			if result != tc.expected {
				t.Errorf("DetectCompressionByBytes(%s) = %v, expected %v",
					tc.name, result, tc.expected)
			}
		}
	})

	t.Run("GetSupportedFormats", func(t *testing.T) {
		formats := detector.GetSupportedFormats()
		expectedFormats := []CompressionType{CompressionGzip, CompressionBzip2}

		if len(formats) != len(expectedFormats) {
			t.Errorf("GetSupportedFormats() returned %d formats, expected %d",
				len(formats), len(expectedFormats))
		}

		for _, expected := range expectedFormats {
			found := false
			for _, format := range formats {
				if format == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GetSupportedFormats() missing expected format: %v", expected)
			}
		}
	})

	t.Run("GetSupportedExtensions", func(t *testing.T) {
		extensions := detector.GetSupportedExtensions()
		expectedExtensions := map[string]CompressionType{
			".gz":    CompressionGzip,
			".gzip":  CompressionGzip,
			".bz2":   CompressionBzip2,
			".bzip2": CompressionBzip2,
		}

		for ext, expectedType := range expectedExtensions {
			if actualType, exists := extensions[ext]; !exists {
				t.Errorf("GetSupportedExtensions() missing extension: %s", ext)
			} else if actualType != expectedType {
				t.Errorf("GetSupportedExtensions()[%s] = %v, expected %v",
					ext, actualType, expectedType)
			}
		}
	})
}

func TestCompressionTypeString(t *testing.T) {
	testCases := []struct {
		compressionType CompressionType
		expected        string
	}{
		{CompressionNone, "none"},
		{CompressionGzip, "gzip"},
		{CompressionBzip2, "bzip2"},
		{CompressionType(999), "unknown"},
	}

	for _, tc := range testCases {
		result := tc.compressionType.String()
		if result != tc.expected {
			t.Errorf("CompressionType(%d).String() = %s, expected %s",
				int(tc.compressionType), result, tc.expected)
		}
	}
}

func TestDecompression(t *testing.T) {
	detector := NewCompressionDetector()
	testContent := "Hello, World!\nThis is a test file for compression.\nLine 3\nLine 4\n"

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("GzipDecompression", func(t *testing.T) {
		// Create gzip compressed file
		gzipFile := filepath.Join(tempDir, "test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test decompression
		decompressed, compressionType, err := detector.DecompressFile(gzipFile)
		if err != nil {
			t.Fatalf("Failed to decompress gzip file: %v", err)
		}

		if compressionType != CompressionGzip {
			t.Errorf("Expected compression type %v, got %v", CompressionGzip, compressionType)
		}

		if string(decompressed) != testContent {
			t.Errorf("Decompressed content doesn't match original.\nExpected: %q\nGot: %q",
				testContent, string(decompressed))
		}
	})

	t.Run("Bzip2Decompression", func(t *testing.T) {
		// For bzip2, we'll create a test file with known magic bytes
		// Since Go's bzip2 package only provides a reader, we'll create a simple test
		bzip2File := filepath.Join(tempDir, "test.bz2")

		// Create a minimal bzip2 file manually for testing
		// This is a valid bzip2 file containing "Hello"
		bzip2Data := []byte{
			0x42, 0x5a, 0x68, 0x39, 0x31, 0x41, 0x59, 0x26, 0x53, 0x59,
			0x65, 0x8b, 0xb9, 0x29, 0x00, 0x00, 0x02, 0x44, 0x00, 0x00,
			0x10, 0x02, 0x00, 0x0c, 0x00, 0x20, 0x00, 0x31, 0x4c, 0x59,
			0x0e, 0x17, 0x72, 0x45, 0x38, 0x50, 0x90, 0x65, 0x8b, 0xb9, 0x29,
		}

		err := os.WriteFile(bzip2File, bzip2Data, 0644)
		if err != nil {
			t.Fatalf("Failed to create bzip2 test file: %v", err)
		}

		// Test compression detection
		compressionType, err := detector.DetectCompression(bzip2File)
		if err != nil {
			t.Fatalf("Failed to detect bzip2 compression: %v", err)
		}

		if compressionType != CompressionBzip2 {
			t.Errorf("Expected compression type %v, got %v", CompressionBzip2, compressionType)
		}

		// Test that we can create a bzip2 reader (actual decompression depends on valid data)
		file, err := os.Open(bzip2File)
		if err != nil {
			t.Fatalf("Failed to open bzip2 file: %v", err)
		}
		defer file.Close()

		reader, err := detector.DecompressReader(file, CompressionBzip2)
		if err != nil {
			t.Fatalf("Failed to create bzip2 reader: %v", err)
		}

		// Verify it's a valid reader (bzip2.Reader is not a public type)
		if reader == nil {
			t.Error("Expected valid reader, got nil")
		}
	})

	t.Run("PlainFileDecompression", func(t *testing.T) {
		// Create plain text file
		plainFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(plainFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create plain test file: %v", err)
		}

		// Test decompression (should return content as-is)
		decompressed, compressionType, err := detector.DecompressFile(plainFile)
		if err != nil {
			t.Fatalf("Failed to read plain file: %v", err)
		}

		if compressionType != CompressionNone {
			t.Errorf("Expected compression type %v, got %v", CompressionNone, compressionType)
		}

		if string(decompressed) != testContent {
			t.Errorf("Plain file content doesn't match original.\nExpected: %q\nGot: %q",
				testContent, string(decompressed))
		}
	})

	t.Run("UnsupportedCompressionTypes", func(t *testing.T) {
		// Test with an invalid compression type
		reader := strings.NewReader("test data")
		_, err := detector.DecompressReader(reader, CompressionType(999))
		if err == nil {
			t.Error("Expected error for unsupported compression type")
		}
		if !strings.Contains(err.Error(), "unsupported compression type") {
			t.Errorf("Expected 'unsupported compression type' error, got: %v", err)
		}
	})

	t.Run("IsCompressed", func(t *testing.T) {
		// Test with gzip file
		gzipFile := filepath.Join(tempDir, "test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		gzipWriter.Write([]byte("test"))
		gzipWriter.Close()
		file.Close()

		isCompressed, compressionType, err := detector.IsCompressed(gzipFile)
		if err != nil {
			t.Fatalf("IsCompressed failed: %v", err)
		}

		if !isCompressed {
			t.Error("Expected file to be detected as compressed")
		}

		if compressionType != CompressionGzip {
			t.Errorf("Expected compression type %v, got %v", CompressionGzip, compressionType)
		}

		// Test with plain file
		plainFile := filepath.Join(tempDir, "plain.txt")
		err = os.WriteFile(plainFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create plain test file: %v", err)
		}

		isCompressed, compressionType, err = detector.IsCompressed(plainFile)
		if err != nil {
			t.Fatalf("IsCompressed failed: %v", err)
		}

		if isCompressed {
			t.Error("Expected file to be detected as not compressed")
		}

		if compressionType != CompressionNone {
			t.Errorf("Expected compression type %v, got %v", CompressionNone, compressionType)
		}
	})
}

func TestDecompressReaderErrors(t *testing.T) {
	detector := NewCompressionDetector()

	t.Run("InvalidGzipData", func(t *testing.T) {
		invalidData := bytes.NewReader([]byte("invalid gzip data"))
		_, err := detector.DecompressReader(invalidData, CompressionGzip)
		if err == nil {
			t.Error("Expected error for invalid gzip data")
		}
	})

	t.Run("UnknownCompressionType", func(t *testing.T) {
		reader := strings.NewReader("test data")
		_, err := detector.DecompressReader(reader, CompressionType(999))
		if err == nil {
			t.Error("Expected error for unknown compression type")
		}
		if !strings.Contains(err.Error(), "unsupported compression type") {
			t.Errorf("Expected 'unsupported compression type' error, got: %v", err)
		}
	})
}

func TestStreamingDecompression(t *testing.T) {
	testContent := "Hello, World!\nThis is a test file for streaming compression.\nLine 3\nLine 4\n"

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "streaming_compression_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("StreamingGzipDecompression", func(t *testing.T) {
		// Create gzip compressed file
		gzipFile := filepath.Join(tempDir, "stream_test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test streaming decompression
		decompressor := NewStreamingDecompressor(1024)
		stream, compressionType, err := decompressor.DecompressStream(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create decompression stream: %v", err)
		}
		defer stream.Close()

		if compressionType != CompressionGzip {
			t.Errorf("Expected compression type %v, got %v", CompressionGzip, compressionType)
		}

		// Read decompressed content
		decompressed, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("Failed to read decompressed stream: %v", err)
		}

		if string(decompressed) != testContent {
			t.Errorf("Decompressed content doesn't match original.\nExpected: %q\nGot: %q",
				testContent, string(decompressed))
		}
	})

	t.Run("StreamingPlainFile", func(t *testing.T) {
		// Create plain text file
		plainFile := filepath.Join(tempDir, "stream_plain.txt")
		err := os.WriteFile(plainFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create plain test file: %v", err)
		}

		// Test streaming (should pass through)
		decompressor := NewStreamingDecompressor(1024)
		stream, compressionType, err := decompressor.DecompressStream(plainFile)
		if err != nil {
			t.Fatalf("Failed to create stream: %v", err)
		}
		defer stream.Close()

		if compressionType != CompressionNone {
			t.Errorf("Expected compression type %v, got %v", CompressionNone, compressionType)
		}

		// Read content
		content, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("Failed to read stream: %v", err)
		}

		if string(content) != testContent {
			t.Errorf("Content doesn't match original.\nExpected: %q\nGot: %q",
				testContent, string(content))
		}
	})

	t.Run("ProcessCompressedFile", func(t *testing.T) {
		// Create gzip compressed file
		gzipFile := filepath.Join(tempDir, "process_test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test processing with callback
		decompressor := NewStreamingDecompressor(1024)
		var processedContent string
		var processedType CompressionType

		err = decompressor.ProcessCompressedFile(gzipFile, func(reader io.Reader, compressionType CompressionType) error {
			content, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			processedContent = string(content)
			processedType = compressionType
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to process compressed file: %v", err)
		}

		if processedType != CompressionGzip {
			t.Errorf("Expected compression type %v, got %v", CompressionGzip, processedType)
		}

		if processedContent != testContent {
			t.Errorf("Processed content doesn't match original.\nExpected: %q\nGot: %q",
				testContent, processedContent)
		}
	})

	t.Run("DefaultBufferSize", func(t *testing.T) {
		decompressor := NewStreamingDecompressor(0) // Should use default
		if decompressor.bufferSize != 64*1024 {
			t.Errorf("Expected default buffer size %d, got %d", 64*1024, decompressor.bufferSize)
		}

		decompressor = NewStreamingDecompressor(-1) // Should use default
		if decompressor.bufferSize != 64*1024 {
			t.Errorf("Expected default buffer size %d, got %d", 64*1024, decompressor.bufferSize)
		}

		decompressor = NewStreamingDecompressor(1024) // Should use specified
		if decompressor.bufferSize != 1024 {
			t.Errorf("Expected buffer size %d, got %d", 1024, decompressor.bufferSize)
		}
	})
}

func TestErrorHandlingAndResourceManagement(t *testing.T) {
	detector := NewCompressionDetector()

	t.Run("NonExistentFile", func(t *testing.T) {
		_, _, err := detector.IsCompressed("non_existent_file.gz")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("CorruptedGzipFile", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "error_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a file with .gz extension but invalid gzip content
		corruptFile := filepath.Join(tempDir, "corrupt.gz")
		err = os.WriteFile(corruptFile, []byte("This is not gzip data"), 0644)
		if err != nil {
			t.Fatalf("Failed to create corrupt file: %v", err)
		}

		// Should detect as gzip by extension but fail on decompression
		compressionType := detector.DetectCompressionByExtension(corruptFile)
		if compressionType != CompressionGzip {
			t.Errorf("Expected CompressionGzip, got %v", compressionType)
		}

		// Decompression should fail gracefully
		_, _, err = detector.DecompressFile(corruptFile)
		if err == nil {
			t.Error("Expected error for corrupted gzip file")
		}
	})

	t.Run("EmptyFile", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "empty_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create empty file
		emptyFile := filepath.Join(tempDir, "empty.txt")
		err = os.WriteFile(emptyFile, []byte{}, 0644)
		if err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		// Should handle empty files gracefully
		content, compressionType, err := detector.DecompressFile(emptyFile)
		if err != nil {
			t.Fatalf("Failed to handle empty file: %v", err)
		}

		if compressionType != CompressionNone {
			t.Errorf("Expected CompressionNone, got %v", compressionType)
		}

		if len(content) != 0 {
			t.Errorf("Expected empty content, got %d bytes", len(content))
		}
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		// This test is platform-specific and may not work on all systems
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "permission_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a file and remove read permissions
		restrictedFile := filepath.Join(tempDir, "restricted.gz")
		err = os.WriteFile(restrictedFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create restricted file: %v", err)
		}

		// Remove read permissions
		err = os.Chmod(restrictedFile, 0000)
		if err != nil {
			t.Fatalf("Failed to change file permissions: %v", err)
		}

		// Restore permissions for cleanup
		defer os.Chmod(restrictedFile, 0644)

		// Should handle permission errors gracefully
		_, _, err = detector.DecompressFile(restrictedFile)
		if err == nil {
			t.Error("Expected permission error")
		}
	})

	t.Run("StreamingResourceManagement", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "resource_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create gzip file
		gzipFile := filepath.Join(tempDir, "resource_test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		testContent := "Resource management test content"
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test that resources are properly closed
		decompressor := NewStreamingDecompressor(1024)
		stream, compressionType, err := decompressor.DecompressStream(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create decompression stream: %v", err)
		}

		if compressionType != CompressionGzip {
			t.Errorf("Expected CompressionGzip, got %v", compressionType)
		}

		// Read some data
		buffer := make([]byte, 10)
		_, err = stream.Read(buffer)
		if err != nil && err != io.EOF {
			t.Fatalf("Failed to read from stream: %v", err)
		}

		// Close should not return error
		err = stream.Close()
		if err != nil {
			t.Errorf("Stream close returned error: %v", err)
		}

		// Second close should be safe
		err = stream.Close()
		if err != nil {
			t.Errorf("Second stream close returned error: %v", err)
		}
	})

	t.Run("LargeFileHandling", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "large_file_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create a moderately large file (1MB of data)
		largeContent := strings.Repeat("This is a line of test data for large file handling.\n", 20000)

		// Create gzip file
		gzipFile := filepath.Join(tempDir, "large_test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create large gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		_, err = gzipWriter.Write([]byte(largeContent))
		if err != nil {
			t.Fatalf("Failed to write to large gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test streaming decompression with small buffer
		decompressor := NewStreamingDecompressor(1024) // Small buffer to test streaming

		var totalRead int64
		err = decompressor.ProcessCompressedFile(gzipFile, func(reader io.Reader, compressionType CompressionType) error {
			buffer := make([]byte, 512)
			for {
				n, err := reader.Read(buffer)
				totalRead += int64(n)
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to process large compressed file: %v", err)
		}

		if totalRead != int64(len(largeContent)) {
			t.Errorf("Expected to read %d bytes, got %d", len(largeContent), totalRead)
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		// Create temporary directory
		tempDir, err := os.MkdirTemp("", "concurrent_test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create gzip file
		gzipFile := filepath.Join(tempDir, "concurrent_test.gz")
		file, err := os.Create(gzipFile)
		if err != nil {
			t.Fatalf("Failed to create gzip test file: %v", err)
		}

		gzipWriter := gzip.NewWriter(file)
		testContent := "Concurrent access test content"
		_, err = gzipWriter.Write([]byte(testContent))
		if err != nil {
			t.Fatalf("Failed to write to gzip file: %v", err)
		}
		gzipWriter.Close()
		file.Close()

		// Test concurrent access to the same file
		const numGoroutines = 10
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				decompressor := NewStreamingDecompressor(1024)
				_, _, err := decompressor.DecompressStream(gzipFile)
				if err != nil {
					errors <- err
					return
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		for err := range errors {
			t.Errorf("Concurrent access error: %v", err)
		}
	})
}
