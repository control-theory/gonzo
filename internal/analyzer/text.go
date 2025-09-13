package analyzer

import (
	"regexp"
	"strings"

	"github.com/control-theory/gonzo/internal/timestamp"
)

type TextAnalyzer struct {
	minWordLength   int
	maxPhraseLength int
	wordPattern     *regexp.Regexp
	timestampParser *timestamp.Parser
}

type AnalysisResult struct {
	Words   []string
	Phrases []string
}

func NewTextAnalyzer() *TextAnalyzer {
	return &TextAnalyzer{
		minWordLength:   3,
		maxPhraseLength: 4,
		wordPattern:     regexp.MustCompile(`[a-zA-Z_][a-zA-Z0-9_]*`),
		timestampParser: timestamp.NewParser(),
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

	for _, word := range words {
		if len(word) >= ta.minWordLength && !stopWords[word] {
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
func (ta *TextAnalyzer) extractMessage(line string) string {
	return ta.timestampParser.ExtractLogMessage(line)
}
