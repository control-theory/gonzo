// Package memory provides in-memory storage for frequency analysis.
package memory

import (
	"sort"
	"sync"
	"time"
)

// FrequencyEntry represents a term with its occurrence statistics.
type FrequencyEntry struct {
	Term      string
	Count     int64
	LastSeen  time.Time
	FirstSeen time.Time
}

// AttributeStats tracks statistics for log attributes.
type AttributeStats struct {
	Key          string
	UniqueValues map[string]int64
	TotalCount   int64
	LastSeen     time.Time
	FirstSeen    time.Time
}

// AttributeStatsEntry represents aggregated statistics for a single attribute.
type AttributeStatsEntry struct {
	Key               string
	UniqueValueCount  int
	TotalCount        int64
	LastSeen          time.Time
	FirstSeen         time.Time
	Values           map[string]int64 // Individual values and their counts
}

// FrequencyMemory provides thread-safe storage for frequency analysis of words, phrases, and attributes.
type FrequencyMemory struct {
	words      map[string]*FrequencyEntry
	phrases    map[string]*FrequencyEntry
	attributes map[string]*AttributeStats
	mutex      sync.RWMutex
	maxSize    int
}

// FrequencySnapshot represents a point-in-time view of frequency data.
type FrequencySnapshot struct {
	Words      []*FrequencyEntry
	Phrases    []*FrequencyEntry
	Attributes []*AttributeStatsEntry
}

// NewFrequencyMemory creates a new FrequencyMemory with the specified maximum size.
func NewFrequencyMemory(maxSize int) *FrequencyMemory {
	return &FrequencyMemory{
		words:      make(map[string]*FrequencyEntry),
		phrases:    make(map[string]*FrequencyEntry),
		attributes: make(map[string]*AttributeStats),
		maxSize:    maxSize,
	}
}

// AddWords adds words to the frequency memory with occurrence tracking.
func (fm *FrequencyMemory) AddWords(words []string) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	
	now := time.Now()
	for _, word := range words {
		if entry, exists := fm.words[word]; exists {
			entry.Count++
			entry.LastSeen = now
		} else {
			fm.words[word] = &FrequencyEntry{
				Term:      word,
				Count:     1,
				FirstSeen: now,
				LastSeen:  now,
			}
		}
	}
	
	if len(fm.words) > fm.maxSize {
		fm.pruneWords()
	}
}

// AddPhrases adds phrases to the frequency memory with occurrence tracking.
func (fm *FrequencyMemory) AddPhrases(phrases []string) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	
	now := time.Now()
	for _, phrase := range phrases {
		if entry, exists := fm.phrases[phrase]; exists {
			entry.Count++
			entry.LastSeen = now
		} else {
			fm.phrases[phrase] = &FrequencyEntry{
				Term:      phrase,
				Count:     1,
				FirstSeen: now,
				LastSeen:  now,
			}
		}
	}
	
	if len(fm.phrases) > fm.maxSize {
		fm.prunePhrases()
	}
}

// AddAttributes adds log attributes to the frequency memory with value tracking.
func (fm *FrequencyMemory) AddAttributes(attributes map[string]string) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	
	now := time.Now()
	for key, value := range attributes {
		if stats, exists := fm.attributes[key]; exists {
			stats.TotalCount++
			stats.LastSeen = now
			stats.UniqueValues[value]++
		} else {
			fm.attributes[key] = &AttributeStats{
				Key:          key,
				UniqueValues: map[string]int64{value: 1},
				TotalCount:   1,
				FirstSeen:    now,
				LastSeen:     now,
			}
		}
	}
	
	if len(fm.attributes) > fm.maxSize {
		fm.pruneAttributes()
	}
}

// GetSnapshot returns a snapshot of the current frequency data.
func (fm *FrequencyMemory) GetSnapshot() *FrequencySnapshot {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()
	
	words := make([]*FrequencyEntry, 0, len(fm.words))
	for _, entry := range fm.words {
		words = append(words, &FrequencyEntry{
			Term:      entry.Term,
			Count:     entry.Count,
			FirstSeen: entry.FirstSeen,
			LastSeen:  entry.LastSeen,
		})
	}
	
	phrases := make([]*FrequencyEntry, 0, len(fm.phrases))
	for _, entry := range fm.phrases {
		phrases = append(phrases, &FrequencyEntry{
			Term:      entry.Term,
			Count:     entry.Count,
			FirstSeen: entry.FirstSeen,
			LastSeen:  entry.LastSeen,
		})
	}
	
	sort.Slice(words, func(i, j int) bool {
		if words[i].Count == words[j].Count {
			return words[i].Term < words[j].Term // Secondary sort alphabetically
		}
		return words[i].Count > words[j].Count
	})
	
	sort.Slice(phrases, func(i, j int) bool {
		if phrases[i].Count == phrases[j].Count {
			return phrases[i].Term < phrases[j].Term // Secondary sort alphabetically
		}
		return phrases[i].Count > phrases[j].Count
	})
	
	attributes := make([]*AttributeStatsEntry, 0, len(fm.attributes))
	for _, stats := range fm.attributes {
		// Copy the values map to include individual value counts
		valuesCopy := make(map[string]int64)
		for key, count := range stats.UniqueValues {
			valuesCopy[key] = count
		}
		
		attributes = append(attributes, &AttributeStatsEntry{
			Key:               stats.Key,
			UniqueValueCount:  len(stats.UniqueValues),
			TotalCount:        stats.TotalCount,
			FirstSeen:         stats.FirstSeen,
			LastSeen:          stats.LastSeen,
			Values:           valuesCopy,
		})
	}
	
	sort.Slice(attributes, func(i, j int) bool {
		if attributes[i].UniqueValueCount == attributes[j].UniqueValueCount {
			return attributes[i].Key < attributes[j].Key // Secondary sort alphabetically
		}
		return attributes[i].UniqueValueCount > attributes[j].UniqueValueCount
	})
	
	return &FrequencySnapshot{
		Words:      words,
		Phrases:    phrases,
		Attributes: attributes,
	}
}

func (fm *FrequencyMemory) pruneWords() {
	entries := make([]*FrequencyEntry, 0, len(fm.words))
	for _, entry := range fm.words {
		entries = append(entries, entry)
	}
	
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Term < entries[j].Term // Secondary sort alphabetically
		}
		return entries[i].Count > entries[j].Count
	})
	
	fm.words = make(map[string]*FrequencyEntry)
	keepCount := fm.maxSize * 3 / 4
	for i := 0; i < keepCount && i < len(entries); i++ {
		fm.words[entries[i].Term] = entries[i]
	}
}

func (fm *FrequencyMemory) prunePhrases() {
	entries := make([]*FrequencyEntry, 0, len(fm.phrases))
	for _, entry := range fm.phrases {
		entries = append(entries, entry)
	}
	
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Term < entries[j].Term // Secondary sort alphabetically
		}
		return entries[i].Count > entries[j].Count
	})
	
	fm.phrases = make(map[string]*FrequencyEntry)
	keepCount := fm.maxSize * 3 / 4
	for i := 0; i < keepCount && i < len(entries); i++ {
		fm.phrases[entries[i].Term] = entries[i]
	}
}

func (fm *FrequencyMemory) pruneAttributes() {
	entries := make([]*AttributeStatsEntry, 0, len(fm.attributes))
	for _, stats := range fm.attributes {
		// Copy the values map 
		valuesCopy := make(map[string]int64)
		for key, count := range stats.UniqueValues {
			valuesCopy[key] = count
		}
		
		entries = append(entries, &AttributeStatsEntry{
			Key:               stats.Key,
			UniqueValueCount:  len(stats.UniqueValues),
			TotalCount:        stats.TotalCount,
			FirstSeen:         stats.FirstSeen,
			LastSeen:          stats.LastSeen,
			Values:           valuesCopy,
		})
	}
	
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].UniqueValueCount == entries[j].UniqueValueCount {
			return entries[i].Key < entries[j].Key // Secondary sort alphabetically
		}
		return entries[i].UniqueValueCount > entries[j].UniqueValueCount
	})
	
	fm.attributes = make(map[string]*AttributeStats)
	keepCount := fm.maxSize * 3 / 4
	for i := 0; i < keepCount && i < len(entries); i++ {
		entry := entries[i]
		// Reconstruct the original stats with the preserved unique values
		fm.attributes[entry.Key] = &AttributeStats{
			Key:          entry.Key,
			UniqueValues: entry.Values,
			TotalCount:   entry.TotalCount,
			FirstSeen:    entry.FirstSeen,
			LastSeen:     entry.LastSeen,
		}
	}
}

// Reset clears all stored frequency data.
func (fm *FrequencyMemory) Reset() {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	
	fm.words = make(map[string]*FrequencyEntry)
	fm.phrases = make(map[string]*FrequencyEntry)
	fm.attributes = make(map[string]*AttributeStats)
}