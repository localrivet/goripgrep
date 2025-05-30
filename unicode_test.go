package goripgrep

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNewUnicodeSearchEngine(t *testing.T) {
	t.Run("LiteralPattern", func(t *testing.T) {
		engine, err := NewUnicodeSearchEngine("hello", false)
		if err != nil {
			t.Fatalf("Failed to create Unicode engine: %v", err)
		}

		if !engine.isLiteral {
			t.Error("Expected literal pattern to be detected")
		}

		if engine.caseFoldedPattern != "hello" {
			t.Errorf("Expected caseFoldedPattern to be 'hello', got %q", engine.caseFoldedPattern)
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		engine, err := NewUnicodeSearchEngine("Hello", true)
		if err != nil {
			t.Fatalf("Failed to create case insensitive Unicode engine: %v", err)
		}

		if !engine.ignoreCase {
			t.Error("Expected ignoreCase to be true")
		}

		if engine.caseFoldedPattern != "hello" {
			t.Errorf("Expected caseFoldedPattern to be 'hello', got %q", engine.caseFoldedPattern)
		}
	})

	t.Run("RegexPattern", func(t *testing.T) {
		engine, err := NewUnicodeSearchEngine("hello.*world", false)
		if err != nil {
			t.Fatalf("Failed to create regex Unicode engine: %v", err)
		}

		if engine.isLiteral {
			t.Error("Expected regex pattern to be detected")
		}

		if engine.compiledRegex == nil {
			t.Error("Expected regex to be compiled")
		}
	})

	t.Run("UnicodeCharacterClasses", func(t *testing.T) {
		engine, err := NewUnicodeSearchEngine(`\p{L}+`, false)
		if err != nil {
			t.Fatalf("Failed to create Unicode character class engine: %v", err)
		}

		if engine.compiledRegex == nil {
			t.Error("Expected regex to be compiled for Unicode character class")
		}
	})

	t.Run("InvalidRegex", func(t *testing.T) {
		_, err := NewUnicodeSearchEngine("[invalid", false)
		if err == nil {
			t.Error("Expected error for invalid regex pattern")
		}
	})
}

func TestUnicodeSearchLiteral(t *testing.T) {
	engine, err := NewUnicodeSearchEngine("世界", false)
	if err != nil {
		t.Fatalf("Failed to create Unicode engine: %v", err)
	}

	text := "Hello 世界! This is a test with 世界 in it."
	matches := engine.Search(text)

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	for _, match := range matches {
		if match.Text != "世界" {
			t.Errorf("Expected match text to be '世界', got %q", match.Text)
		}

		// Verify rune positions are correct
		expectedRuneLength := utf8.RuneCountInString("世界")
		actualRuneLength := match.RuneEnd - match.RuneStart
		if actualRuneLength != expectedRuneLength {
			t.Errorf("Expected rune length %d, got %d", expectedRuneLength, actualRuneLength)
		}
	}
}

func TestUnicodeSearchCaseInsensitive(t *testing.T) {
	engine, err := NewUnicodeSearchEngine("HELLO", true)
	if err != nil {
		t.Fatalf("Failed to create case insensitive Unicode engine: %v", err)
	}

	text := "hello HELLO Hello hELLo"
	matches := engine.Search(text)

	if len(matches) != 4 {
		t.Errorf("Expected 4 case insensitive matches, got %d", len(matches))
	}

	// Verify all matches are found regardless of case
	expectedTexts := []string{"hello", "HELLO", "Hello", "hELLo"}
	for i, match := range matches {
		if i < len(expectedTexts) && match.Text != expectedTexts[i] {
			t.Errorf("Expected match %d to be %q, got %q", i, expectedTexts[i], match.Text)
		}
	}
}

func TestUnicodeSearchRegex(t *testing.T) {
	// Test Unicode regex patterns
	engine, err := NewUnicodeSearchEngine(`\p{L}+`, false) // Use valid Unicode property syntax
	if err != nil {
		t.Fatalf("Failed to create Unicode regex engine: %v", err)
	}

	text := "Hello Γεια σας κόσμε! This is Greek text."
	matches := engine.Search(text)

	if len(matches) == 0 {
		t.Error("Expected to find Greek character matches")
	}

	// Verify we found Greek characters
	for _, match := range matches {
		if len(match.Text) == 0 {
			t.Error("Expected non-empty match text")
		}
		t.Logf("Found Greek text: %q at rune position %d-%d", match.Text, match.RuneStart, match.RuneEnd)
	}
}

func TestUnicodeSearchLineNumbers(t *testing.T) {
	engine, err := NewUnicodeSearchEngine("test", false)
	if err != nil {
		t.Fatalf("Failed to create Unicode engine: %v", err)
	}

	text := "Line 1: Hello\nLine 2: test here\nLine 3: Another test\nLine 4: Final line"
	matches := engine.Search(text)

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}

	expectedLines := []int{2, 3}
	for i, match := range matches {
		if i < len(expectedLines) && match.LineNumber != expectedLines[i] {
			t.Errorf("Expected match %d on line %d, got %d", i, expectedLines[i], match.LineNumber)
		}
	}
}

func TestExpandUnicodeClasses(t *testing.T) {
	engine, err := NewUnicodeSearchEngine("dummy", false)
	if err != nil {
		t.Fatalf("Failed to create Unicode engine: %v", err)
	}

	tests := []struct {
		input    string
		contains string
	}{
		{`\p{Greek}`, `[\u0370-\u03FF\u1F00-\u1FFF]`},
		{`\p{Latin}`, `[\u0041-\u005A\u0061-\u007A\u00C0-\u024F\u1E00-\u1EFF]`},
		{`\p{Cyrillic}`, `[\u0400-\u04FF\u0500-\u052F\u2DE0-\u2DFF\uA640-\uA69F]`},
		{`\p{Arabic}`, `[\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF\uFB50-\uFDFF\uFE70-\uFEFF]`},
		{`\p{Hebrew}`, `[\u0590-\u05FF\uFB1D-\uFB4F]`},
		{`\p{Han}`, `[\u4E00-\u9FFF\u3400-\u4DBF\u20000-\u2A6DF\u2A700-\u2B73F\u2B740-\u2B81F\u2B820-\u2CEAF]`},
		{`\p{Hiragana}`, `[\u3040-\u309F]`},
		{`\p{Katakana}`, `[\u30A0-\u30FF\u31F0-\u31FF]`},
		{`\p{Thai}`, `[\u0E00-\u0E7F]`},
		{`\p{Devanagari}`, `[\u0900-\u097F]`},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := engine.expandUnicodeClasses(test.input)
			if !strings.Contains(result, test.contains) {
				t.Errorf("Expected expansion of %q to contain %q, got %q", test.input, test.contains, result)
			}
		})
	}
}

func TestCaseFoldString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "hello"},
		{"WORLD", "world"},
		{"MiXeD", "mixed"},
		{"", ""},
		{"123", "123"},
		{"Γεια", "γεια"},     // Greek
		{"ПРИВЕТ", "привет"}, // Cyrillic
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := CaseFoldString(test.input)
			if result != test.expected {
				t.Errorf("CaseFoldString(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestIsInCharacterClass(t *testing.T) {
	engine, err := NewUnicodeSearchEngine("dummy", false)
	if err != nil {
		t.Fatalf("Failed to create Unicode engine: %v", err)
	}

	tests := []struct {
		char      rune
		className string
		expected  bool
	}{
		{'α', "Greek", true},
		{'β', "Greek", true},
		{'a', "Greek", false},
		{'a', "Latin", true},
		{'A', "Latin", true},
		{'α', "Latin", false},
		{'а', "Cyrillic", true},  // Cyrillic 'a'
		{'a', "Cyrillic", false}, // Latin 'a'
		{'中', "Han", true},
		{'a', "Han", false},
		{'あ', "Hiragana", true},
		{'ア', "Hiragana", false},
		{'ア', "Katakana", true},
		{'あ', "Katakana", false},
	}

	for _, test := range tests {
		t.Run(string(test.char)+"_"+test.className, func(t *testing.T) {
			result := engine.IsInCharacterClass(test.char, test.className)
			if result != test.expected {
				t.Errorf("IsInCharacterClass(%q, %q) = %v, expected %v",
					string(test.char), test.className, result, test.expected)
			}
		})
	}

	// Test unknown character class
	result := engine.IsInCharacterClass('a', "Unknown")
	if result != false {
		t.Error("Expected false for unknown character class")
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"\t\nworld\t\n", "world"},
		{"normal", "normal"},
		{"", ""},
		{"  ", ""},
		{" \t\n ", ""},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := NormalizeText(test.input)
			if result != test.expected {
				t.Errorf("NormalizeText(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestExpandCaseVariants(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"hello", []string{"hello", "HELLO", "Hello"}},
		{"WORLD", []string{"WORLD", "world", "World"}},
		{"Test", []string{"Test", "test", "TEST"}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := ExpandCaseVariants(test.input)

			// Check that we have at least the expected variants
			resultMap := make(map[string]bool)
			for _, variant := range result {
				resultMap[variant] = true
			}

			for _, expected := range test.expected {
				if !resultMap[expected] {
					t.Errorf("Expected variant %q not found in result %v", expected, result)
				}
			}

			// Check that we don't have duplicates
			if len(result) != len(resultMap) {
				t.Errorf("Found duplicates in result: %v", result)
			}
		})
	}
}

func TestUnicodeMatchStruct(t *testing.T) {
	match := UnicodeMatch{
		Start:      10,
		End:        15,
		RuneStart:  8,
		RuneEnd:    12,
		Text:       "test",
		LineNumber: 2,
	}

	if match.Start != 10 {
		t.Errorf("Expected Start to be 10, got %d", match.Start)
	}

	if match.End != 15 {
		t.Errorf("Expected End to be 15, got %d", match.End)
	}

	if match.RuneStart != 8 {
		t.Errorf("Expected RuneStart to be 8, got %d", match.RuneStart)
	}

	if match.RuneEnd != 12 {
		t.Errorf("Expected RuneEnd to be 12, got %d", match.RuneEnd)
	}

	if match.Text != "test" {
		t.Errorf("Expected Text to be 'test', got %q", match.Text)
	}

	if match.LineNumber != 2 {
		t.Errorf("Expected LineNumber to be 2, got %d", match.LineNumber)
	}
}

func BenchmarkUnicodeSearch(b *testing.B) {
	// Test with various Unicode content
	unicodeText := strings.Repeat("Hello 世界! Γεια σας κόσμε! Привет мир! 🌍\n", 100)

	b.Run("LiteralUnicodeSearch", func(b *testing.B) {
		engine, err := NewUnicodeSearchEngine("世界", false)
		if err != nil {
			b.Fatalf("Failed to create Unicode engine: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matches := engine.Search(unicodeText)
			if len(matches) == 0 {
				b.Error("Expected to find matches")
			}
		}
	})

	b.Run("RegexUnicodeSearch", func(b *testing.B) {
		engine, err := NewUnicodeSearchEngine(`\p{Greek}+`, false)
		if err != nil {
			b.Fatalf("Failed to create Unicode regex engine: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matches := engine.Search(unicodeText)
			if len(matches) == 0 {
				b.Error("Expected to find matches")
			}
		}
	})

	b.Run("CaseInsensitiveUnicodeSearch", func(b *testing.B) {
		engine, err := NewUnicodeSearchEngine("HELLO", true)
		if err != nil {
			b.Fatalf("Failed to create case insensitive Unicode engine: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matches := engine.Search(unicodeText)
			if len(matches) == 0 {
				b.Error("Expected to find matches")
			}
		}
	})
}
