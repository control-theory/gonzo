package tui

import (
	"regexp"
	"sort"
	"time"

	"github.com/control-theory/gonzo/internal/ai"
	"github.com/control-theory/gonzo/internal/memory"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Section represents different dashboard sections
type Section int

const (
	SectionWords Section = iota
	SectionAttributes
	SectionDistribution
	SectionCounts
	SectionFilter
	SectionLogs
)

// LogEntry represents a formatted log entry
type LogEntry struct {
	Timestamp     time.Time // Receive time - when we processed this log
	OrigTimestamp time.Time // Original timestamp from the log (if available)
	Severity      string
	Message       string
	RawLine       string
	Attributes    map[string]string
}

// HeatmapMinute represents severity counts for one minute in the heatmap
type HeatmapMinute struct {
	Timestamp time.Time
	Counts    SeverityCounts
}

// PatternCount represents a pattern and its count for a specific severity
type PatternCount struct {
	Pattern string
	Count   int
}

// ServiceCount represents a service and its count for a specific severity  
type ServiceCount struct {
	Service string
	Count   int
}

// DashboardModel represents the main TUI model
type DashboardModel struct {
	// Dashboard state
	width             int
	height            int
	activeSection     Section
	showModal         bool
	modalContent      string
	showHelp          bool
	showPatternsModal bool
	showStatsModal    bool
	showCountsModal   bool

	// Data
	snapshot      *memory.FrequencySnapshot
	logEntries    []LogEntry       // Filtered view for display
	allLogEntries []LogEntry       // Complete unfiltered log buffer
	countsHistory []SeverityCounts // Line counts per interval by severity
	
	// Log Counts Modal Data
	heatmapData      []HeatmapMinute    // Minute-by-minute severity counts for heatmap (60 minute rolling window)
	drain3BySeverity map[string]*Drain3Manager // Separate drain3 instance for each severity
	servicesBySeverity map[string][]ServiceCount // Top services by severity level

	// Configuration
	maxLogBuffer   int
	updateInterval time.Duration

	// Filter
	filterInput  textinput.Model
	filterActive bool
	filterRegex  *regexp.Regexp

	// Search/Highlight
	searchInput  textinput.Model
	searchActive bool
	searchTerm   string // For 's' command - highlights just the term

	// Charts data for rendering
	chartsInitialized bool

	// Selection state
	selectedIndex    map[Section]int
	selectedLogIndex int  // For log section navigation
	viewPaused       bool // Pause view updates when navigating logs
	logAutoScroll    bool // Auto-scroll to latest logs in log viewer

	// Update interval management
	availableIntervals []time.Duration
	currentIntervalIdx int

	// AI Analysis
	aiClient         *ai.OpenAIClient
	aiAnalyzing      bool
	currentLogEntry  *LogEntry // Track current log entry being viewed for AI analysis
	aiAnalysisResult string    // Store the AI analysis result for display
	aiSpinnerFrame   int       // Animation frame for AI spinner

	// AI Status tracking
	aiConfigured   bool   // Whether AI is properly configured
	aiServiceName  string // e.g., "OpenAI", "LM Studio", "Ollama"
	aiModelName    string // The actual model being used
	aiErrorMessage string // Error message if configuration failed

	// Modal viewports for split layout
	infoViewport viewport.Model // Left side: log details, attributes, AI analysis
	chatViewport viewport.Model // Right side: chat history
	modalReady   bool

	// Modal section navigation
	modalActiveSection string // "info" or "chat"

	// Model selection modal
	showModelSelectionModal bool
	selectedModelIndex      int
	availableModelsList     []string

	// Chat functionality
	chatInput        textarea.Model
	chatActive       bool
	chatHistory      []string // Store chat history
	chatAutoScroll   bool     // Whether to auto-scroll chat to bottom
	chatAiAnalyzing  bool     // Whether chat AI is working (separate from info AI)
	chatSpinnerFrame int      // Animation frame for chat spinner

	// Column display
	showColumns bool // Toggle Host and Service columns in log view

	// Drain3 pattern extraction
	drain3Manager       *Drain3Manager
	drain3LastProcessed int // Track last processed log index for drain3
	
	// Statistics tracking
	statsStartTime        time.Time
	statsTotalBytes       int64
	statsPeakLogsPerSec   float64
	statsLastSecond       time.Time
	statsLogsThisSecond   int
	statsTotalLogsEver    int         // Total logs processed (not limited by buffer)
	// Sliding window for real-time rate calculation (last 10 seconds)
	statsRecentCounts     []int       // Count of logs in each second
	statsRecentTimes      []time.Time // Timestamp for each second
	
	// Lifetime statistics (unlimited, not affected by buffer pruning)
	lifetimeSeverityCounts map[string]int64              // Total count per severity level
	lifetimeHostCounts     map[string]int64              // Total count per host
	lifetimeServiceCounts  map[string]int64              // Total count per service
	lifetimeAttrCounts     map[string]int64              // Total count per attribute=value pair
	lifetimeWordCounts     map[string]int64              // Total count per word (for charts)
	lifetimeAttrKeyCounts  map[string]map[string]int64   // Per attribute key: value -> count (for charts)
}

// UpdateMsg contains data updates for the dashboard
type UpdateMsg struct {
	Snapshot         *memory.FrequencySnapshot
	NewLogEntry      *LogEntry
	NewLogBatch      []*LogEntry     // Support for batch updates
	LineCount        int             // Deprecated - use SeverityCount
	SeverityCount    *SeverityCounts // Counts by severity level
	ForceCountUpdate bool            // Force update count history even with 0
	ResetDrain3      bool            // Signal to reset drain3 pattern extraction
}

// TickMsg represents periodic updates
type TickMsg time.Time

// UpdateIntervalMsg represents a request to change update interval
type UpdateIntervalMsg time.Duration

// AIAnalysisMsg represents the result of AI log analysis
type AIAnalysisMsg struct {
	Result string
	Error  error
	IsChat bool // true for chat responses, false for initial analysis
}

// ManualResetMsg represents a manual reset request triggered by user
type ManualResetMsg struct{}

// initializeDrain3BySeverity creates separate drain3 instances for each severity level
func initializeDrain3BySeverity() map[string]*Drain3Manager {
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}
	drain3Map := make(map[string]*Drain3Manager)
	
	for _, severity := range severities {
		drain3Map[severity] = NewDrain3Manager()
	}
	
	return drain3Map
}

// NewDashboardModel creates a new dashboard model
func NewDashboardModel(maxLogBuffer int, updateInterval time.Duration, aiModel string) *DashboardModel {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter logs (regex supported)..."
	filterInput.CharLimit = 100

	searchInput := textinput.New()
	searchInput.Placeholder = "Search and highlight text..."
	searchInput.CharLimit = 100

	chatInput := textarea.New()
	chatInput.Prompt = "> "
	chatInput.Placeholder = "Ask a follow-up question about this log..."
	chatInput.CharLimit = 500 // Increased for multi-line
	chatInput.SetHeight(3)    // Allow 3 lines of input
	chatInput.MaxHeight = 6   // Max 6 lines before scrolling
	chatInput.ShowLineNumbers = false

	// Available update intervals
	availableIntervals := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		5 * time.Second,
		10 * time.Second,
		30 * time.Second,
		1 * time.Minute,
	}

	// Find current interval index
	currentIdx := 1 // Default to 1 second if not found
	for i, interval := range availableIntervals {
		if interval == updateInterval {
			currentIdx = i
			break
		}
	}

	m := &DashboardModel{
		maxLogBuffer:        maxLogBuffer,
		updateInterval:      updateInterval,
		filterInput:         filterInput,
		searchInput:         searchInput,
		chatInput:           chatInput,
		selectedIndex:       make(map[Section]int),
		logEntries:          make([]LogEntry, 0, maxLogBuffer),
		allLogEntries:       make([]LogEntry, 0, maxLogBuffer),
		countsHistory:       make([]SeverityCounts, 0),
		heatmapData:         make([]HeatmapMinute, 0),
		drain3BySeverity:    initializeDrain3BySeverity(),
		servicesBySeverity:  make(map[string][]ServiceCount),
		availableIntervals:  availableIntervals,
		currentIntervalIdx:  currentIdx,
		aiClient:            ai.NewOpenAIClient(aiModel), // Initialize AI client with configurable model
		infoViewport:        viewport.New(80, 20),        // Will be resized later
		chatViewport:        viewport.New(30, 20),        // Will be resized later
		modalActiveSection:  "info",                      // Start with info section active
		chatHistory:         make([]string, 0),
		chatAutoScroll:      true,               // Enable auto-scroll for new messages
		drain3Manager:       NewDrain3Manager(), // Initialize drain3 manager
		drain3LastProcessed: 0,                  // Initialize drain3 tracking
		logAutoScroll:       true,               // Start with auto-scroll enabled
		showColumns:         true,               // Show Host/Service columns by default
		// Initialize statistics tracking
		statsStartTime:        time.Now(),
		statsTotalBytes:       0,
		statsPeakLogsPerSec:   0.0,
		statsLastSecond:       time.Now(),
		statsLogsThisSecond:   0,
		statsTotalLogsEver:    0,
		statsRecentCounts:     make([]int, 0, 10),    // 10 second sliding window
		statsRecentTimes:      make([]time.Time, 0, 10),
		
		// Initialize lifetime statistics
		lifetimeSeverityCounts: make(map[string]int64),
		lifetimeHostCounts:     make(map[string]int64),
		lifetimeServiceCounts:  make(map[string]int64),
		lifetimeAttrCounts:     make(map[string]int64),
		lifetimeWordCounts:     make(map[string]int64),
		lifetimeAttrKeyCounts:  make(map[string]map[string]int64),
	}

	// Initialize AI status based on client validation
	if m.aiClient != nil {
		m.aiConfigured, m.aiErrorMessage, m.aiServiceName, m.aiModelName = m.aiClient.GetValidationStatus()
		// Get available models list for model selection modal
		if m.aiConfigured {
			m.availableModelsList = m.aiClient.AvailableModels
		}
	} else {
		m.aiConfigured = false
		m.aiServiceName = "None"
		m.aiModelName = ""
		m.aiErrorMessage = "No API key configured"
	}

	return m
}

// switchToModel switches the AI client to use a different model
func (m *DashboardModel) switchToModel(newModel string) (tea.Model, tea.Cmd) {
	if m.aiClient == nil {
		return m, nil
	}

	// Update the model in the AI client
	m.aiClient.Model = newModel
	m.aiModelName = newModel

	// Close the model selection modal
	m.showModelSelectionModal = false

	// Update any existing AI analysis result to show the new model
	if m.currentLogEntry != nil {
		m.modalContent = m.formatLogDetails(*m.currentLogEntry, 60)
	}

	return m, nil
}

// Helper functions to convert lifetime statistics to chart-compatible format

// getLifetimeWordEntries returns word entries sorted by count (for dashboard charts)
func (m *DashboardModel) getLifetimeWordEntries() []*memory.FrequencyEntry {
	entries := make([]*memory.FrequencyEntry, 0, len(m.lifetimeWordCounts))
	
	for word, count := range m.lifetimeWordCounts {
		entries = append(entries, &memory.FrequencyEntry{
			Term:  word,
			Count: count,
		})
	}
	
	// Sort by count (descending) then by word (ascending)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Term < entries[j].Term
		}
		return entries[i].Count > entries[j].Count
	})
	
	return entries
}

// getLifetimeAttributeEntries returns attribute entries sorted by unique value count (for dashboard charts)
func (m *DashboardModel) getLifetimeAttributeEntries() []*memory.AttributeStatsEntry {
	entries := make([]*memory.AttributeStatsEntry, 0, len(m.lifetimeAttrKeyCounts))
	
	for key, valueCounts := range m.lifetimeAttrKeyCounts {
		totalCount := int64(0)
		for _, count := range valueCounts {
			totalCount += count
		}
		
		entries = append(entries, &memory.AttributeStatsEntry{
			Key:              key,
			UniqueValueCount: len(valueCounts),
			TotalCount:       totalCount,
			Values:          valueCounts,
		})
	}
	
	// Sort by unique value count (descending) then by key (ascending)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].UniqueValueCount == entries[j].UniqueValueCount {
			return entries[i].Key < entries[j].Key
		}
		return entries[i].UniqueValueCount > entries[j].UniqueValueCount
	})
	
	return entries
}

// Init initializes the model
func (m *DashboardModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textarea.Blink)

	// Enable mouse support
	cmds = append(cmds, func() tea.Msg { return tea.EnableMouseCellMotion() })

	// Set up regular tick for dashboard updates
	cmds = append(cmds, tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	}))

	return tea.Batch(cmds...)
}

// GetCountsHistory returns the current counts history for debugging
func (m *DashboardModel) GetCountsHistory() []SeverityCounts {
	return m.countsHistory
}

// getSpinner returns an animated spinner character based on frame
func (m *DashboardModel) getSpinner() string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸"}
	return spinners[m.aiSpinnerFrame]
}

// getChatSpinner returns an animated spinner character for chat
func (m *DashboardModel) getChatSpinner() string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸"}
	return spinners[m.chatSpinnerFrame]
}
