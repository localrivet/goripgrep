package goripgrep

import (
	"testing"
)

func TestNewRegexEngine(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		ignoreCase bool
		wantErr    bool
	}{
		{
			name:       "BasicPattern",
			pattern:    "hello",
			ignoreCase: false,
			wantErr:    false,
		},
		{
			name:       "CaseInsensitive",
			pattern:    "HELLO",
			ignoreCase: true,
			wantErr:    false,
		},
		{
			name:       "InvalidPattern",
			pattern:    "[invalid",
			ignoreCase: false,
			wantErr:    true,
		},
		{
			name:       "EmptyPattern",
			pattern:    "",
			ignoreCase: false,
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewRegex(test.pattern, test.ignoreCase)

			if test.wantErr {
				if err == nil {
					t.Errorf("NewRegex(%q, %v) expected error, got nil", test.pattern, test.ignoreCase)
				}
				return
			}

			if err != nil {
				t.Errorf("NewRegex(%q, %v) unexpected error: %v", test.pattern, test.ignoreCase, err)
				return
			}

			if engine == nil {
				t.Errorf("NewRegex(%q, %v) returned nil engine", test.pattern, test.ignoreCase)
			}
		})
	}
}

func TestRegexEngineFindAllMatches(t *testing.T) {
	engine, err := NewRegex(`\b\w+\b`, false)
	if err != nil {
		t.Fatalf("Failed to create regex engine: %v", err)
	}

	text := "hello world test"
	matches := engine.FindAll(text)

	if len(matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(matches))
	}

	expectedWords := []string{"hello", "world", "test"}
	for i, match := range matches {
		if i < len(expectedWords) && match.Text != expectedWords[i] {
			t.Errorf("Expected match %d to be '%s', got '%s'", i, expectedWords[i], match.Text)
		}
	}
}

func TestRegexEngineMatchesPattern(t *testing.T) {
	engine, err := NewRegex(`^\d{3}-\d{2}-\d{4}$`, false)
	if err != nil {
		t.Fatalf("Failed to create regex engine: %v", err)
	}

	tests := []struct {
		input    string
		expected bool
	}{
		{"123-45-6789", true},
		{"000-00-0000", true},
		{"123-456-789", false},
		{"12-34-5678", false},
		{"123-45-67890", false},
		{"abc-de-fghi", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := engine.Matches(test.input)
			if result != test.expected {
				t.Errorf("Matches(%q) = %v, expected %v", test.input, result, test.expected)
			}
		})
	}
}

func TestRegexEngineReplaceAllMatches(t *testing.T) {
	engine, err := NewRegex(`\b\w+\b`, false)
	if err != nil {
		t.Fatalf("Failed to create regex engine: %v", err)
	}

	text := "hello world"
	result := engine.ReplaceAll(text, "X")
	expected := "X X"

	if result != expected {
		t.Errorf("ReplaceAll(%q, %q) = %q, expected %q", text, "X", result, expected)
	}
}

func TestRegexEngineGetCaptureGroups(t *testing.T) {
	engine, err := NewRegex(`(\w+)\s+(\w+)`, false)
	if err != nil {
		t.Fatalf("Failed to create regex engine: %v", err)
	}

	text := "hello world"
	groups := engine.Groups(text)

	expectedGroups := []string{"hello world", "hello", "world"}
	if len(groups) != len(expectedGroups) {
		t.Errorf("Expected %d groups, got %d", len(expectedGroups), len(groups))
		return
	}

	for i, group := range groups {
		if group != expectedGroups[i] {
			t.Errorf("Group %d: expected %q, got %q", i, expectedGroups[i], group)
		}
	}
}

func TestRegexEngineGetNamedGroups(t *testing.T) {
	engine, err := NewRegex(`(?P<first>\w+)\s+(?P<second>\w+)`, false)
	if err != nil {
		t.Fatalf("Failed to create regex engine: %v", err)
	}

	text := "hello world"
	named := engine.NamedGroups(text)

	expected := map[string]string{
		"first":  "hello",
		"second": "world",
	}

	for name, expectedValue := range expected {
		if value, exists := named[name]; !exists {
			t.Errorf("Expected named group %q not found", name)
		} else if value != expectedValue {
			t.Errorf("Named group %q: expected %q, got %q", name, expectedValue, value)
		}
	}
}

func TestRegexEngineValidatePattern(t *testing.T) {
	tests := []struct {
		pattern string
		valid   bool
	}{
		{"hello", true},
		{"hello.*world", true},
		{"^start", true},
		{"end$", true},
		{"[a-z]+", true},
		{`\d{3}-\d{2}-\d{4}`, true},
		{"(?P<name>\\w+)", true},
		{"[invalid", false},
		{"(unclosed", false},
		{"*invalid", false},
		{"+invalid", false},
		{"?invalid", false},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			err := Validate(test.pattern)
			isValid := err == nil

			if isValid != test.valid {
				t.Errorf("Validate(%q) validity = %v, expected %v", test.pattern, isValid, test.valid)
				if err != nil {
					t.Logf("Error: %v", err)
				}
			}
		})
	}
}

func TestRegexEngineOptimizePattern(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello|hello", "hello|hello"},
		{"(hello)", "(hello)"},
		{"hello+", "hello+"},
		{"hello*", "hello*"},
		{"hello?", "hello?"},
		{"^hello$", "^hello$"},
		{"[a-z]", "[a-z]"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := Optimize(test.input)
			if result != test.expected {
				t.Errorf("Optimize(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestRegexEngineEscapeSpecialChars(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello.world", "hello\\.world"},
		{"hello*world", "hello\\*world"},
		{"hello+world", "hello\\+world"},
		{"hello?world", "hello\\?world"},
		{"hello^world", "hello\\^world"},
		{"hello$world", "hello\\$world"},
		{"hello|world", "hello\\|world"},
		{"hello(world)", "hello\\(world\\)"},
		{"hello[world]", "hello\\[world\\]"},
		{"hello{world}", "hello\\{world\\}"},
		{"hello\\world", "hello\\\\world"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := Escape(test.input)
			if result != test.expected {
				t.Errorf("Escape(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestRegexEngineGetComplexity(t *testing.T) {
	tests := []struct {
		pattern  string
		minScore int
		maxScore int
	}{
		{"hello", 1, 1},
		{"hello.*world", 3, 4},
		{"^hello$", 1, 2},
		{"[a-z]+", 3, 4},
		{`\d{3}-\d{2}-\d{4}`, 2, 3},
		{"(?P<name>\\w+)@(?P<domain>\\w+)\\.(?P<tld>\\w+)", 5, 7},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			score := Complexity(test.pattern)
			if score < test.minScore || score > test.maxScore {
				t.Errorf("Complexity(%q) = %d, expected between %d and %d", test.pattern, score, test.minScore, test.maxScore)
			}
		})
	}
}

func TestRegexEngineIsLiteral(t *testing.T) {
	tests := []struct {
		pattern  string
		expected bool
	}{
		{"hello", true},
		{"hello_world", true},
		{"hello123", true},
		{"hello_world", true},
		{"hello-world", true},
		{"hello.world", false},
		{"hello*", false},
		{"hello+", false},
		{"hello?", false},
		{"^hello", false},
		{"hello$", false},
		{"hello|world", false},
		{"hello(world)", false},
		{"hello[world]", false},
		{"hello{1,2}", false},
		{"hello\\world", false},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			result := IsLiteral(test.pattern)
			if result != test.expected {
				t.Errorf("IsLiteral(%q) = %v, expected %v", test.pattern, result, test.expected)
			}
		})
	}
}

func TestRegexEngineExtractLiterals(t *testing.T) {
	tests := []struct {
		pattern  string
		expected int
	}{
		{"hello", 1},
		{"hello.*world", 2},
		{"^start.*end$", 2},
		{"prefix\\d+suffix", 2},
		{".*", 0},
		{"[abc]", 0},
		{"a+", 0},
	}

	for _, test := range tests {
		t.Run(test.pattern, func(t *testing.T) {
			literals := ExtractLiterals(test.pattern)
			if len(literals) != test.expected {
				t.Errorf("ExtractLiterals(%q) returned %d literals, expected %d", test.pattern, len(literals), test.expected)
			}
		})
	}
}

func TestRegexMatchStruct(t *testing.T) {
	match := RegexMatch{
		Start:  0,
		End:    5,
		Text:   "hello",
		Groups: []string{"hello"},
		Named:  map[string]string{"word": "hello"},
	}

	if match.Start != 0 {
		t.Errorf("Expected Start = 0, got %d", match.Start)
	}
	if match.End != 5 {
		t.Errorf("Expected End = 5, got %d", match.End)
	}
	if match.Text != "hello" {
		t.Errorf("Expected Text = 'hello', got '%s'", match.Text)
	}
}

func TestRegexEngineMultilineSearch(t *testing.T) {
	engine, err := NewRegex(`test`, false)
	if err != nil {
		t.Fatalf("Failed to create multiline regex engine: %v", err)
	}

	text := "hello\ntest line\nanother test\nend"
	matches := engine.FindAll(text)

	if len(matches) != 2 {
		t.Errorf("Expected 2 multiline matches, got %d", len(matches))
	}

	for _, match := range matches {
		if match.Text != "test" {
			t.Errorf("Expected match text 'test', got '%s'", match.Text)
		}
	}
}

func TestRegexEngineCompileWithFlags(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		ignoreCase  bool
		text        string
		shouldMatch bool
	}{
		{
			name:        "hello",
			pattern:     "hello",
			ignoreCase:  false,
			text:        "Hello World",
			shouldMatch: false,
		},
		{
			name:        "hello",
			pattern:     "hello",
			ignoreCase:  true,
			text:        "Hello World",
			shouldMatch: true,
		},
		{
			name:        "^test",
			pattern:     "^test",
			ignoreCase:  false,
			text:        "test line",
			shouldMatch: true,
		},
		{
			name:        "^test",
			pattern:     "^test",
			ignoreCase:  true,
			text:        "TEST line",
			shouldMatch: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			engine, err := NewRegex(test.pattern, test.ignoreCase)
			if err != nil {
				t.Fatalf("Failed to create regex engine: %v", err)
			}

			matches := engine.Matches(test.text)
			if matches != test.shouldMatch {
				t.Errorf("Pattern %q with ignoreCase=%v on text %q: expected %v, got %v",
					test.pattern, test.ignoreCase, test.text, test.shouldMatch, matches)
			}
		})
	}
}

func BenchmarkAdvancedRegexEngine(b *testing.B) {
	text := `Contact us at support@example.com or call 555-123-4567 for assistance.
Another email: user@domain.org and phone: 555-987-6543.
Final contact: admin@test.net`

	b.Run("EmailRegex", func(b *testing.B) {
		engine, err := NewRegex(`\b\w+@\w+\.\w+\b`, false)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.FindAll(text)
		}
	})

	b.Run("PhoneRegex", func(b *testing.B) {
		engine, err := NewRegex(`\b\d{3}-\d{3}-\d{4}\b`, false)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.FindAll(text)
		}
	})

	b.Run("NamedGroups", func(b *testing.B) {
		engine, err := NewRegex(`(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`, false)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.NamedGroups(text)
		}
	})
}
