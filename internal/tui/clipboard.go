package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

// copyFeedbackDuration is how long a "Copied ..." status message stays
// visible in the modal status bar before it's cleared on the next tick.
const copyFeedbackDuration = 2 * time.Second

// copyToClipboard writes text to the OS clipboard.
func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

// setCopyFeedback records the result of a copy action so it can be
// rendered as a transient message in the modal status bar (see
// renderModalStatusBar). It self-expires after copyFeedbackDuration,
// cleared by the TickMsg handler in update.go.
func (m *DashboardModel) setCopyFeedback(err error, what string) {
	if err != nil {
		m.copyFeedback = fmt.Sprintf("Copy failed: %v", err)
	} else {
		m.copyFeedback = "Copied " + what + " to clipboard"
	}
	m.copyFeedbackExpiry = time.Now().Add(copyFeedbackDuration)
}

// formatLogEntryForClipboard renders a log entry as plain text (no ANSI/
// lipgloss styling), suitable for pasting outside the terminal — into an
// AI chat, a Slack message, or a bug report. Mirrors the same fields and
// ordering as formatLogDetails/formatAttributesTable (tables.go) so the
// copied text matches what's on screen, minus the styling.
func (m *DashboardModel) formatLogEntryForClipboard(entry LogEntry) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Received: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05.000"))
	if !entry.OrigTimestamp.IsZero() {
		fmt.Fprintf(&b, "Log Time: %s\n", entry.OrigTimestamp.Format("2006-01-02 15:04:05.000"))
	}
	fmt.Fprintf(&b, "Severity: %s\n", entry.Severity)
	fmt.Fprintf(&b, "Message: %s\n", entry.Message)

	if len(entry.Attributes) > 0 {
		b.WriteString("\nAttributes:\n")
		keys := make([]string, 0, len(entry.Attributes))
		for k := range entry.Attributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s: %s\n", k, entry.Attributes[k])
		}
	}

	if m.aiAnalysisResult != "" && m.aiAnalysisResult != "Analyzing..." {
		b.WriteString("\nAI Analysis:\n")
		b.WriteString(m.aiAnalysisResult)
		b.WriteString("\n")
	}

	return b.String()
}
