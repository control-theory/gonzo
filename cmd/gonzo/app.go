package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"io/fs"

	"github.com/control-theory/gonzo/internal/analyzer"
	"github.com/control-theory/gonzo/internal/engine"
	"github.com/control-theory/gonzo/internal/filereader"
	"github.com/control-theory/gonzo/internal/formats"
	"github.com/control-theory/gonzo/internal/k8s"
	"github.com/control-theory/gonzo/internal/memory"
	"github.com/control-theory/gonzo/internal/otlplog"
	"github.com/control-theory/gonzo/internal/otlpreceiver"
	"github.com/control-theory/gonzo/internal/releases"
	"github.com/control-theory/gonzo/internal/state"
	"github.com/control-theory/gonzo/internal/tui"
	versioncheck "github.com/control-theory/gonzo/internal/version"
	"github.com/control-theory/gonzo/internal/vmlogs"
	gonzoweb "github.com/control-theory/gonzo/internal/web"
	webembed "github.com/control-theory/gonzo/web"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
)

// runApp initializes and runs the application
func runApp(cmd *cobra.Command, args []string) error {
	// Check if version flag was used
	if v, _ := cmd.Flags().GetBool("version"); v {
		versionCmd.Run(cmd, args)
		return nil
	}

	// Redirect log output to discard to avoid messing up the TUI
	// All log.Printf calls will be silently discarded
	log.SetOutput(io.Discard)

	// Suppress klog output from Kubernetes client-go
	// This prevents errors like "request.go:752" from appearing in the TUI
	klog.SetOutput(io.Discard)
	klog.LogToStderr(false)

	// Start version checking in background (if not disabled)
	var versionChecker *versioncheck.Checker
	if !cfg.DisableVersionCheck {
		currentVersion, currentCommit := GetVersionInfo()
		versionChecker = versioncheck.NewChecker(currentVersion, currentCommit)
		versionChecker.CheckInBackground()
	}

	// Initialize skin/color scheme
	configDir := os.Getenv("HOME") + "/.config/gonzo"
	if err := tui.InitializeSkin(cfg.Skin, configDir); err != nil {
		// Log warning but continue with default skin
		log.Printf("Warning: Failed to load skin '%s': %v (using default)", cfg.Skin, err)
	}

	// Initialize format detector and converter with custom format if specified
	var formatDetector *otlplog.FormatDetector
	var logConverter *otlplog.LogConverter
	var customParser *formats.Parser

	if cfg.Format != "" {
		// Check if it's a built-in format
		switch strings.ToLower(cfg.Format) {
		case "otlp", "json", "text":
			// Built-in format
			formatDetector = otlplog.NewFormatDetectorWithFormat(cfg.Format)
			logConverter = otlplog.NewLogConverter()
		default:
			// Try to load custom format
			format, err := formats.LoadFormatByName(cfg.Format, configDir)
			if err != nil {
				log.Printf("Warning: Failed to load custom format '%s': %v (using auto-detect)", cfg.Format, err)
				formatDetector = otlplog.NewFormatDetector()
				logConverter = otlplog.NewLogConverter()
			} else {
				// Create parser for the custom format
				customParser, err = formats.NewParser(format)
				if err != nil {
					log.Printf("Warning: Failed to create parser for format '%s': %v (using auto-detect)", cfg.Format, err)
					formatDetector = otlplog.NewFormatDetector()
					logConverter = otlplog.NewLogConverter()
				} else {
					formatDetector = otlplog.NewFormatDetectorWithFormat(cfg.Format)
					logConverter = otlplog.NewLogConverterWithFormat(cfg.Format, customParser)
					log.Printf("Using custom format: %s", cfg.Format)
				}
			}
		}
	} else {
		// Auto-detect format
		formatDetector = otlplog.NewFormatDetector()
		logConverter = otlplog.NewLogConverter()
	}

	textAnalyzer := analyzer.NewTextAnalyzerWithStopWords(cfg.StopWords)
	otlpAnalyzer := analyzer.NewOTLPAnalyzer()
	freqMemory := memory.NewFrequencyMemory(cfg.MemorySize)

	// Initialize TUI model with components
	dashboard := tui.NewDashboardModel(cfg.LogBuffer, cfg.UpdateInterval, cfg.AIProvider, cfg.AIModel, textAnalyzer.GetStopWords(), cfg.ReverseScrollWheel, cfg.UseLogTime)
	if versionChecker != nil {
		dashboard.SetVersionChecker(versionChecker)
	}
	if !cfg.WebDisabled {
		dashboard.SetWebPort(cfg.WebPort)
	}

	// What's New: fetch GitHub releases in background and wire up auto-show
	currentVersion, _ := GetVersionInfo()
	relFetcher := releases.NewFetcher()
	if currentVersion != "dev" && currentVersion != "" && !cfg.DisableVersionCheck {
		relFetcher.FetchInBackground()
	}
	appState := state.Load()
	dashboard.SetReleasesFetcher(relFetcher)
	dashboard.SetCurrentVersion(currentVersion)
	dashboard.SetLastSeenVersion(appState.LastSeenVersion)

	// Create shared analysis engine (used by web dashboard)
	eng := engine.NewEngine(cfg.LogBuffer, textAnalyzer.GetStopWords(), nil, cfg.UseLogTime)

	tuiModel := &simpleTuiModel{
		formatDetector: formatDetector,
		logConverter:   logConverter,
		customParser:   customParser,
		textAnalyzer:   textAnalyzer,
		otlpAnalyzer:   otlpAnalyzer,
		freqMemory:     freqMemory,
		dashboard:      dashboard,
		engine:         eng,
		updateInterval: cfg.UpdateInterval,
		testMode:       cfg.TestMode,
		versionChecker: versionChecker,
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

	// Start web dashboard in background (if not disabled)
	if !cfg.WebDisabled {
		// Get embedded web dashboard assets
		var staticFS fs.FS
		if distFS, err := fs.Sub(webembed.DistFS, "dist"); err == nil {
			staticFS = distFS
		}
		webSrv := gonzoweb.NewServer(eng, staticFS, currentVersion, relFetcher)
		if err := webSrv.Start(ctx, cfg.WebPort); err != nil {
			log.Printf("Warning: web dashboard failed to start: %v", err)
		}
	}

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
	customParser   *formats.Parser
	textAnalyzer   *analyzer.TextAnalyzer
	otlpAnalyzer   *analyzer.OTLPAnalyzer
	freqMemory     *memory.FrequencyMemory
	dashboard      *tui.DashboardModel
	engine         *engine.Engine // Shared analysis state for web dashboard
	updateInterval time.Duration
	testMode       bool
	ctx            context.Context
	cancelFunc     context.CancelFunc
	versionChecker *versioncheck.Checker

	// stdin is the source for stdin-mode reading. It defaults to os.Stdin but is
	// an interface so the quit-teardown path can be unit-tested. On quit we close
	// it to unblock a goroutine parked in a blocking read(2) on a live pipe
	// (e.g. `kubectl logs -f ... | gonzo`); see handleQuitKey and issue #58.
	stdin         io.ReadCloser
	stopStdinOnce sync.Once

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

	// Victoria Logs receiver support
	vmlogsReceiver *vmlogs.Receiver // Victoria Logs receiver for streaming logs
	hasVmlogsInput bool             // Whether we're receiving Victoria Logs data

	// Kubernetes receiver support
	k8sReceiver *k8s.KubernetesLogSource // Kubernetes log source for streaming pod logs
	hasK8sInput bool                     // Whether we're receiving Kubernetes logs

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

	// Check if Kubernetes receiver is enabled
	if cfg.K8sEnabled {
		// Kubernetes input mode
		m.hasK8sInput = true
		m.inputChan = make(chan string, 100)

		// Create kubernetes config
		k8sConfig := &k8s.Config{
			Kubeconfig: cfg.K8sKubeconfig,
			Context:    cfg.K8sContext,
			Namespaces: cfg.K8sNamespaces,
			Selector:   cfg.K8sSelector,
			Since:      cfg.K8sSince,
			TailLines:  cfg.K8sTailLines,
		}

		// Create and start Kubernetes log source
		k8sSource, err := k8s.NewKubernetesLogSource(k8sConfig)
		if err != nil {
			log.Printf("Error creating Kubernetes log source: %v", err)
			// Fall back to other input methods if Kubernetes fails
			m.hasK8sInput = false
		} else {
			m.k8sReceiver = k8sSource
			if err := m.k8sReceiver.Start(); err != nil {
				log.Printf("Error starting Kubernetes log source: %v", err)
				// Fall back to other input methods if Kubernetes fails
				m.hasK8sInput = false
			} else {
				// Wire K8s source to the dashboard for namespace/pod listing
				m.dashboard.SetK8sSource(k8sSource)
				// Start reading from Kubernetes receiver in the background
				go m.readK8sAsync()
			}
		}
	}

	// Check if Victoria Logs receiver is enabled (only if Kubernetes is not enabled)
	if !m.hasK8sInput && cfg.VmlogsURL != "" {
		// Victoria Logs input mode
		m.hasVmlogsInput = true
		m.inputChan = make(chan string, 100)

		// Create and start Victoria Logs receiver
		params := make(map[string]string)
		// Add any default parameters here if needed
		m.vmlogsReceiver = vmlogs.NewReceiver(cfg.VmlogsURL, cfg.VmlogsUser, cfg.VmlogsPassword, cfg.VmlogsQuery, params)
		if err := m.vmlogsReceiver.Start(); err != nil {
			log.Printf("Error starting Victoria Logs receiver: %v", err)
			// Fall back to other input methods if Victoria Logs fails
			m.hasVmlogsInput = false
		} else {
			// Start reading from Victoria Logs receiver in the background
			go m.readVmlogsAsync()
		}
	}

	// Check if OTLP receiver is enabled (only if Kubernetes and Victoria Logs are not enabled)
	if !m.hasK8sInput && !m.hasVmlogsInput && cfg.OTLPEnabled {
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

	// Check if we have file inputs specified (only if Kubernetes, Victoria Logs and OTLP are not enabled)
	if !m.hasK8sInput && !m.hasVmlogsInput && !m.hasOTLPInput && len(cfg.Files) > 0 {
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

	// If no Kubernetes, no Victoria Logs, no OTLP, no file input or file input failed, check stdin
	if !m.hasK8sInput && !m.hasVmlogsInput && !m.hasOTLPInput && !m.hasFileInput {
		// Check if stdin has data available (not a terminal)
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// stdin is a pipe or file, we have data
			m.hasStdinData = true
			m.inputChan = make(chan string, 100)
			m.stdin = os.Stdin

			// Start goroutine to read stdin without blocking
			go m.readStdinAsync()
		}
	}

	// Auto-detect log source if nothing was explicitly configured
	if !m.hasK8sInput && !m.hasVmlogsInput && !m.hasOTLPInput && !m.hasFileInput && !m.hasStdinData && !hasExplicitSourceFlag() {
		m.autoDetectLogSource()
	}

	// Start the dashboard
	dashboardCmd := m.dashboard.Init()

	var cmds []tea.Cmd
	cmds = append(cmds, dashboardCmd)
	cmds = append(cmds, m.periodicUpdate())

	// Start checking for input data if we have any input source
	if m.hasStdinData || m.hasFileInput || m.hasOTLPInput || m.hasVmlogsInput || m.hasK8sInput {
		cmds = append(cmds, m.checkInputChannel())
	}

	return tea.Batch(cmds...)
}

// readK8sAsync reads from the Kubernetes log source
func (m *simpleTuiModel) readK8sAsync() {
	defer close(m.inputChan)

	if m.k8sReceiver == nil {
		return
	}

	// Get the channel from Kubernetes receiver
	k8sLineChan := m.k8sReceiver.GetLineChan()

	// Forward lines from Kubernetes receiver to input channel
	for {
		select {
		case <-m.ctx.Done():
			m.k8sReceiver.Stop()
			return
		case line, ok := <-k8sLineChan:
			if !ok {
				// Kubernetes receiver finished
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

// readVmlogsAsync reads from the Victoria Logs receiver
func (m *simpleTuiModel) readVmlogsAsync() {
	defer close(m.inputChan)

	if m.vmlogsReceiver == nil {
		return
	}

	// Get the channel from Victoria Logs receiver
	vmlogsLineChan := m.vmlogsReceiver.GetLineChan()

	// Forward lines from Victoria Logs receiver to input channel
	for {
		select {
		case <-m.ctx.Done():
			m.vmlogsReceiver.Stop()
			return
		case line, ok := <-vmlogsLineChan:
			if !ok {
				// Victoria Logs receiver finished
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

// handleQuitKey cancels the app context and stops the input reader when the
// user presses a quit key ("q" or Ctrl-C). This is the lifecycle teardown seam
// for issue #58: on quit, bubbletea restores the terminal via tea.Quit, but the
// stdin-reading goroutine can stay parked in a blocking read(2) on a live pipe
// (e.g. `kubectl logs -f ... | gonzo`) whose writer never sends EOF. Without
// closing stdin, that read holds the terminal and control is not returned to the
// parent (K9s) until a follow-up Ctrl-C. It returns true if msg was a quit key.
func (m *simpleTuiModel) handleQuitKey(msg tea.KeyMsg) bool {
	if s := msg.String(); s != "q" && s != "ctrl+c" {
		return false
	}
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	m.stopInput()
	return true
}

// stopInput unblocks the stdin reader by closing the stdin source exactly once.
// Closing the read end makes a blocked Scan/read(2) return immediately so the
// reader goroutine can exit and the terminal is released to the parent process.
func (m *simpleTuiModel) stopInput() {
	m.stopStdinOnce.Do(func() {
		if m.stdin != nil {
			_ = m.stdin.Close()
		}
	})
}

// readStdinAsync reads from stdin in a goroutine without blocking
func (m *simpleTuiModel) readStdinAsync() {
	defer close(m.inputChan)

	// Default to os.Stdin; tests inject a different reader via m.stdin.
	if m.stdin == nil {
		m.stdin = os.Stdin
	}

	scanner := bufio.NewScanner(m.stdin)

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
			// On quit, stopInput closes stdin, which unblocks the in-flight
			// scanner.Scan above so its goroutine can exit instead of leaking.
			m.stopInput()
			return
		case hasLine := <-scanChan:
			if !hasLine {
				// EOF or error - exit gracefully (return, not break, so we do
				// not re-spawn Scan and busy-spin on a closed input).
				return
			}

			line := scanner.Text()
			if line != "" {
				select {
				case m.inputChan <- line:
				case <-m.ctx.Done():
					m.stopInput()
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
		// On a quit key, cancel the app context and stop the stdin reader so
		// control returns to the parent process (e.g. K9s) without requiring a
		// follow-up Ctrl-C. See handleQuitKey and issue #58.
		m.handleQuitKey(msg)
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
		if (m.hasStdinData || m.hasFileInput || m.hasOTLPInput || m.hasVmlogsInput || m.hasK8sInput) && !m.finished {
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

		// Feed snapshot to the shared engine
		if m.engine != nil {
			m.engine.UpdateFrequencySnapshot(snapshot)
		}

		// No automatic reset - only manual reset via 'r' key now
		shouldReset := false

		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount,  // Keep for backward compatibility
			ForceCountUpdate: true,        // Always update count history, even with 0
			ResetDrain3:      shouldReset, // Reset drain3 when frequency memory resets
		}

		// Feed interval counts to the shared engine
		if m.engine != nil && m.severityCounts != nil {
			m.engine.IngestSeverityCounts(*m.severityCounts)
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

// hasExplicitSourceFlag returns true if the user explicitly provided any log source flag.
func hasExplicitSourceFlag() bool {
	return cfg.K8sEnabled ||
		len(cfg.Files) > 0 ||
		cfg.OTLPEnabled ||
		cfg.VmlogsURL != ""
}

// autoDetectLogSource tries to automatically find a log source:
// 1. If a kubeconfig exists, enable Kubernetes log streaming (all namespaces)
// 2. Otherwise, fall back to OS system logs
func (m *simpleTuiModel) autoDetectLogSource() {
	// Try kubeconfig first
	if kubeconfigs := k8s.DetectKubeconfig(); len(kubeconfigs) > 0 {
		// Kubeconfig found — auto-enable Kubernetes with all namespaces
		log.Printf("Auto-detected kubeconfig(s): %s; enabling Kubernetes log streaming", strings.Join(kubeconfigs, ", "))
		m.inputChan = make(chan string, 100)

		// Kubeconfig left empty so client-go resolves KUBECONFIG itself
		k8sConfig := &k8s.Config{
			Context:    cfg.K8sContext,
			Namespaces: cfg.K8sNamespaces,
			Selector:   cfg.K8sSelector,
			Since:      cfg.K8sSince,
			TailLines:  cfg.K8sTailLines,
		}

		k8sSource, err := k8s.NewKubernetesLogSource(k8sConfig)
		if err == nil {
			if err := k8sSource.Start(); err == nil {
				m.hasK8sInput = true
				m.k8sReceiver = k8sSource
				m.dashboard.SetK8sSource(k8sSource)
				go m.readK8sAsync()
				return
			}
			log.Printf("Auto-detect: Kubernetes start failed: %v", err)
		} else {
			log.Printf("Auto-detect: Kubernetes init failed: %v", err)
		}
	}

	// No kubeconfig — fall back to OS system logs
	logPath := systemLogPath()
	if logPath == "" {
		log.Printf("Auto-detect: no kubeconfig and no known system log path for %s", runtime.GOOS)
		return
	}

	if _, err := os.Stat(logPath); err != nil {
		log.Printf("Auto-detect: system log %s not accessible: %v", logPath, err)
		return
	}

	log.Printf("Auto-detected system log at %s", logPath)
	m.inputChan = make(chan string, 100)

	var err error
	m.fileReader, err = filereader.New([]string{logPath}, true) // follow mode
	if err != nil {
		log.Printf("Auto-detect: failed to read %s: %v", logPath, err)
		return
	}
	m.hasFileInput = true
	go m.readFilesAsync()
}

// systemLogPath returns the default system log file path for the current OS.
func systemLogPath() string {
	switch runtime.GOOS {
	case "darwin":
		return "/var/log/system.log"
	case "linux":
		// Try syslog first, then messages
		for _, p := range []string{"/var/log/syslog", "/var/log/messages"} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return ""
	default:
		return ""
	}
}
