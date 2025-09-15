// Package output provides output formatting for logs.
package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/memory"
)

// StdoutFormatter formats log analysis output for standard output.
type StdoutFormatter struct {
	processedLines        int64
	intervalProcessedLines int64
}

// NewStdoutFormatter creates a new stdout formatter.
func NewStdoutFormatter() *StdoutFormatter {
	return &StdoutFormatter{}
}

// RecordLineProcessed increments the line processing counters.
func (sf *StdoutFormatter) RecordLineProcessed() {
	sf.processedLines++
	sf.intervalProcessedLines++
}

// PrintMetrics displays frequency analysis metrics to stdout.
func (sf *StdoutFormatter) PrintMetrics(snapshot *memory.FrequencySnapshot) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	fmt.Printf("\n═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("📊 Log Analysis Report - %s\n", timestamp)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("📈 Total Lines Processed: %d\n", sf.processedLines)
	fmt.Printf("📊 Lines This Interval: %d\n", sf.intervalProcessedLines)
	fmt.Printf("📝 Unique Words This Interval: %d\n", len(snapshot.Words))
	fmt.Printf("🔗 Unique Phrases This Interval: %d\n", len(snapshot.Phrases))
	fmt.Printf("🔑 Unique Attribute Keys: %d\n", len(snapshot.Attributes))
	fmt.Printf("\n")

	if len(snapshot.Words) > 0 {
		fmt.Printf("🔤 TOP WORDS BY FREQUENCY:\n")
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		topWords := 15
		if len(snapshot.Words) < topWords {
			topWords = len(snapshot.Words)
		}
		
		for i := 0; i < topWords; i++ {
			entry := snapshot.Words[i]
			bar := sf.createBar(entry.Count, snapshot.Words[0].Count, 20)
			fmt.Printf("%2d. %-15s │%s│ %d\n", 
				i+1, entry.Term, bar, entry.Count)
		}
		fmt.Printf("\n")
	}

	if len(snapshot.Phrases) > 0 {
		fmt.Printf("🔗 TOP PHRASES BY FREQUENCY:\n")
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		topPhrases := 10
		if len(snapshot.Phrases) < topPhrases {
			topPhrases = len(snapshot.Phrases)
		}
		
		for i := 0; i < topPhrases; i++ {
			entry := snapshot.Phrases[i]
			bar := sf.createBar(entry.Count, snapshot.Phrases[0].Count, 15)
			maxLen := 30
			phrase := entry.Term
			if len(phrase) > maxLen {
				phrase = phrase[:maxLen-3] + "..."
			}
			fmt.Printf("%2d. %-33s │%s│ %d\n", 
				i+1, phrase, bar, entry.Count)
		}
		fmt.Printf("\n")
	}

	if len(snapshot.Attributes) > 0 {
		fmt.Printf("🔑 TOP ATTRIBUTE KEYS BY UNIQUE VALUE COUNT:\n")
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		topAttributes := 10
		if len(snapshot.Attributes) < topAttributes {
			topAttributes = len(snapshot.Attributes)
		}
		
		for i := 0; i < topAttributes; i++ {
			entry := snapshot.Attributes[i]
			var maxUniqueCount int
			for _, attr := range snapshot.Attributes {
				if attr.UniqueValueCount > maxUniqueCount {
					maxUniqueCount = attr.UniqueValueCount
				}
			}
			bar := sf.createBar(int64(entry.UniqueValueCount), int64(maxUniqueCount), 15)
			maxLen := 30
			key := entry.Key
			if len(key) > maxLen {
				key = key[:maxLen-3] + "..."
			}
			fmt.Printf("%2d. %-33s │%s│ %d unique values (%d total)\n", 
				i+1, key, bar, entry.UniqueValueCount, entry.TotalCount)
		}
		fmt.Printf("\n")
	}

	if len(snapshot.Words) > 0 {
		fmt.Printf("📊 FREQUENCY DISTRIBUTION:\n")
		fmt.Printf("───────────────────────────────────────────────────────────────\n")
		sf.printFrequencyDistribution(snapshot.Words)
		fmt.Printf("\n")
	}

	fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")
}

// ResetInterval resets the interval line counter.
func (sf *StdoutFormatter) ResetInterval() {
	sf.intervalProcessedLines = 0
}

func (sf *StdoutFormatter) createBar(count, maxCount int64, width int) string {
	if maxCount == 0 {
		return strings.Repeat(" ", width)
	}
	
	filled := int((float64(count) / float64(maxCount)) * float64(width))
	if filled == 0 && count > 0 {
		filled = 1
	}
	
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return bar
}

func (sf *StdoutFormatter) printFrequencyDistribution(words []*memory.FrequencyEntry) {
	ranges := []struct {
		min, max int64
		label    string
	}{
		{1, 1, "1 occurrence"},
		{2, 5, "2-5 occurrences"},
		{6, 10, "6-10 occurrences"},
		{11, 25, "11-25 occurrences"},
		{26, 100, "26-100 occurrences"},
		{101, 9999999, "100+ occurrences"},
	}

	distribution := make([]int, len(ranges))
	
	for _, word := range words {
		for i, r := range ranges {
			if word.Count >= r.min && word.Count <= r.max {
				distribution[i]++
				break
			}
		}
	}

	maxCount := 0
	for _, count := range distribution {
		if count > maxCount {
			maxCount = count
		}
	}

	for i, r := range ranges {
		count := distribution[i]
		if count > 0 {
			bar := sf.createBar(int64(count), int64(maxCount), 20)
			fmt.Printf("%-20s │%s│ %d words\n", r.label, bar, count)
		}
	}
}