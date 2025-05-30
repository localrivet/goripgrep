// Package main demonstrates regex pattern capabilities of GoRipGrep.
//
// This example shows how to use various regex patterns for complex
// text search scenarios, including lookaheads, character classes, and more.
package main

import (
	"fmt"
	"log"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Regex Pattern Examples ===")

	// Example 1: Basic regex patterns
	fmt.Println("1. Basic Regex Patterns:")
	patterns := map[string]string{
		`func\s+\w+`:       "Function declarations",
		`\b[A-Z][a-z]+\b`:  "Capitalized words",
		`\d{1,3}\.\d{1,3}`: "Version numbers (partial)",
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`: "Email addresses",
	}

	for pattern, description := range patterns {
		fmt.Printf("  %s:\n", description)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Pattern: %s\n", pattern)
		fmt.Printf("    Found %d matches\n", results.Count())

		// Show first match as example
		if results.HasMatches() {
			match := results.Matches[0]
			fmt.Printf("    Example: %s:%d: %s\n", match.File, match.Line, match.Content)
		}
		fmt.Println()
	}

	// Example 2: Character classes and Unicode
	fmt.Println("2. Character Classes and Unicode:")
	unicodePatterns := map[string]string{
		`\p{L}+`:        "Unicode letters",
		`\p{N}+`:        "Unicode numbers",
		`\p{P}+`:        "Unicode punctuation",
		`[^\x00-\x7F]+`: "Non-ASCII characters",
		`[\p{Greek}]+`:  "Greek characters",
		`[\p{Han}]+`:    "Chinese characters",
	}

	for pattern, description := range unicodePatterns {
		fmt.Printf("  %s:\n", description)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Pattern: %s\n", pattern)
		fmt.Printf("    Found %d matches\n", results.Count())
		fmt.Println()
	}

	// Example 3: Anchors and boundaries
	fmt.Println("3. Anchors and Boundaries:")
	anchorPatterns := map[string]string{
		`^package\s+\w+`: "Lines starting with 'package'",
		`\bfunc\b`:       "Word 'func' with boundaries",
		`\w+$`:           "Words at end of line",
		`^\s*//`:         "Comment lines",
		`^import\s*\(`:   "Import blocks",
	}

	for pattern, description := range anchorPatterns {
		fmt.Printf("  %s:\n", description)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Pattern: %s\n", pattern)
		fmt.Printf("    Found %d matches\n", results.Count())
		fmt.Println()
	}

	// Example 4: Quantifiers and repetition
	fmt.Println("4. Quantifiers and Repetition:")
	quantifierPatterns := map[string]string{
		`\w{3,}`:     "Words with 3+ characters",
		`\d+`:        "One or more digits",
		`\w*Test\w*`: "Words containing 'Test'",
		`go{2,}`:     "'go' with 2+ o's",
		`\w+?`:       "Non-greedy word matching",
	}

	for pattern, description := range quantifierPatterns {
		fmt.Printf("  %s:\n", description)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Pattern: %s\n", pattern)
		fmt.Printf("    Found %d matches\n", results.Count())
		fmt.Println()
	}

	// Example 5: Groups and alternation
	fmt.Println("5. Groups and Alternation:")
	groupPatterns := map[string]string{
		`(func|type|var)\s+\w+`: "Go declarations",
		`(TODO|FIXME|HACK)`:     "Code annotations",
		`(http|https)://\S+`:    "URLs",
		`\b(get|set|is|has)\w+`: "Getter/setter patterns",
		`(error|err)\b`:         "Error-related words",
	}

	for pattern, description := range groupPatterns {
		fmt.Printf("  %s:\n", description)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Pattern: %s\n", pattern)
		fmt.Printf("    Found %d matches\n", results.Count())
		fmt.Println()
	}

	// Example 6: Case-insensitive patterns
	fmt.Println("6. Case-Insensitive Patterns:")
	casePatterns := []string{
		`(?i)error`,
		`(?i)test.*case`,
		`(?i)todo|fixme`,
		`(?i)func\s+test\w+`,
	}

	for _, pattern := range casePatterns {
		fmt.Printf("  Pattern: %s\n", pattern)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Found %d matches\n", results.Count())

		// Compare with case-sensitive version
		caseResults, err := goripgrep.Find(pattern[4:], ".") // Remove (?i)
		if err == nil {
			fmt.Printf("    Case-sensitive would find: %d matches\n", caseResults.Count())
		}
		fmt.Println()
	}

	// Example 7: Complex real-world patterns
	fmt.Println("7. Complex Real-World Patterns:")
	complexPatterns := map[string]string{
		`func\s+\(\w+\s+\*?\w+\)\s+\w+\([^)]*\)`:  "Go method definitions",
		`type\s+\w+\s+struct\s*\{`:                "Struct definitions",
		`if\s+err\s*!=\s*nil\s*\{`:                "Go error handling",
		`\w+\s*:=\s*\w+\([^)]*\)`:                 "Go short variable declarations",
		`import\s*\(\s*\n(\s*"[^"]+"\s*\n)*\s*\)`: "Multi-line imports",
	}

	for pattern, description := range complexPatterns {
		fmt.Printf("  %s:\n", description)
		results, err := goripgrep.Find(pattern, ".")
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
			continue
		}
		fmt.Printf("    Pattern: %s\n", pattern)
		fmt.Printf("    Found %d matches\n", results.Count())

		// Show first match as example
		if results.HasMatches() {
			match := results.Matches[0]
			fmt.Printf("    Example: %s:%d: %s\n", match.File, match.Line, match.Content)
		}
		fmt.Println()
	}

	// Example 8: Performance comparison
	fmt.Println("8. Performance Comparison:")
	fmt.Println("  Comparing literal vs regex performance:")

	// Literal search
	literalResults, err := goripgrep.Find("func", ".")
	if err != nil {
		log.Fatal(err)
	}

	// Regex search for same pattern
	regexResults, err := goripgrep.Find(`func`, ".")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("    Literal 'func': %d matches in %v\n",
		literalResults.Count(), literalResults.Stats.Duration)
	fmt.Printf("    Regex 'func': %d matches in %v\n",
		regexResults.Count(), regexResults.Stats.Duration)

	if literalResults.Stats.Duration > 0 && regexResults.Stats.Duration > 0 {
		ratio := float64(regexResults.Stats.Duration) / float64(literalResults.Stats.Duration)
		fmt.Printf("    Regex is %.2fx the time of literal search\n", ratio)
	}
}
