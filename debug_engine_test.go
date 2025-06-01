package goripgrep

import (
	"strings"
	"testing"
)

func TestDebugEngineRegexSearch(t *testing.T) {
	pattern := `\w+Sushi`
	testContent := "github.com/BurntSushi/toml,v0.3.1,h1:WXkYYl6Yr3qBf1K79EBnL4mak0OimBfB0XUf9Vl28OQ=,815c6e594745f2d8842ff9a4b0569c6695e6cdfd5e07e5b3d98d06b72ca41e3c"

	// Test pattern classification
	isLit := isLiteralPattern(pattern)
	t.Logf("Pattern '%s' is literal: %v", pattern, isLit)

	// Create engine
	args := SearchArgs{
		Pattern:      pattern,
		IgnoreCase:   nil, // Default case sensitive
		ContextLines: nil, // No context
	}

	engine, err := NewEngine(args)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	t.Logf("Engine isLiteral: %v", engine.isLiteral)
	t.Logf("Engine ignoreCase: %v", engine.ignoreCase)
	t.Logf("Engine regex: %v", engine.regex)

	// Test direct regex compilation
	if engine.regex != nil {
		regexMatches := engine.regex.FindAllIndex([]byte(testContent), -1)
		t.Logf("Direct regex matches: %v", regexMatches)

		// Test what the regex actually matches
		allMatches := engine.regex.FindAllString(testContent, -1)
		t.Logf("Direct regex match strings: %v", allMatches)
	}

	// Test findMatches function
	matches := engine.findMatches([]byte(testContent))
	t.Logf("Engine findMatches result: %v", matches)

	// Test what happens when we search the content line by line
	lines := strings.Split(testContent, "\n")
	for i, line := range lines {
		lineMatches := engine.findMatches([]byte(line))
		if len(lineMatches) > 0 {
			t.Logf("Line %d matches: %v, content: '%s'", i+1, lineMatches, line)
		}
	}
}

func TestDebugCurrentEngineAPI(t *testing.T) {
	pattern := `\w+Sushi`

	// Test using the Find API that the CLI uses
	results, err := Find(pattern, "large_test.csv",
		WithWorkers(1),
		WithContextLines(0),
		WithOptimization(true),
	)

	if err != nil {
		t.Fatalf("Find API failed: %v", err)
	}

	t.Logf("Find API returned %d matches", len(results.Matches))

	// Let's also test what happens when we disable optimization
	resultsNoOpt, err := Find(pattern, "large_test.csv",
		WithWorkers(1),
		WithContextLines(0),
		WithOptimization(false),
	)

	if err != nil {
		t.Fatalf("Find API (no optimization) failed: %v", err)
	}

	t.Logf("Find API (no optimization) returned %d matches", len(resultsNoOpt.Matches))

	// Let's test with streaming disabled too
	resultsNoStream, err := Find(pattern, "large_test.csv",
		WithWorkers(1),
		WithContextLines(0),
		WithOptimization(true),
		WithStreamingSearch(false),
	)

	if err != nil {
		t.Fatalf("Find API (no streaming) failed: %v", err)
	}

	t.Logf("Find API (no streaming) returned %d matches", len(resultsNoStream.Matches))
}
