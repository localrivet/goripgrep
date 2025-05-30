// Package main demonstrates performance benchmarking capabilities of GoRipGrep.
//
// This example shows how to measure and compare performance across different
// search configurations, file sizes, and patterns to optimize search operations.
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Performance Benchmarking ===")

	// Display system information
	displaySystemInfo()

	// Create test data for benchmarking
	if err := createBenchmarkData(); err != nil {
		log.Fatal(err)
	}
	defer cleanupBenchmarkData()

	// Run various performance benchmarks
	runSIMDBenchmarks()
	runDFACacheBenchmarks()
	runEngineOptimizationBenchmarks()
	runComparisonBenchmarks()
	runOptimizedSearchComparison()
}

func runSIMDBenchmarks() {
	fmt.Println("\n=== Pure Go Optimization Benchmarks ===")

	optimized := goripgrep.NewOptimizedEngine()

	// Display capabilities
	caps := optimized.GetCapabilities()
	fmt.Printf("Optimization Capabilities: %+v\n", caps)

	// Test data of various sizes
	testSizes := []int{1024, 8192, 65536}

	for _, size := range testSizes {
		fmt.Printf("\n--- Testing with %d bytes ---\n", size)

		// Create test data
		data := make([]byte, size)
		for i := range data {
			data[i] = byte('a' + (i % 26))
		}
		// Add target at 75% position
		target := byte('z')
		data[size*3/4] = target

		// Benchmark FastIndexByte
		start := time.Now()
		iterations := 100000
		for i := 0; i < iterations; i++ {
			optimized.FastIndexByte(data, target)
		}
		duration := time.Since(start)

		throughput := float64(size*iterations) / duration.Seconds() / (1024 * 1024)
		fmt.Printf("FastIndexByte: %.2f MB/s (%d iterations)\n", throughput, iterations)

		// Benchmark FastCountLines
		dataWithNewlines := bytes.Replace(data, []byte("abcde"), []byte("ab\nde"), -1)
		start = time.Now()
		for i := 0; i < iterations; i++ {
			optimized.FastCountLines(dataWithNewlines)
		}
		duration = time.Since(start)

		throughput = float64(len(dataWithNewlines)*iterations) / duration.Seconds() / (1024 * 1024)
		fmt.Printf("FastCountLines: %.2f MB/s (%d iterations)\n", throughput, iterations)

		// Compare methods
		results := optimized.BenchmarkMethods(data, target)
		fmt.Printf("Method comparison - Word optimized: %d, Byte-by-byte: %d\n",
			results["word_optimized"], results["byte_by_byte"])
	}
}

func runDFACacheBenchmarks() {
	fmt.Println("\n2. DFA Cache Performance Benchmarks:")

	patterns := []string{
		"test.*pattern",
		"hello.*world",
		"[a-z]+@[a-z]+\\.[a-z]+",
		"\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}",
		"function\\s+\\w+\\s*\\(",
	}

	// Benchmark cached vs uncached regex compilation
	fmt.Println("   Regex Compilation Performance:")

	// Warm up cache
	for _, pattern := range patterns {
		_, _ = goripgrep.CompileWithCache(pattern, false)
	}

	// Benchmark cached compilation
	start := time.Now()
	for i := 0; i < 1000; i++ {
		for _, pattern := range patterns {
			_, _ = goripgrep.CompileWithCache(pattern, false)
		}
	}
	cachedTime := time.Since(start)

	// Benchmark standard compilation
	start = time.Now()
	for i := 0; i < 1000; i++ {
		for _, pattern := range patterns {
			_, _ = regexp.Compile(pattern)
		}
	}
	stdTime := time.Since(start)

	speedup := float64(stdTime) / float64(cachedTime)
	fmt.Printf("   Cached=%v, Standard=%v, Speedup=%.2fx\n",
		cachedTime, stdTime, speedup)

	// Display cache statistics
	cache := goripgrep.GetGlobalDFACache()
	stats := cache.Stats()
	fmt.Printf("   Cache Stats: %s\n", stats.String())
}

func runEngineOptimizationBenchmarks() {
	fmt.Println("\n=== Engine Optimization Benchmarks ===")

	optimized := goripgrep.NewOptimizedEngine()

	// Display optimization capabilities
	caps := optimized.GetCapabilities()
	fmt.Printf("  Optimization Support: %+v\n", caps)
}

func runComparisonBenchmarks() {
	fmt.Println("\n4. Comparison with Standard Tools:")

	testFile := "benchmark_data/large_file.txt"
	pattern := "function"

	// GoRipGrep search
	args := goripgrep.SearchArgs{
		Pattern: pattern,
	}

	engine, err := goripgrep.NewEngine(args)
	if err != nil {
		log.Printf("Failed to create engine: %v", err)
		return
	}

	ctx := context.Background()
	start := time.Now()
	results, err := engine.Search(ctx, testFile)
	if err != nil {
		log.Printf("GoRipGrep search failed: %v", err)
		return
	}
	goripgrepTime := time.Since(start)

	fmt.Printf("   GoRipGrep: %v, matches=%d\n", goripgrepTime, len(results))

	// Display final statistics
	stats := engine.GetStats()
	fmt.Printf("   Performance Summary:\n")
	fmt.Printf("     Throughput: %.2f MB/s\n",
		float64(stats["bytes_scanned"].(int64))/1024/1024/goripgrepTime.Seconds())
	fmt.Printf("     SIMD acceleration: %v\n", stats["simd_simd"])
	fmt.Printf("     Cache efficiency: %.2f%%\n", stats["cache_hit_rate"].(float64)*100)
}

func createBenchmarkData() error {
	fmt.Println("Creating benchmark data...")

	// Create benchmark directory
	if err := os.MkdirAll("benchmark_data", 0755); err != nil {
		return fmt.Errorf("failed to create benchmark directory: %w", err)
	}

	// Generate large test file with various patterns
	content := generateTestContent(100000) // 100k lines

	if err := os.WriteFile("benchmark_data/large_file.txt", []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create large test file: %w", err)
	}

	// Create multiple smaller files
	for i := range 10 {
		filename := fmt.Sprintf("benchmark_data/file_%d.txt", i)
		smallContent := generateTestContent(1000) // 1k lines each
		if err := os.WriteFile(filename, []byte(smallContent), 0644); err != nil {
			return fmt.Errorf("failed to create test file %s: %w", filename, err)
		}
	}

	fmt.Println("Benchmark data created successfully")
	return nil
}

func generateTestContent(lines int) string {
	var builder strings.Builder

	patterns := []string{
		"function main() {",
		"var result = process();",
		"if (condition) {",
		"for (int i = 0; i < count; i++) {",
		"class MyClass {",
		"public void method() {",
		"// This is a comment",
		"import java.util.*;",
		"package com.example;",
		"return value;",
	}

	for i := range lines {
		pattern := patterns[i%len(patterns)]
		line := fmt.Sprintf("Line %d: %s\n", i+1, pattern)
		builder.WriteString(line)
	}

	return builder.String()
}

func cleanupBenchmarkData() {
	fmt.Println("Cleaning up benchmark data...")
	if err := os.RemoveAll("benchmark_data"); err != nil {
		log.Printf("Warning: failed to cleanup benchmark data: %v", err)
	}
}

func displaySystemInfo() {
	fmt.Printf("System Information:\n")
	fmt.Printf("  OS: %s\n", runtime.GOOS)
	fmt.Printf("  Architecture: %s\n", runtime.GOARCH)
	fmt.Printf("  CPUs: %d\n", runtime.NumCPU())
	fmt.Printf("  Go Version: %s\n", runtime.Version())

	// Display optimization capabilities
	optimized := goripgrep.NewOptimizedEngine()
	caps := optimized.GetCapabilities()
	fmt.Printf("  Optimization Support: %+v\n", caps)
}

func runOptimizedSearchComparison() {
	fmt.Println("\n=== Optimized vs Standard Search Comparison ===")

	optimized := goripgrep.NewOptimizedEngine()

	// Test with different data patterns
	testCases := []struct {
		name   string
		data   []byte
		target byte
	}{
		{"Small ASCII", []byte("The quick brown fox jumps over the lazy dog"), 'o'},
		{"Large ASCII", bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 1000), 'z'},
		{"With Unicode", []byte("Hello 世界! Testing unicode search capabilities."), '!'},
		{"Binary-like", make([]byte, 10000), 0xFF},
	}

	// Initialize binary-like data
	for i := range testCases[3].data {
		testCases[3].data[i] = byte(i % 256)
	}
	testCases[3].data[7500] = 0xFF // Add target

	for _, tc := range testCases {
		fmt.Printf("\n--- %s (%d bytes) ---\n", tc.name, len(tc.data))

		// Optimized search
		start := time.Now()
		iterations := 10000
		var optimizedResult int
		for i := 0; i < iterations; i++ {
			optimizedResult = optimized.FastIndexByte(tc.data, tc.target)
		}
		optimizedDuration := time.Since(start)

		// Standard search
		start = time.Now()
		var standardResult int
		for i := 0; i < iterations; i++ {
			standardResult = bytes.IndexByte(tc.data, tc.target)
		}
		standardDuration := time.Since(start)

		// Results should match
		if optimizedResult != standardResult {
			fmt.Printf("ERROR: Results don't match! Optimized: %d, Standard: %d\n",
				optimizedResult, standardResult)
		} else {
			fmt.Printf("Target found at position: %d\n", optimizedResult)
		}

		// Performance comparison
		speedup := float64(standardDuration) / float64(optimizedDuration)
		fmt.Printf("Optimized: %v, Standard: %v, Speedup: %.2fx\n",
			optimizedDuration, standardDuration, speedup)

		optimizedThroughput := float64(len(tc.data)*iterations) / optimizedDuration.Seconds() / (1024 * 1024)
		standardThroughput := float64(len(tc.data)*iterations) / standardDuration.Seconds() / (1024 * 1024)
		fmt.Printf("Throughput - Optimized: %.2f MB/s, Standard: %.2f MB/s\n",
			optimizedThroughput, standardThroughput)
	}
}
