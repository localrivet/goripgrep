// Package main demonstrates Unicode search capabilities of GoRipGrep.
//
// This example shows how to search for text in various languages and
// character sets, including emojis and special Unicode characters.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/localrivet/goripgrep"
)

func main() {
	fmt.Println("=== GoRipGrep Unicode Search Examples ===")

	// Create test files with Unicode content
	if err := createUnicodeTestFiles(); err != nil {
		log.Fatal(err)
	}
	defer cleanupTestFiles()

	// Example 1: Search for Greek text
	fmt.Println("1. Greek Text Search:")
	results, err := goripgrep.Find("Ελληνικά", "./test_unicode", goripgrep.WithIgnoreCase())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for Greek text\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 2: Search for Chinese characters
	fmt.Println("2. Chinese Text Search:")
	results, err = goripgrep.Find("中文", "./test_unicode")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for Chinese text\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 3: Search for emojis
	fmt.Println("3. Emoji Search:")
	results, err = goripgrep.Find("🚀", "./test_unicode")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for rocket emoji\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 4: Search for accented characters
	fmt.Println("4. Accented Characters Search:")
	results, err = goripgrep.Find("café", "./test_unicode", goripgrep.WithIgnoreCase())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for 'café'\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 5: Search for Arabic text
	fmt.Println("5. Arabic Text Search:")
	results, err = goripgrep.Find("العربية", "./test_unicode")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for Arabic text\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 6: Mixed Unicode search with regex
	fmt.Println("6. Mixed Unicode Regex Search:")
	results, err = goripgrep.Find(`[🚀🎉🔥]+`, "./test_unicode", goripgrep.WithIgnoreCase())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for emoji patterns\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
	}
	fmt.Println()

	// Example 7: Unicode with context lines
	fmt.Println("7. Unicode Search with Context:")
	results, err = goripgrep.Find("世界", "./test_unicode",
		goripgrep.WithContextLines(1),
		goripgrep.WithIgnoreCase())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d matches for '世界' with context\n", results.Count())
	for _, match := range results.Matches {
		fmt.Printf("  %s:%d: %s\n", match.File, match.Line, match.Content)
		for i, contextLine := range match.Context {
			if i == 0 && len(match.Context) > 1 {
				fmt.Printf("    Before: %s\n", contextLine)
			} else if i == len(match.Context)-1 && len(match.Context) > 1 {
				fmt.Printf("    After:  %s\n", contextLine)
			}
		}
	}
	fmt.Println()

	// Example 8: Multiple Unicode patterns
	fmt.Println("8. Multiple Unicode Patterns:")
	patterns := []string{"🚀", "café", "中文", "Ελληνικά"}
	for _, pattern := range patterns {
		results, err := goripgrep.Find(pattern, "./test_unicode", goripgrep.WithIgnoreCase())
		if err != nil {
			fmt.Printf("  Error searching for %s: %v\n", pattern, err)
			continue
		}
		fmt.Printf("  Pattern '%s': %d matches\n", pattern, results.Count())
	}
}

func createUnicodeTestFiles() error {
	// Create test directory
	if err := os.MkdirAll("./test_unicode", 0755); err != nil {
		return err
	}

	// Create files with different Unicode content
	files := map[string]string{
		"greek.txt": `Αυτό είναι ελληνικό κείμενο.
Ελληνικά γράμματα: Α, Β, Γ, Δ, Ε
Καλημέρα κόσμε!`,

		"chinese.txt": `这是中文文本。
中文字符测试
你好世界！`,

		"emoji.txt": `Welcome to our app! 🚀
Performance is amazing! 🔥
Let's celebrate! 🎉
Unicode support rocks! ⭐`,

		"french.txt": `Voici du texte français.
Café, résumé, naïve
Les accents sont importants!`,

		"arabic.txt": `هذا نص باللغة العربية.
اللغة العربية جميلة
مرحبا بالعالم!`,

		"mixed.txt": `Mixed content example:
English, français, Ελληνικά, 中文, العربية
Emojis: 🚀🎉🔥⭐
Special chars: ñ, ü, ç, ß`,
	}

	for filename, content := range files {
		path := filepath.Join("./test_unicode", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func cleanupTestFiles() {
	os.RemoveAll("./test_unicode")
}
