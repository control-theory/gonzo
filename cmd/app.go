package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/analyzer"
	"github.com/control-theory/gonzo/internal/filereader"
	"github.com/control-theory/gonzo/internal/memory"
	"github.com/control-theory/gonzo/internal/otlplog"
	"github.com/control-theory/gonzo/internal/otlpreceiver"
	"github.com/control-theory/gonzo/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// runApp initializes and runs the application
func runApp(cmd *cobra.Command, args []string) error {
	// Check if version flag was used
	if v, _ := cmd.Flags().GetBool("version"); v {
		versionCmd.Run(cmd, args)
		return nil
	}

	// Initialize components
	formatDetector := otlplog.NewFormatDetector()
	logConverter := otlplog.NewLogConverter()
	textAnalyzer := analyzer.NewTextAnalyzer()
	otlpAnalyzer := analyzer.NewOTLPAnalyzer()
	freqMemory := memory.NewFrequencyMemory(cfg.MemorySize)

	// Initialize TUI model with components
	tuiModel := &simpleTuiModel{
		formatDetector: formatDetector,
		logConverter:   logConverter,
		textAnalyzer:   textAnalyzer,
		otlpAnalyzer:   otlpAnalyzer,
		freqMemory:     freqMemory,
		dashboard:      tui.NewDashboardModel(cfg.LogBuffer, cfg.UpdateInterval, cfg.AIModel),
		updateInterval: cfg.UpdateInterval,
		testMode:       cfg.TestMode,
	}

	var p *tea.Program
	if cfg.TestMode {
		// Test mode - no TTY requirements
		p = tea.NewProgram(tuiModel, tea.WithInput(nil), tea.WithOutput(os.Stdout))
	} else {
		// Normal mode with manual screen management
		p = tea.NewProgram(tuiModel, tea.WithAltScreen(), tea.WithMouseCellMotion())
	}

	// No manual cleanup needed - Bubble Tea handles it

	// Create cancellable context for the application
	ctx, cancel := context.WithCancel(context.Background())
	tuiModel.ctx = ctx
	tuiModel.cancelFunc = cancel

	if _, err := p.Run(); err != nil {
		if strings.Contains(err.Error(), "TTY") || strings.Contains(err.Error(), "/dev/tty") {
			return fmt.Errorf("TUI requires a real terminal. Try --test-mode for non-interactive testing")
		}
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// Message types for bubbletea
type (
	logLineMsg  string
	snapshotMsg *memory.FrequencySnapshot
	finishedMsg struct{}
	tickMsg     struct {
		time     time.Time
		sequence int
	}
)

// simpleTuiModel is the main TUI model that handles everything internally
type simpleTuiModel struct {
	formatDetector *otlplog.FormatDetector
	logConverter   *otlplog.LogConverter
	textAnalyzer   *analyzer.TextAnalyzer
	otlpAnalyzer   *analyzer.OTLPAnalyzer
	freqMemory     *memory.FrequencyMemory
	dashboard      *tui.DashboardModel
	updateInterval time.Duration
	testMode       bool
	ctx            context.Context
	cancelFunc     context.CancelFunc

	// Internal state
	finished       bool
	logCount       int
	severityCounts *tui.SeverityCounts // Track severity counts for current interval
	lastFreqReset  time.Time           // Track when frequency memory was last reset
	timerSequence  int                 // Track timer sequence to avoid concurrent timers
	hasStdinData   bool                // Whether stdin has data available

	// File reading support
	fileReader   *filereader.FileReader // File reader for file input mode
	inputChan    chan string            // Unified input channel (from stdin or files)
	hasFileInput bool                   // Whether we're reading from files

	// OTLP receiver support
	otlpReceiver *otlpreceiver.Receiver // OTLP receiver for network input
	hasOTLPInput bool                   // Whether we're receiving OTLP data

	// JSON accumulation for multi-line OTLP support
	jsonBuffer   strings.Builder // Buffer for accumulating multi-line JSON
	jsonDepth    int             // Track JSON object/array nesting depth
	inJsonObject bool            // Whether we're currently accumulating a JSON object
}

// Init initializes the TUI model
func (m *simpleTuiModel) Init() tea.Cmd {
	// Initialize severity counts
	m.severityCounts = &tui.SeverityCounts{}

	// Initialize frequency reset timer
	m.lastFreqReset = time.Now()

	// Check if OTLP receiver is enabled
	if cfg.OTLPEnabled {
		// OTLP input mode
		m.hasOTLPInput = true
		m.inputChan = make(chan string, 100)

		// Create and start OTLP receiver
		m.otlpReceiver = otlpreceiver.NewReceiver(cfg.OTLPGRPCPort, cfg.OTLPHTTPPort)
		if err := m.otlpReceiver.Start(); err != nil {
			log.Printf("Error starting OTLP receiver: %v", err)
			// Fall back to other input methods if OTLP fails
			m.hasOTLPInput = false
		} else {
			// Start reading from OTLP receiver in the background
			go m.readOTLPAsync()
		}
	}

	// Check if we have file inputs specified (only if OTLP is not enabled)
	if !m.hasOTLPInput && len(cfg.Files) > 0 {
		// File input mode
		m.hasFileInput = true
		m.inputChan = make(chan string, 100)

		// Create file reader
		var err error
		m.fileReader, err = filereader.New(cfg.Files, cfg.Follow)
		if err != nil {
			log.Printf("Error setting up file reader: %v", err)
			// Fall back to stdin if file reading fails
			m.hasFileInput = false
		} else {
			// Start file reading in the background
			go m.readFilesAsync()
		}
	}

	// If no OTLP, no file input or file input failed, check stdin
	if !m.hasOTLPInput && !m.hasFileInput {
		// Check if stdin has data available (not a terminal)
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// stdin is a pipe or file, we have data
			m.hasStdinData = true
			m.inputChan = make(chan string, 100)

			// Start goroutine to read stdin without blocking
			go m.readStdinAsync()
		}
	}

	// Start the dashboard
	dashboardCmd := m.dashboard.Init()

	var cmds []tea.Cmd
	cmds = append(cmds, dashboardCmd)
	cmds = append(cmds, m.periodicUpdate())

	// Start checking for input data if we have any input source
	if m.hasStdinData || m.hasFileInput || m.hasOTLPInput {
		cmds = append(cmds, m.checkInputChannel())
	}

	return tea.Batch(cmds...)
}

// readOTLPAsync reads from the OTLP receiver
func (m *simpleTuiModel) readOTLPAsync() {
	defer close(m.inputChan)

	if m.otlpReceiver == nil {
		return
	}

	// Get the channel from OTLP receiver
	otlpLineChan := m.otlpReceiver.GetLineChan()

	// Forward lines from OTLP receiver to input channel
	for {
		select {
		case <-m.ctx.Done():
			m.otlpReceiver.Stop()
			return
		case line, ok := <-otlpLineChan:
			if !ok {
				// OTLP receiver finished
				return
			}
			if line != "" {
				select {
				case m.inputChan <- line:
				case <-m.ctx.Done():
					return
				}
			}
		}
	}
}

// readFilesAsync reads from files using the FileReader
func (m *simpleTuiModel) readFilesAsync() {
	defer close(m.inputChan)

	if m.fileReader == nil {
		return
	}

	// Start the file reader and get the channel
	fileLineChan := m.fileReader.Start()

	// Forward lines from file reader to input channel
	for {
		select {
		case <-m.ctx.Done():
			m.fileReader.Stop()
			return
		case line, ok := <-fileLineChan:
			if !ok {
				// File reader finished
				return
			}
			if line != "" {
				select {
				case m.inputChan <- line:
				case <-m.ctx.Done():
					return
				}
			}
		}
	}
}

// readStdinAsync reads from stdin in a goroutine without blocking
func (m *simpleTuiModel) readStdinAsync() {
	defer close(m.inputChan)

	scanner := bufio.NewScanner(os.Stdin)

	// Set larger buffer size (1MB) to handle long OTLP JSON lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	// Channel to receive scan results
	scanChan := make(chan bool, 1)

	for {
		// Start scanning in a separate goroutine
		go func() {
			scanChan <- scanner.Scan()
		}()

		// Wait for either scan result or context cancellation
		select {
		case <-m.ctx.Done():
			return
		case hasLine := <-scanChan:
			if !hasLine {
				// EOF or error - exit gracefully
				break
			}

			line := scanner.Text()
			if line != "" {
				select {
				case m.inputChan <- line:
				case <-m.ctx.Done():
					return
				}
			}
		}
	}
}

// checkInputChannel checks for data from the unified input channel
func (m *simpleTuiModel) checkInputChannel() tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-m.inputChan:
			if !ok {
				// Channel closed, input is done
				return finishedMsg{}
			}
			if line != "" {
				return logLineMsg(line)
			}
			// Empty line, continue checking
			return m.checkInputChannel()()
		case <-time.After(50 * time.Millisecond):
			// No data available right now, check again soon
			// This timeout ensures the UI remains responsive
			return m.checkInputChannel()()
		}
	}
}

// Update handles messages and updates the model
func (m *simpleTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check for quit keys and cancel context
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
		}
		// Always forward to dashboard first - let it decide whether to quit
		newDashboard, cmd := m.dashboard.Update(msg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		// Forward to dashboard
		newDashboard, cmd := m.dashboard.Update(msg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

	case tui.UpdateIntervalMsg:
		// Update the interval and restart the periodic timer
		m.updateInterval = time.Duration(msg)
		m.timerSequence++ // Increment sequence to invalidate old timers
		cmds = append(cmds, m.periodicUpdate())

	case tui.ManualResetMsg:
		// Manual reset triggered by 'r' key
		m.freqMemory.Reset()
		m.lastFreqReset = time.Now()

		// Send update with reset flag to clear drain3 patterns as well
		snapshot := m.freqMemory.GetSnapshot()
		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount,
			ForceCountUpdate: true,
			ResetDrain3:      true, // Reset drain3 patterns
		}
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

	case logLineMsg:
		m.processLogLine(string(msg))

		// Continue checking for more data if we have input sources
		if (m.hasStdinData || m.hasFileInput || m.hasOTLPInput) && !m.finished {
			cmds = append(cmds, m.checkInputChannel())
		}

	case snapshotMsg:
		// Send snapshot to dashboard
		updateMsg := tui.UpdateMsg{
			Snapshot:      msg,
			SeverityCount: m.severityCounts,
			LineCount:     m.logCount, // Keep for backward compatibility
		}
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

	case finishedMsg:
		m.finished = true
		// Send final snapshot
		snapshot := m.freqMemory.GetSnapshot()
		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount, // Keep for backward compatibility
			ForceCountUpdate: true,       // Ensure final count is recorded
		}
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

		if m.testMode {
			// In test mode, quit after showing results briefly
			cmds = append(cmds, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return tea.Quit()
			}))
		}

	case tickMsg:
		// Ignore ticks from old timers (prevents burst behavior)
		if msg.sequence != m.timerSequence {
			return m, nil // Ignore this tick
		}

		// Send periodic snapshot with current count (even if 0)
		snapshot := m.freqMemory.GetSnapshot()

		// No automatic reset - only manual reset via 'r' key now
		shouldReset := false

		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount,  // Keep for backward compatibility
			ForceCountUpdate: true,        // Always update count history, even with 0
			ResetDrain3:      shouldReset, // Reset drain3 when frequency memory resets
		}

		// Reset severity counts for next interval
		m.severityCounts = &tui.SeverityCounts{}
		m.logCount = 0
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

		// Always reset log count for counts chart tracking
		m.logCount = 0

		// Always schedule next update to keep dashboard refreshing
		cmds = append(cmds, m.periodicUpdate())

	default:
		// Forward unknown messages to dashboard
		newDashboard, cmd := m.dashboard.Update(msg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m *simpleTuiModel) View() string {
	if m.testMode && m.finished {
		// In test mode, show simple status
		snapshot := m.freqMemory.GetSnapshot()
		if snapshot == nil {
			return "No data processed yet.\n"
		}

		result := "Test Mode Results:\n\n"
		result += fmt.Sprintf("Total lines: %d\n", m.logCount)
		result += fmt.Sprintf("Unique words: %d\n", len(snapshot.Words))
		result += fmt.Sprintf("Unique phrases: %d\n", len(snapshot.Phrases))
		result += fmt.Sprintf("Attribute keys: %d\n", len(snapshot.Attributes))
		result += "\nTest completed successfully - no crashes!\n"
		result += "Press 'q' to quit or wait 2 seconds for auto-exit.\n"
		return result
	}

	return m.dashboard.View()
}

// periodicUpdate schedules periodic updates to the dashboard
func (m *simpleTuiModel) periodicUpdate() tea.Cmd {
	sequence := m.timerSequence
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return tickMsg{
			time:     t,
			sequence: sequence,
		}
	})
}
