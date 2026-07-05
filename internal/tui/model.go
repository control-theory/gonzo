package tui

import (
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"time"

	"github.com/control-theory/gonzo/internal/ai"
	"github.com/control-theory/gonzo/internal/memory"
	"github.com/control-theory/gonzo/internal/releases"
	versioncheck "github.com/control-theory/gonzo/internal/version"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// K8sSourceInterface defines the interface for Kubernetes log source
// This allows the TUI to query available namespaces and pods
type K8sSourceInterface interface {
	ListNamespaces() (map[string]bool, error)
	ListPods(selectedNamespaces map[string]bool) (map[string]bool, error)
	UpdateFilter(namespaces []string, selector string, podNames []string) error
}

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

// ColumnConfig represents a configurable column in the log viewer
type ColumnConfig struct {
	Key       string // Attribute key (e.g., "host.name", "k8s.namespace", "timestamp", "severity", "message")
	Label     string // Display label (e.g., "Host", "Namespace")
	Width     int    // Display width in characters (0 means fill remaining space)
	Enabled   bool   // Whether column is visible
	IsDefault bool   // Is this a built-in default column (vs discovered from log attributes)
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
	width                   int
	height                  int
	activeSection           Section
	showModal               bool
	modalContent            string
	showHelp                bool
	showPatternsModal       bool
	showStatsModal          bool
	showCountsModal         bool
	showLogViewerModal      bool
	showSeverityFilterModal bool

	// Data
	snapshot      *memory.FrequencySnapshot
	logEntries    []LogEntry       // Filtered view for display
	allLogEntries []LogEntry       // Complete unfiltered log buffer
	countsHistory []SeverityCounts // Line counts per interval by severity

	// Log Counts Modal Data
	heatmapData        []HeatmapMinute           // Minute-by-minute severity counts for heatmap (60 minute rolling window)
	drain3BySeverity   map[string]*Drain3Manager // Separate drain3 instance for each severity
	servicesBySeverity map[string][]ServiceCount // Top services by severity level

	// Configuration
	maxLogBuffer       int
	updateInterval     time.Duration
	reverseScrollWheel bool
	useLogTime         bool // Use OrigTimestamp instead of Timestamp for heatmap/display

	// Filter
	filterInput  textinput.Model
	filterActive bool
	filterRegex  *regexp.Regexp

	// Search/Highlight
	searchInput  textinput.Model
	searchActive bool
	searchTerm   string // For 's' command - highlights just the term

	// Severity Filter
	severityFilter         map[string]bool // Which severity levels are enabled (true = show, false = hide)
	severityFilterSelected int             // Selected index in severity filter modal
	severityFilterActive   bool            // Whether severity filtering is active (any severity disabled)
	severityFilterOriginal map[string]bool // Original state when modal opened (for ESC cancellation)

	// Kubernetes Filter (requires integration with k8s log source)
	showK8sFilterModal bool               // Whether to show K8s filter modal
	k8sNamespaces      map[string]bool    // Available namespaces and their selection state
	k8sPods            map[string]bool    // Available pods and their selection state
	k8sFilterSelected  int                // Selected index in K8s filter modal
	k8sScrollOffset    int                // Scroll offset for K8s filter modal
	k8sFilterOriginal  map[string]bool    // Original namespace state (for ESC cancellation)
	k8sPodsOriginal    map[string]bool    // Original pod state (for ESC cancellation)
	k8sActiveView      string             // "namespaces" or "pods"
	k8sFilterActive    bool               // Whether K8s filtering is currently active
	k8sSource          K8sSourceInterface // Reference to K8s source for listing namespaces/pods

	// Charts data for rendering
	chartsInitialized bool

	// Selection state
	selectedIndex            map[Section]int
	selectedLogIndex         int  // For log section navigation
	viewPaused               bool // Pause view updates when navigating logs
	logAutoScroll            bool // Auto-scroll to latest logs in log viewer
	instructionsScrollOffset int  // Scroll position for instructions/filter status screen

	// Modal display options
	attributeWrappingEnabled bool // Whether to wrap attribute values instead of truncating them

	// Update interval management
	availableIntervals []time.Duration
	currentIntervalIdx int

	// AI Analysis
	aiClient         ai.Client
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

	// Clipboard feedback (transient status message shown after a copy action)
	copyFeedback       string
	copyFeedbackExpiry time.Time

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

	// Column customization
	showColumnConfigModal    bool            // Whether to show column config modal
	columnConfigSelected     int             // Selected index in column config modal
	columnConfigScrollOffset int             // Scroll offset for column config modal
	columnConfigOriginal     []ColumnConfig  // Original state for ESC cancellation
	activeColumns            []ColumnConfig  // Currently enabled columns in display order
	availableColumns         []ColumnConfig  // All available columns (defaults + discovered)
	discoveredAttributes     map[string]bool // Track all unique attribute keys seen across logs
	columnMaxWidths          map[string]int  // Tracks the max observed width for each column (never shrinks)
	columnWidthLimitEnabled  bool            // Whether the MaxColumnWidth cap is applied
	logViewHorizontalOffset  int             // Horizontal scroll offset for log viewer
	autoSwitchedToK8sMode    bool            // Track if we've already auto-switched to k8s columns

	// Drain3 pattern extraction
	drain3Manager       *Drain3Manager
	drain3LastProcessed int // Track last processed log index for drain3

	// Statistics tracking
	statsStartTime      time.Time
	statsTotalBytes     int64
	statsPeakLogsPerSec float64
	statsLastSecond     time.Time
	statsLogsThisSecond int
	statsTotalLogsEver  int // Total logs processed (not limited by buffer)
	// Sliding window for real-time rate calculation (last 10 seconds)
	statsRecentCounts []int       // Count of logs in each second
	statsRecentTimes  []time.Time // Timestamp for each second

	// Lifetime statistics (unlimited, not affected by buffer pruning)
	lifetimeSeverityCounts map[string]int64            // Total count per severity level
	lifetimeHostCounts     map[string]int64            // Total count per host
	lifetimeServiceCounts  map[string]int64            // Total count per service
	lifetimeAttrCounts     map[string]int64            // Total count per attribute=value pair
	lifetimeWordCounts     map[string]int64            // Total count per word (for charts)
	lifetimeAttrKeyCounts  map[string]map[string]int64 // Per attribute key: value -> count (for charts)
	stopWords              map[string]bool             // Stop words to filter from word counting

	// Version checking
	versionChecker *versioncheck.Checker // Version checker for update notifications

	// Web dashboard
	webPort int // Port for the Dstl8 Lite web dashboard (for browser open shortcut)

	// What's New modal
	showWhatsNewModal      bool
	releasesFetcher        *releases.Fetcher
	currentVersion         string
	lastSeenVersion        string
	whatsNewCheckDone      bool   // true after we've checked whether to auto-show
	whatsNewRetries        int    // retry counter for waiting on background fetch
	whatsNewRenderedCache  string // pre-rendered content (glamour is expensive)
	whatsNewCacheWidth     int    // width the cache was rendered at
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

// InitializeDrain3BySeverity creates separate drain3 instances for each severity level
func InitializeDrain3BySeverity() map[string]*Drain3Manager {
	severities := []string{"FATAL", "ERROR", "WARN", "INFO", "DEBUG", "TRACE", "UNKNOWN"}
	drain3Map := make(map[string]*Drain3Manager)

	for _, severity := range severities {
		drain3Map[severity] = NewDrain3Manager()
	}

	return drain3Map
}

// NewDashboardModel creates a new dashboard model with stop words
func NewDashboardModel(maxLogBuffer int, updateInterval time.Duration, aiProvider string, aiModel string, stopWords map[string]bool, reverseScrollWheel bool, useLogTime bool) *DashboardModel {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter logs by message or attributes (regex supported)..."
	filterInput.CharLimit = 200

	searchInput := textinput.New()
	searchInput.Placeholder = "Search and highlight text..."
	searchInput.CharLimit = 200

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
		reverseScrollWheel:  reverseScrollWheel,
		useLogTime:          useLogTime,
		filterInput:         filterInput,
		searchInput:         searchInput,
		chatInput:           chatInput,
		selectedIndex:       make(map[Section]int),
		logEntries:          make([]LogEntry, 0, maxLogBuffer),
		allLogEntries:       make([]LogEntry, 0, maxLogBuffer),
		countsHistory:       make([]SeverityCounts, 0),
		heatmapData:         make([]HeatmapMinute, 0),
		drain3BySeverity:    InitializeDrain3BySeverity(),
		servicesBySeverity:  make(map[string][]ServiceCount),
		availableIntervals: availableIntervals,
		currentIntervalIdx: currentIdx,
		infoViewport:       viewport.New(80, 20), // Will be resized later
		chatViewport:        viewport.New(30, 20),        // Will be resized later
		modalActiveSection:  "info",                      // Start with info section active
		chatHistory:         make([]string, 0),
		chatAutoScroll:      true,               // Enable auto-scroll for new messages
		drain3Manager:       NewDrain3Manager(), // Initialize drain3 manager
		drain3LastProcessed: 0,                  // Initialize drain3 tracking
		logAutoScroll:       true,               // Start with auto-scroll enabled
		showColumns:         true,               // Show Host/Service columns by default
		// Initialize column customization
		discoveredAttributes:     make(map[string]bool),
		columnMaxWidths:          make(map[string]int),
		columnWidthLimitEnabled:  true, // Enable 100-char column width cap by default
		activeColumns:            getDefaultActiveColumns(),
		availableColumns:         getDefaultAvailableColumns(),
		columnConfigSelected:     0,
		logViewHorizontalOffset:  0,     // Start with no horizontal scroll
		instructionsScrollOffset: 0,     // Start at top of instructions
		attributeWrappingEnabled: false, // Default to truncating (not wrapping)
		// Initialize statistics tracking
		statsStartTime:      time.Now(),
		statsTotalBytes:     0,
		statsPeakLogsPerSec: 0.0,
		statsLastSecond:     time.Now(),
		statsLogsThisSecond: 0,
		statsTotalLogsEver:  0,
		statsRecentCounts:   make([]int, 0, 10), // 10 second sliding window
		statsRecentTimes:    make([]time.Time, 0, 10),

		// Initialize lifetime statistics
		lifetimeSeverityCounts: make(map[string]int64),
		lifetimeHostCounts:     make(map[string]int64),
		lifetimeServiceCounts:  make(map[string]int64),
		lifetimeAttrCounts:     make(map[string]int64),
		lifetimeWordCounts:     make(map[string]int64),
		lifetimeAttrKeyCounts:  make(map[string]map[string]int64),
		stopWords:              stopWords,

		// Initialize severity filter (all levels enabled by default)
		severityFilter: map[string]bool{
			"TRACE":    true,
			"DEBUG":    true,
			"INFO":     true,
			"WARN":     true,
			"ERROR":    true,
			"FATAL":    true,
			"CRITICAL": true,
			"UNKNOWN":  true,
		},
		severityFilterSelected: 0,
		severityFilterActive:   false,
		severityFilterOriginal: make(map[string]bool), // Initialize empty map for modal state backup
	}

	// Initialize AI client via factory
	aiClient, err := ai.NewClient(ai.ProviderType(aiProvider), aiModel)
	if err != nil {
		// Provider error (e.g., invalid provider or missing API key when explicitly requested)
		m.aiClient = nil
		m.aiConfigured = false
		m.aiServiceName = "None"
		m.aiModelName = ""
		m.aiErrorMessage = err.Error()
	} else if aiClient != nil {
		m.aiClient = aiClient
		status := aiClient.GetValidationStatus()
		m.aiConfigured = status.Validated
		m.aiErrorMessage = status.ErrorMessage
		m.aiServiceName = status.ServiceName
		m.aiModelName = status.ModelName
		// Get available models list for model selection modal
		if m.aiConfigured {
			m.availableModelsList = aiClient.CachedModels()
		}
	} else {
		// nil client, nil error = AI disabled (no API key in auto mode)
		m.aiClient = nil
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
	m.aiClient.SetModel(newModel)
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
			Values:           valueCounts,
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

// SetVersionChecker sets the version checker for update notifications
func (m *DashboardModel) SetVersionChecker(checker *versioncheck.Checker) {
	m.versionChecker = checker
}

// SetWebPort sets the web dashboard port for the browser open shortcut
func (m *DashboardModel) SetWebPort(port int) {
	m.webPort = port
}

// SetReleasesFetcher sets the releases fetcher for the what's-new modal
func (m *DashboardModel) SetReleasesFetcher(f *releases.Fetcher) {
	m.releasesFetcher = f
}

// SetCurrentVersion sets the current app version for the what's-new modal
func (m *DashboardModel) SetCurrentVersion(v string) {
	m.currentVersion = v
}

// SetLastSeenVersion sets the last version the user has seen
func (m *DashboardModel) SetLastSeenVersion(v string) {
	m.lastSeenVersion = v
}

// openBrowser opens the given URL in the default browser
func (m *DashboardModel) openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

// SetK8sSource sets the Kubernetes log source for the dashboard
func (m *DashboardModel) SetK8sSource(source K8sSourceInterface) {
	m.k8sSource = source
}

// isK8sMode returns true if any logs have k8s attributes (namespace or pod)
func (m *DashboardModel) isK8sMode() bool {
	// Check the most recent log entries for k8s attributes
	// We only need to check a few entries to determine mode
	checkCount := min(10, len(m.logEntries))
	for i := len(m.logEntries) - checkCount; i < len(m.logEntries); i++ {
		if i < 0 {
			continue
		}
		entry := m.logEntries[i]
		if entry.Attributes["k8s.namespace"] != "" || entry.Attributes["k8s.pod"] != "" {
			return true
		}
	}
	return false
}

// toggleColumnsMode switches between normal (Host/Service) and k8s (Namespace/Pod) column modes
func (m *DashboardModel) toggleColumnsMode() {
	if m.isK8sMode() {
		// In k8s mode: swap between k8s columns and host/service columns
		hasK8sColumns := false
		hasNormalColumns := false

		for _, col := range m.activeColumns {
			if col.Key == "k8s.namespace" || col.Key == "k8s.pod" {
				hasK8sColumns = true
			}
			if col.Key == "host.name" || col.Key == "service.name" {
				hasNormalColumns = true
			}
		}

		// Toggle: if showing k8s columns, switch to normal; otherwise switch to k8s
		newColumns := []ColumnConfig{}
		if hasK8sColumns && !hasNormalColumns {
			// Switch to normal mode (Host/Service)
			for _, col := range m.activeColumns {
				if col.Key == "k8s.namespace" {
					// Replace with host.name
					newColumns = append(newColumns, ColumnConfig{
						Key: "host.name", Label: "Host", Width: 12, Enabled: true, IsDefault: true,
					})
				} else if col.Key == "k8s.pod" {
					// Replace with service.name
					newColumns = append(newColumns, ColumnConfig{
						Key: "service.name", Label: "Service", Width: 16, Enabled: true, IsDefault: true,
					})
				} else {
					newColumns = append(newColumns, col)
				}
			}
		} else {
			// Switch to k8s mode (Namespace/Pod)
			for _, col := range m.activeColumns {
				if col.Key == "host.name" {
					// Replace with k8s.namespace
					newColumns = append(newColumns, ColumnConfig{
						Key: "k8s.namespace", Label: "Namespace", Width: 20, Enabled: true, IsDefault: true,
					})
				} else if col.Key == "service.name" {
					// Replace with k8s.pod
					newColumns = append(newColumns, ColumnConfig{
						Key: "k8s.pod", Label: "Pod", Width: 20, Enabled: true, IsDefault: true,
					})
				} else {
					newColumns = append(newColumns, col)
				}
			}
		}
		m.activeColumns = newColumns

		// Also update availableColumns to reflect the change
		for i := range m.availableColumns {
			if m.availableColumns[i].Key == "host.name" || m.availableColumns[i].Key == "service.name" {
				m.availableColumns[i].Enabled = hasK8sColumns
			}
			if m.availableColumns[i].Key == "k8s.namespace" || m.availableColumns[i].Key == "k8s.pod" {
				m.availableColumns[i].Enabled = !hasK8sColumns
			}
		}
	} else {
		// Not in k8s mode: just toggle showColumns visibility
		m.showColumns = !m.showColumns
	}
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

// getDefaultActiveColumns returns the default columns that are shown initially
func getDefaultActiveColumns() []ColumnConfig {
	return []ColumnConfig{
		{Key: "timestamp", Label: "Time", Width: 8, Enabled: true, IsDefault: true},
		{Key: "severity", Label: "Level", Width: 5, Enabled: true, IsDefault: true},
		{Key: "host.name", Label: "Host", Width: 12, Enabled: true, IsDefault: true},
		{Key: "service.name", Label: "Service", Width: 16, Enabled: true, IsDefault: true},
		{Key: "message", Label: "Message", Width: 0, Enabled: true, IsDefault: true}, // Width 0 = fill remaining
	}
}

// getDefaultAvailableColumns returns all default columns (including K8s ones that start disabled)
func getDefaultAvailableColumns() []ColumnConfig {
	return []ColumnConfig{
		{Key: "timestamp", Label: "Time", Width: 8, Enabled: true, IsDefault: true},
		{Key: "severity", Label: "Level", Width: 5, Enabled: true, IsDefault: true},
		{Key: "host.name", Label: "Host", Width: 12, Enabled: true, IsDefault: true},
		{Key: "service.name", Label: "Service", Width: 16, Enabled: true, IsDefault: true},
		{Key: "k8s.namespace", Label: "Namespace", Width: 20, Enabled: false, IsDefault: true},
		{Key: "k8s.pod", Label: "Pod", Width: 20, Enabled: false, IsDefault: true},
		{Key: "message", Label: "Message", Width: 0, Enabled: true, IsDefault: true},
	}
}
