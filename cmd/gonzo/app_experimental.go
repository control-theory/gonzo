package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/control-theory/gonzo/internal/analyzer"
	"github.com/control-theory/gonzo/internal/logger"
	"github.com/control-theory/gonzo/internal/memory"
	"github.com/control-theory/gonzo/internal/otlplog"
	"github.com/control-theory/gonzo/internal/plugin"
	"github.com/control-theory/gonzo/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// runAppExperimental is a simplified version using the experimental plugin architecture
func runAppExperimental(cmd *cobra.Command, args []string) error {
	// Check if version flag was used
	if v, _ := cmd.Flags().GetBool("version"); v {
		versionCmd.Run(cmd, args)
		return nil
	}

	// Initialize skin/color scheme
	configDir := os.Getenv("HOME") + "/.config/gonzo"
	if err := tui.InitializeSkin(cfg.Skin, configDir); err != nil {
		logger.Warnf("Failed to load skin '%s': %v (using default)", cfg.Skin, err)
	}

	// Check for multi-source configuration first
	var adapter interface {
		Start() error
		Stop()
		GetLineChan() <-chan string
		GetMetrics() plugin.Metrics
	}
	var sourceName string

	// Check for config file first
	if cfgFile := viper.GetString("config"); cfgFile != "" {
		// Try to load sources from config file
		if fileConfig, err := loadConfigFile(cfgFile); err == nil && len(fileConfig.Sources) > 0 {
			multiAdapter, err := startSourcesFromConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to start sources from config: %w", err)
			}
			adapter = multiAdapter
			sourceName = fmt.Sprintf("config file (%d sources)", len(fileConfig.Sources))
			logger.Infof("Loaded %d sources from config file: %s", len(fileConfig.Sources), cfgFile)
		}
	} else if cfg.Source != "" {
		// Parse source configuration - could be single or multi
		multiConfig, err := parseSourceConfig(cfg.Source)
		if err != nil {
			return fmt.Errorf("failed to parse source config: %w", err)
		}

		if len(multiConfig.Sources) == 1 {
			// Single source from --source flag
			source := multiConfig.Sources[0]
			singleAdapter, err := startInputSource(source.Type, source.Config)
			if err != nil {
				return err
			}
			adapter = singleAdapter
			sourceName = source.Type
		} else {
			// Multiple sources from --source flag
			multiAdapter, err := startMultipleSources(multiConfig)
			if err != nil {
				return err
			}
			adapter = multiAdapter
			sourceName = fmt.Sprintf("multi (%d sources)", len(multiConfig.Sources))
		}
	} else {
		// Single source mode
		singleSourceName, sourceConfig, err := determineInputSource(&cfg)
		if err != nil {
			return err
		}

		singleAdapter, err := startInputSource(singleSourceName, sourceConfig)
		if err != nil {
			return err
		}
		adapter = singleAdapter
		sourceName = singleSourceName
	}
	defer adapter.Stop()

	// Initialize components
	formatDetector := otlplog.NewFormatDetector()
	logConverter := otlplog.NewLogConverter()
	textAnalyzer := analyzer.NewTextAnalyzer()
	otlpAnalyzer := analyzer.NewOTLPAnalyzer()
	freqMemory := memory.NewFrequencyMemory(cfg.MemorySize)

	// Initialize experimental TUI model
	tuiModel := &experimentalTuiModel{
		formatDetector: formatDetector,
		logConverter:   logConverter,
		textAnalyzer:   textAnalyzer,
		otlpAnalyzer:   otlpAnalyzer,
		freqMemory:     freqMemory,
		dashboard:      tui.NewDashboardModel(cfg.LogBuffer, cfg.UpdateInterval, cfg.AIModel),
		updateInterval: cfg.UpdateInterval,
		testMode:       cfg.TestMode,
		adapter:        adapter,
		sourceName:     sourceName,
	}

	var p *tea.Program
	if cfg.TestMode {
		p = tea.NewProgram(tuiModel, tea.WithInput(nil), tea.WithOutput(os.Stdout))
	} else {
		p = tea.NewProgram(tuiModel, tea.WithAltScreen(), tea.WithMouseCellMotion())
	}

	// Create cancellable context
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

// experimentalTuiModel is the simplified TUI model using the plugin system
type experimentalTuiModel struct {
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

	// Plugin adapter (can be single or multiplexer)
	adapter interface {
		GetLineChan() <-chan string
		GetMetrics() plugin.Metrics
	}
	sourceName string

	// Internal state
	finished       bool
	logCount       int
	severityCounts *tui.SeverityCounts
	lastFreqReset  time.Time
	timerSequence  int

	// JSON accumulation for multi-line OTLP support
	jsonBuffer   strings.Builder
	jsonDepth    int
	inJSONObject bool
}

// Init initializes the experimental TUI model
func (m *experimentalTuiModel) Init() tea.Cmd {
	// Initialize severity counts
	m.severityCounts = &tui.SeverityCounts{}
	m.lastFreqReset = time.Now()

	// Log which source we're using
	logger.Infof("Starting %s input source", m.sourceName)

	// Get metrics periodically
	go m.monitorMetrics()

	// Start the dashboard
	dashboardCmd := m.dashboard.Init()

	var cmds []tea.Cmd
	cmds = append(cmds, dashboardCmd)
	cmds = append(cmds, m.periodicUpdate())
	cmds = append(cmds, m.checkInputChannel())

	return tea.Batch(cmds...)
}

// monitorMetrics periodically logs adapter metrics
func (m *experimentalTuiModel) monitorMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			metrics := m.adapter.GetMetrics()
			if metrics.Connected {
				logger.Infof("[%s] Status: Connected | Logs: %d total, %.1f/sec",
					m.sourceName, metrics.TotalLogs, metrics.LogsPerSecond)
			} else if metrics.LastError != "" {
				logger.Errorf("[%s] Status: Disconnected | Error: %s",
					m.sourceName, metrics.LastError)
			}
		}
	}
}

// checkInputChannel checks for data from the plugin adapter
func (m *experimentalTuiModel) checkInputChannel() tea.Cmd {
	return func() tea.Msg {
		lineChan := m.adapter.GetLineChan()

		select {
		case line, ok := <-lineChan:
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
			return m.checkInputChannel()()
		}
	}
}

// Update handles messages and updates the model
func (m *experimentalTuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Check for quit keys and cancel context
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			if m.cancelFunc != nil {
				m.cancelFunc()
			}
		}
		// Forward to dashboard
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
		m.timerSequence++
		cmds = append(cmds, m.periodicUpdate())

	case tui.ManualResetMsg:
		// Manual reset triggered by 'r' key
		m.freqMemory.Reset()
		m.lastFreqReset = time.Now()

		// Send update with reset flag
		snapshot := m.freqMemory.GetSnapshot()

		// Calculate total for severity counts
		if m.severityCounts != nil {
			m.severityCounts.Total = m.severityCounts.Trace + m.severityCounts.Debug +
				m.severityCounts.Info + m.severityCounts.Warn + m.severityCounts.Error +
				m.severityCounts.Fatal + m.severityCounts.Critical + m.severityCounts.Unknown
		}

		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount,
			ForceCountUpdate: true,
			ResetDrain3:      true,
		}
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

	case logLineMsg:
		m.processLogLine(string(msg))
		// Continue checking for more data
		if !m.finished {
			cmds = append(cmds, m.checkInputChannel())
		}

	case snapshotMsg:
		// Send snapshot to dashboard

		// Calculate total for severity counts
		if m.severityCounts != nil {
			m.severityCounts.Total = m.severityCounts.Trace + m.severityCounts.Debug +
				m.severityCounts.Info + m.severityCounts.Warn + m.severityCounts.Error +
				m.severityCounts.Fatal + m.severityCounts.Critical + m.severityCounts.Unknown
		}

		updateMsg := tui.UpdateMsg{
			Snapshot:      msg,
			SeverityCount: m.severityCounts,
			LineCount:     m.logCount,
		}
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

	case finishedMsg:
		m.finished = true
		// Send final snapshot
		snapshot := m.freqMemory.GetSnapshot()

		// Calculate total for severity counts
		if m.severityCounts != nil {
			m.severityCounts.Total = m.severityCounts.Trace + m.severityCounts.Debug +
				m.severityCounts.Info + m.severityCounts.Warn + m.severityCounts.Error +
				m.severityCounts.Fatal + m.severityCounts.Critical + m.severityCounts.Unknown
		}

		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount,
			ForceCountUpdate: true,
		}
		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

		if m.testMode {
			// In test mode, quit after showing results briefly
			cmds = append(cmds, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
				return tea.Quit()
			}))
		}

	case tickMsg:
		// Ignore ticks from old timers
		if msg.sequence != m.timerSequence {
			return m, nil
		}

		// Send periodic snapshot
		snapshot := m.freqMemory.GetSnapshot()

		// Calculate total for severity counts
		if m.severityCounts != nil {
			m.severityCounts.Total = m.severityCounts.Trace + m.severityCounts.Debug +
				m.severityCounts.Info + m.severityCounts.Warn + m.severityCounts.Error +
				m.severityCounts.Fatal + m.severityCounts.Critical + m.severityCounts.Unknown
		}

		updateMsg := tui.UpdateMsg{
			Snapshot:         snapshot,
			SeverityCount:    m.severityCounts,
			LineCount:        m.logCount,
			ForceCountUpdate: true,
			ResetDrain3:      false,
		}

		// Reset counts for next interval
		m.severityCounts = &tui.SeverityCounts{}
		m.logCount = 0

		newDashboard, cmd := m.dashboard.Update(updateMsg)
		m.dashboard = newDashboard.(*tui.DashboardModel)
		cmds = append(cmds, cmd)

		// Schedule next update
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
func (m *experimentalTuiModel) View() string {
	if m.testMode && m.finished {
		// In test mode, show simple status
		snapshot := m.freqMemory.GetSnapshot()
		if snapshot == nil {
			return "No data processed yet.\n"
		}

		result := "Test Mode Results:\n\n"
		result += fmt.Sprintf("Total lines: %d\n", m.dashboard.GetTotalLogsProcessed())
		result += fmt.Sprintf("Dashboard attributes: %d\n", m.dashboard.GetAttributeCount())
		result += fmt.Sprintf("Unique words: %d\n", len(snapshot.Words))
		result += fmt.Sprintf("Unique phrases: %d\n", len(snapshot.Phrases))
		result += fmt.Sprintf("Attribute keys: %d\n", len(snapshot.Attributes))
		result += fmt.Sprintf("Input source: %s\n", m.sourceName)
		result += "\nTest completed successfully - no crashes!\n"
		result += "Press 'q' to quit or wait 2 seconds for auto-exit.\n"
		return result
	}

	return m.dashboard.View()
}

// periodicUpdate schedules periodic updates to the dashboard
func (m *experimentalTuiModel) periodicUpdate() tea.Cmd {
	sequence := m.timerSequence
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return tickMsg{
			time:     t,
			sequence: sequence,
		}
	})
}
