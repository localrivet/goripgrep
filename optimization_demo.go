package goripgrep

import (
	"fmt"
	"log"
	"time"
)

func DemoOptimizations() {
	pattern := `\w+Sushi`
	searchPath := "."

	fmt.Println("=== GoRipGrep: One Way of Doing Things Demo ===")
	fmt.Println()

	// 1. Basic search (default optimizations)
	fmt.Println("1. Basic search with default optimizations:")
	start := time.Now()
	results, err := Find(pattern, searchPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d matches in %v\n", results.Count(), time.Since(start))
	fmt.Println()

	// 2. Maximum performance mode (all optimizations)
	fmt.Println("2. Performance mode with all optimizations:")
	start = time.Now()
	results, err = Find(pattern, searchPath,
		WithPerformanceMode(),
		WithRecursive(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d matches in %v\n", results.Count(), time.Since(start))
	fmt.Println()

	// 3. Custom optimization combination
	fmt.Println("3. Custom optimization combination:")
	start = time.Now()
	results, err = Find(pattern, searchPath,
		WithLiteralStringOptimization(), // Fast literal string search
		WithMemoryPooling(),             // Reduce allocations
		WithLargeFileBuffers(),          // Larger I/O buffers
		WithRecursive(true),
		WithWorkers(8), // More concurrent workers
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d matches in %v\n", results.Count(), time.Since(start))
	fmt.Println()

	// 4. Show the ONE WAY principle
	fmt.Println("4. The 'One Way of Doing Things' Principle:")
	fmt.Println("   ✅ ONE Find() function with options")
	fmt.Println("   ✅ Options compose to create behavior")
	fmt.Println("   ✅ No separate SimplifiedFind(), QuickFind(), etc.")
	fmt.Println("   ✅ Add options to change behavior, not new functions")
	fmt.Println()

	// 5. Available optimization options
	fmt.Println("5. Available optimization options:")
	fmt.Println("   • WithPerformanceMode()          - Enable all optimizations")
	fmt.Println("   • WithLiteralStringOptimization() - Fast string search for literals")
	fmt.Println("   • WithMemoryPooling()            - Object pooling to reduce allocations")
	fmt.Println("   • WithLargeFileBuffers()         - Larger I/O buffers")
	fmt.Println("   • WithRegexCaching()             - Cache compiled regex patterns")
	fmt.Println("   • WithFastFileFiltering()        - Fast binary file detection")
	fmt.Println("   • WithOptimizedWalking()         - Use filepath.WalkDir")
	fmt.Println("   • WithEarlyBinaryDetection()     - Optimized binary detection")
}
