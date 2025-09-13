package analyzer

import (
	"regexp"
	"strings"
)

type TextAnalyzer struct {
	minWordLength   int
	maxPhraseLength int
	wordPattern     *regexp.Regexp
	stopWords       map[string]bool
}

type AnalysisResult struct {
	Words   []string
	Phrases []string
}

func NewTextAnalyzer() *TextAnalyzer {
	return NewTextAnalyzerWithStopWords(nil)
}

func NewTextAnalyzerWithStopWords(customStopWords []string) *TextAnalyzer {
	// Built-in stop words
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "his": true,
		"how": true, "man": true, "new": true, "now": true, "old": true,
		"see": true, "two": true, "way": true, "who": true, "boy": true,
		"end": true, "did": true, "its": true, "let": true, "put": true,
		"say": true, "she": true, "too": true, "use": true, "from": true,
		"this": true, "that": true, "there": true, "they": true, "with": true,
		"what": true, "when": true, "where": true, "which": true, "while": true,
		"why": true, "will": true, "would": true, "could": true,
		"should": true, "might": true, "must": true, "if": true, "then": true,
		"than": true, "so": true, "just": true, "like": true, "more": true,
		"some": true, "such": true, "very": true,
		"also": true, "back": true, "down": true, "over": true, "up": true,
		"after": true, "before": true, "between": true, "during": true,
		"around": true, "through": true, "across": true, "against": true,
		"without": true,
	}

	// Add custom stop words
	for _, word := range customStopWords {
		if word != "" {
			stopWords[strings.ToLower(word)] = true
		}
	}

	return &TextAnalyzer{
		minWordLength:   3,
		maxPhraseLength: 4,
		wordPattern:     regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`),
		stopWords:       stopWords,
	}
}

func (ta *TextAnalyzer) AnalyzeLine(line string) *AnalysisResult {
	result := &AnalysisResult{
		Words:   make([]string, 0),
		Phrases: make([]string, 0),
	}

	// Extract just the message part, not the timestamp/severity prefix
	messageOnly := ta.extractMessage(line)

	words := ta.extractWords(messageOnly)
	result.Words = ta.filterWords(words)
	result.Phrases = ta.extractPhrases(words)

	return result
}

func (ta *TextAnalyzer) extractWords(text string) []string {
	text = strings.ToLower(text)
	matches := ta.wordPattern.FindAllString(text, -1)
	return matches
}

func (ta *TextAnalyzer) filterWords(words []string) []string {
	filtered := make([]string, 0)

	for _, word := range words {
		if len(word) >= ta.minWordLength && !ta.stopWords[word] {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

func (ta *TextAnalyzer) extractPhrases(words []string) []string {
	phrases := make([]string, 0)

	for length := 2; length <= ta.maxPhraseLength && length <= len(words); length++ {
		for i := 0; i <= len(words)-length; i++ {
			phrase := strings.Join(words[i:i+length], " ")
			phrases = append(phrases, phrase)
		}
	}

	return phrases
}

// extractMessage attempts to extract just the log message from a full log line,
// stripping out timestamp, severity, and other metadata prefixes
// GetStopWords returns the stopwords map for external use
func (ta *TextAnalyzer) GetStopWords() map[string]bool {
	return ta.stopWords
}

func (ta *TextAnalyzer) extractMessage(line string) string {
	// Common log format patterns to match and extract message from:
	// - "2024-01-01 10:00:00 INFO message here"
	// - "2024-01-01T10:00:00Z ERROR message here"
	// - "[INFO] 2024-01-01 message here"
	// - "INFO: message here"
	// - "10:00:00 WARN message here"

	// Pattern to match: optional brackets, timestamp (various formats), severity level, separators
	// Then capture everything after as the message
	patterns := []string{
		// ISO datetime + severity + message: "2024-01-01 10:00:00 INFO message"
		`^\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\s+(?:\[)?(?:TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)(?:\])?\s*[:>-]?\s*(.*)$`,

		// Time only + severity: "10:00:00 INFO message"
		`^\d{2}:\d{2}:\d{2}(?:\.\d+)?\s+(?:\[)?(?:TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)(?:\])?\s*[:>-]?\s*(.*)$`,

		// Severity first: "[INFO] message" or "INFO: message"
		`^(?:\[)?(?:TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)(?:\])?\s*[:>-]\s*(.*)$`,

		// Bracketed severity + timestamp: "[INFO] 2024-01-01 message"
		`^\[(?:TRACE|DEBUG|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\]\s+\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?\s+(.*)$`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			message := strings.TrimSpace(matches[1])
			if message != "" {
				return message
			}
		}
	}

	// If no pattern matches, return the original line (might be a pure message)
	return line
}
