package goripgrep

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	unicodeenc "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/language"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// UnicodeSearchEngine provides advanced Unicode-aware search capabilities
type UnicodeSearchEngine struct {
	pattern           string
	compiledRegex     *regexp.Regexp
	caseFoldedPattern string
	isLiteral         bool
	ignoreCase        bool

	// Unicode character class support
	characterClasses map[string]*unicode.RangeTable
}

// NewUnicodeSearchEngine creates a Unicode-aware search engine
func NewUnicodeSearchEngine(pattern string, ignoreCase bool) (*UnicodeSearchEngine, error) {
	engine := &UnicodeSearchEngine{
		pattern:    pattern,
		ignoreCase: ignoreCase,
		isLiteral:  isLiteralPattern(pattern),
		characterClasses: map[string]*unicode.RangeTable{
			"Greek":      unicode.Greek,
			"Latin":      unicode.Latin,
			"Cyrillic":   unicode.Cyrillic,
			"Arabic":     unicode.Arabic,
			"Hebrew":     unicode.Hebrew,
			"Han":        unicode.Han,
			"Hiragana":   unicode.Hiragana,
			"Katakana":   unicode.Katakana,
			"Thai":       unicode.Thai,
			"Devanagari": unicode.Devanagari,
		},
	}

	if engine.isLiteral {
		if ignoreCase {
			engine.caseFoldedPattern = strings.ToLower(pattern)
		} else {
			engine.caseFoldedPattern = pattern
		}
	} else {
		// Expand Unicode character classes in the pattern
		expandedPattern := engine.expandUnicodeClasses(pattern)

		if ignoreCase {
			expandedPattern = "(?i)" + expandedPattern
		}

		var err error
		engine.compiledRegex, err = regexp.Compile(expandedPattern)
		if err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// expandUnicodeClasses expands \p{ClassName} patterns to character ranges
func (e *UnicodeSearchEngine) expandUnicodeClasses(pattern string) string {
	// Handle \p{ClassName} patterns
	result := pattern

	// Simple expansion for common Unicode classes
	unicodeClasses := map[string]string{
		`\p{Greek}`:      `[\u0370-\u03FF\u1F00-\u1FFF]`,
		`\p{Latin}`:      `[\u0041-\u005A\u0061-\u007A\u00C0-\u024F\u1E00-\u1EFF]`,
		`\p{Cyrillic}`:   `[\u0400-\u04FF\u0500-\u052F\u2DE0-\u2DFF\uA640-\uA69F]`,
		`\p{Arabic}`:     `[\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF\uFB50-\uFDFF\uFE70-\uFEFF]`,
		`\p{Hebrew}`:     `[\u0590-\u05FF\uFB1D-\uFB4F]`,
		`\p{Han}`:        `[\u4E00-\u9FFF\u3400-\u4DBF\u20000-\u2A6DF\u2A700-\u2B73F\u2B740-\u2B81F\u2B820-\u2CEAF]`,
		`\p{Hiragana}`:   `[\u3040-\u309F]`,
		`\p{Katakana}`:   `[\u30A0-\u30FF\u31F0-\u31FF]`,
		`\p{Thai}`:       `[\u0E00-\u0E7F]`,
		`\p{Devanagari}`: `[\u0900-\u097F]`,
	}

	for class, replacement := range unicodeClasses {
		result = strings.ReplaceAll(result, class, replacement)
	}

	return result
}

// Search performs Unicode-aware search on text
func (e *UnicodeSearchEngine) Search(text string) []UnicodeMatch {
	var matches []UnicodeMatch

	if e.isLiteral {
		matches = e.searchLiteral(text)
	} else if e.compiledRegex != nil {
		matches = e.searchRegex(text)
	}

	return matches
}

// UnicodeMatch represents a match with Unicode-aware information
type UnicodeMatch struct {
	Start      int    // Byte offset
	End        int    // Byte offset
	RuneStart  int    // Rune offset
	RuneEnd    int    // Rune offset
	Text       string // Matched text
	LineNumber int    // Line number (1-based)
}

// searchLiteral performs Unicode-aware literal search
func (e *UnicodeSearchEngine) searchLiteral(text string) []UnicodeMatch {
	var matches []UnicodeMatch
	searchText := text
	pattern := e.caseFoldedPattern

	if e.ignoreCase {
		searchText = strings.ToLower(text)
	}

	pos := 0
	lineNum := 1

	for {
		idx := strings.Index(searchText[pos:], pattern)
		if idx == -1 {
			break
		}

		actualPos := pos + idx

		// Convert byte positions to rune positions
		runeStart := utf8.RuneCountInString(text[:actualPos])
		runeEnd := runeStart + utf8.RuneCountInString(pattern)

		// Count line number
		lineNum += strings.Count(text[pos:actualPos], "\n")

		match := UnicodeMatch{
			Start:      actualPos,
			End:        actualPos + len(pattern),
			RuneStart:  runeStart,
			RuneEnd:    runeEnd,
			Text:       text[actualPos : actualPos+len(pattern)],
			LineNumber: lineNum,
		}

		matches = append(matches, match)
		pos = actualPos + len(pattern)
	}

	return matches
}

// searchRegex performs Unicode-aware regex search
func (e *UnicodeSearchEngine) searchRegex(text string) []UnicodeMatch {
	var matches []UnicodeMatch

	regexMatches := e.compiledRegex.FindAllStringIndex(text, -1)

	for _, match := range regexMatches {
		start, end := match[0], match[1]

		// Convert byte positions to rune positions
		runeStart := utf8.RuneCountInString(text[:start])
		runeEnd := utf8.RuneCountInString(text[:end])

		// Count line number
		lineNum := strings.Count(text[:start], "\n") + 1

		unicodeMatch := UnicodeMatch{
			Start:      start,
			End:        end,
			RuneStart:  runeStart,
			RuneEnd:    runeEnd,
			Text:       text[start:end],
			LineNumber: lineNum,
		}

		matches = append(matches, unicodeMatch)
	}

	return matches
}

// CaseFoldString performs Unicode case folding
func CaseFoldString(s string) string {
	// Go's strings.ToLower handles basic Unicode case folding
	// For more advanced case folding, we'd need to implement
	// the full Unicode case folding algorithm
	return strings.ToLower(s)
}

// IsInCharacterClass checks if a rune belongs to a Unicode character class
func (e *UnicodeSearchEngine) IsInCharacterClass(r rune, className string) bool {
	if rangeTable, exists := e.characterClasses[className]; exists {
		return unicode.Is(rangeTable, r)
	}
	return false
}

// NormalizeText performs Unicode normalization (basic implementation)
func NormalizeText(text string) string {
	// This is a simplified normalization
	// For full Unicode normalization, we'd use golang.org/x/text/unicode/norm
	return strings.TrimSpace(text)
}

// ExpandCaseVariants generates case variants for a string
func ExpandCaseVariants(s string) []string {
	variants := make(map[string]bool)

	// Add original
	variants[s] = true

	// Add lowercase
	variants[strings.ToLower(s)] = true

	// Add uppercase
	variants[strings.ToUpper(s)] = true

	// Add title case using the new cases package
	caser := cases.Title(language.English)
	variants[caser.String(strings.ToLower(s))] = true

	// Convert map to slice
	result := make([]string, 0, len(variants))
	for variant := range variants {
		result = append(result, variant)
	}

	return result
}

// EncodingDetector provides encoding detection and transcoding capabilities
type EncodingDetector struct {
	supportedEncodings map[string]transform.Transformer
}

// NewEncodingDetector creates a new encoding detector with support for common encodings
func NewEncodingDetector() *EncodingDetector {
	return &EncodingDetector{
		supportedEncodings: map[string]transform.Transformer{
			// Unicode encodings
			"UTF-8":    unicodeenc.UTF8.NewDecoder(),
			"UTF-16BE": unicodeenc.UTF16(unicodeenc.BigEndian, unicodeenc.IgnoreBOM).NewDecoder(),
			"UTF-16LE": unicodeenc.UTF16(unicodeenc.LittleEndian, unicodeenc.IgnoreBOM).NewDecoder(),

			// Western European encodings
			"ISO-8859-1":   charmap.ISO8859_1.NewDecoder(),
			"ISO-8859-15":  charmap.ISO8859_15.NewDecoder(),
			"Windows-1252": charmap.Windows1252.NewDecoder(),

			// Eastern European encodings
			"ISO-8859-2":   charmap.ISO8859_2.NewDecoder(),
			"Windows-1250": charmap.Windows1250.NewDecoder(),

			// Cyrillic encodings
			"ISO-8859-5":   charmap.ISO8859_5.NewDecoder(),
			"Windows-1251": charmap.Windows1251.NewDecoder(),
			"KOI8-R":       charmap.KOI8R.NewDecoder(),

			// Japanese encodings
			"Shift_JIS":   japanese.ShiftJIS.NewDecoder(),
			"EUC-JP":      japanese.EUCJP.NewDecoder(),
			"ISO-2022-JP": japanese.ISO2022JP.NewDecoder(),

			// Chinese encodings
			"GBK":     simplifiedchinese.GBK.NewDecoder(),
			"GB18030": simplifiedchinese.GB18030.NewDecoder(),
			"Big5":    traditionalchinese.Big5.NewDecoder(),

			// Korean encodings
			"EUC-KR": korean.EUCKR.NewDecoder(),
		},
	}
}

// DetectEncoding attempts to detect the encoding of the given data
func (ed *EncodingDetector) DetectEncoding(data []byte) (string, transform.Transformer) {
	// Check for BOM first
	if bomEncoding, bomName := ed.detectBOM(data); bomEncoding != nil {
		return bomName, bomEncoding
	}

	// Check if it's valid UTF-8
	if utf8.Valid(data) {
		return "UTF-8", unicodeenc.UTF8.NewDecoder()
	}

	// Try to detect other encodings by attempting to decode
	// This is a simplified heuristic-based approach
	return ed.heuristicDetection(data)
}

// detectBOM detects Byte Order Mark and returns the corresponding encoding
func (ed *EncodingDetector) detectBOM(data []byte) (transform.Transformer, string) {
	if len(data) < 2 {
		return nil, ""
	}

	// UTF-8 BOM: EF BB BF
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return unicodeenc.UTF8.NewDecoder(), "UTF-8"
	}

	// UTF-16 BE BOM: FE FF
	if data[0] == 0xFE && data[1] == 0xFF {
		return unicodeenc.UTF16(unicodeenc.BigEndian, unicodeenc.UseBOM).NewDecoder(), "UTF-16BE"
	}

	// UTF-16 LE BOM: FF FE
	if data[0] == 0xFF && data[1] == 0xFE {
		return unicodeenc.UTF16(unicodeenc.LittleEndian, unicodeenc.UseBOM).NewDecoder(), "UTF-16LE"
	}

	return nil, ""
}

// heuristicDetection uses heuristics to detect encoding
func (ed *EncodingDetector) heuristicDetection(data []byte) (string, transform.Transformer) {
	// Try common encodings and see which one produces the most valid text
	candidates := []string{
		"Windows-1252", "ISO-8859-1", "Windows-1251", "GBK", "Shift_JIS", "EUC-JP",
	}

	bestEncoding := "ISO-8859-1" // Fallback
	bestScore := 0

	for _, encodingName := range candidates {
		if enc, exists := ed.supportedEncodings[encodingName]; exists {
			score := ed.scoreEncoding(data, enc)
			if score > bestScore {
				bestScore = score
				bestEncoding = encodingName
			}
		}
	}

	return bestEncoding, ed.supportedEncodings[bestEncoding]
}

// scoreEncoding gives a score to how likely the data is in the given encoding
func (ed *EncodingDetector) scoreEncoding(data []byte, transformer transform.Transformer) int {
	decoded, _, err := transform.Bytes(transformer, data)
	if err != nil {
		return 0
	}

	// Score based on valid UTF-8 after decoding and printable characters
	if !utf8.Valid(decoded) {
		return 0
	}

	score := 0
	for _, b := range decoded {
		if b >= 32 && b <= 126 { // Printable ASCII
			score += 2
		} else if b >= 128 { // Non-ASCII but valid UTF-8
			score += 1
		}
	}

	return score
}

// TranscodeToUTF8 converts data from the detected encoding to UTF-8
func (ed *EncodingDetector) TranscodeToUTF8(data []byte, srcTransformer transform.Transformer) ([]byte, error) {
	if srcTransformer == nil {
		return data, nil
	}

	decoded, _, err := transform.Bytes(srcTransformer, data)
	return decoded, err
}

// ProcessFileWithEncoding reads a file, detects its encoding, and returns UTF-8 content
func (ed *EncodingDetector) ProcessFileWithEncoding(filePath string) ([]byte, string, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}

	// Detect encoding
	encodingName, transformer := ed.DetectEncoding(data)

	// Transcode to UTF-8
	utf8Data, err := ed.TranscodeToUTF8(data, transformer)
	if err != nil {
		return nil, "", fmt.Errorf("failed to transcode from %s: %w", encodingName, err)
	}

	return utf8Data, encodingName, nil
}

// UnicodeNormalizer provides Unicode normalization capabilities
type UnicodeNormalizer struct {
	form norm.Form
}

// NewUnicodeNormalizer creates a new Unicode normalizer
func NewUnicodeNormalizer(form norm.Form) *UnicodeNormalizer {
	return &UnicodeNormalizer{form: form}
}

// Normalize normalizes the input string using the specified form
func (un *UnicodeNormalizer) Normalize(s string) string {
	return un.form.String(s)
}

// NormalizeBytes normalizes the input bytes using the specified form
func (un *UnicodeNormalizer) NormalizeBytes(b []byte) []byte {
	return un.form.Bytes(b)
}

// IsNormalized checks if the string is already normalized
func (un *UnicodeNormalizer) IsNormalized(s string) bool {
	return un.form.IsNormal([]byte(s))
}

// AdvancedCaseFolding provides Unicode-aware case folding
type AdvancedCaseFolding struct {
	caser cases.Caser
}

// NewAdvancedCaseFolding creates a new advanced case folding instance
func NewAdvancedCaseFolding(lang language.Tag) *AdvancedCaseFolding {
	return &AdvancedCaseFolding{
		caser: cases.Fold(),
	}
}

// Fold performs Unicode case folding on the input string
func (acf *AdvancedCaseFolding) Fold(s string) string {
	return acf.caser.String(s)
}

// FoldBytes performs Unicode case folding on the input bytes
func (acf *AdvancedCaseFolding) FoldBytes(b []byte) []byte {
	return []byte(acf.caser.String(string(b)))
}

// EnhancedUnicodeSearchEngine combines encoding detection, normalization, and case folding
type EnhancedUnicodeSearchEngine struct {
	*UnicodeSearchEngine
	detector   *EncodingDetector
	normalizer *UnicodeNormalizer
	caseFolder *AdvancedCaseFolding
}

// NewEnhancedUnicodeSearchEngine creates an enhanced Unicode search engine
func NewEnhancedUnicodeSearchEngine(pattern string, ignoreCase bool, lang language.Tag) (*EnhancedUnicodeSearchEngine, error) {
	baseEngine, err := NewUnicodeSearchEngine(pattern, ignoreCase)
	if err != nil {
		return nil, err
	}

	return &EnhancedUnicodeSearchEngine{
		UnicodeSearchEngine: baseEngine,
		detector:            NewEncodingDetector(),
		normalizer:          NewUnicodeNormalizer(norm.NFC), // Use NFC normalization by default
		caseFolder:          NewAdvancedCaseFolding(lang),
	}, nil
}

// SearchFile searches a file with automatic encoding detection and Unicode normalization
func (euse *EnhancedUnicodeSearchEngine) SearchFile(filePath string) ([]UnicodeMatch, string, error) {
	// Process file with encoding detection
	utf8Data, encodingName, err := euse.detector.ProcessFileWithEncoding(filePath)
	if err != nil {
		return nil, "", err
	}

	// Normalize the text
	normalizedText := euse.normalizer.Normalize(string(utf8Data))

	// Perform the search
	matches := euse.Search(normalizedText)

	return matches, encodingName, nil
}

// SearchWithPreprocessing searches text with full Unicode preprocessing
func (euse *EnhancedUnicodeSearchEngine) SearchWithPreprocessing(text string) []UnicodeMatch {
	// Normalize the text
	normalizedText := euse.normalizer.Normalize(text)

	// Apply case folding if case-insensitive
	if euse.ignoreCase {
		normalizedText = euse.caseFolder.Fold(normalizedText)
	}

	// Perform the search
	return euse.Search(normalizedText)
}
