package tui

import (
	"strings"
	"testing"
	"time"
)

func TestFormatLogEntryForClipboard(t *testing.T) {
	origTs := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	entry := LogEntry{
		Timestamp:     time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
		OrigTimestamp: origTs,
		Severity:      "ERROR",
		Message:       "connection refused",
		Attributes: map[string]string{
			"service.name": "checkout",
			"host.name":    "node-1",
		},
	}

	m := &DashboardModel{}
	got := m.formatLogEntryForClipboard(entry)

	for _, want := range []string{
		"Severity: ERROR",
		"Message: connection refused",
		"Log Time: 2026-01-02 03:04:05.000",
		"Attributes:",
		"  host.name: node-1",
		"  service.name: checkout",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("formatLogEntryForClipboard() missing %q, got:\n%s", want, got)
		}
	}

	// Attributes must be sorted alphabetically, matching formatAttributesTable.
	hostIdx := strings.Index(got, "host.name")
	serviceIdx := strings.Index(got, "service.name")
	if hostIdx == -1 || serviceIdx == -1 || hostIdx > serviceIdx {
		t.Errorf("expected attributes sorted alphabetically (host.name before service.name), got:\n%s", got)
	}
}

func TestFormatLogEntryForClipboard_NoAttributesNoAIAnalysis(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
		Severity:  "INFO",
		Message:   "server started",
	}

	m := &DashboardModel{}
	got := m.formatLogEntryForClipboard(entry)

	if strings.Contains(got, "Attributes:") {
		t.Errorf("expected no Attributes section for entry with no attributes, got:\n%s", got)
	}
	if strings.Contains(got, "AI Analysis") {
		t.Errorf("expected no AI Analysis section when aiAnalysisResult is empty, got:\n%s", got)
	}
	if strings.Contains(got, "Log Time") {
		t.Errorf("expected no Log Time line when OrigTimestamp is zero, got:\n%s", got)
	}
}

func TestFormatLogEntryForClipboard_IncludesAIAnalysisWhenPresent(t *testing.T) {
	entry := LogEntry{Timestamp: time.Now(), Severity: "WARN", Message: "retrying request"}

	m := &DashboardModel{aiAnalysisResult: "This looks like a transient network blip."}
	got := m.formatLogEntryForClipboard(entry)

	if !strings.Contains(got, "AI Analysis:\nThis looks like a transient network blip.") {
		t.Errorf("expected AI analysis to be included, got:\n%s", got)
	}
}

func TestFormatLogEntryForClipboard_SkipsPlaceholderAIAnalysis(t *testing.T) {
	entry := LogEntry{Timestamp: time.Now(), Severity: "WARN", Message: "retrying request"}

	// "Analyzing..." is the transient placeholder set while the AI request is
	// in flight (see navigation.go's "i" case) — it must never be copied.
	m := &DashboardModel{aiAnalysisResult: "Analyzing..."}
	got := m.formatLogEntryForClipboard(entry)

	if strings.Contains(got, "AI Analysis") {
		t.Errorf("expected placeholder 'Analyzing...' result to be excluded, got:\n%s", got)
	}
}

func TestSetCopyFeedback(t *testing.T) {
	m := &DashboardModel{}

	m.setCopyFeedback(nil, "message")
	if m.copyFeedback != "Copied message to clipboard" {
		t.Errorf("setCopyFeedback(nil, %q) = %q, want %q", "message", m.copyFeedback, "Copied message to clipboard")
	}
	if !m.copyFeedbackExpiry.After(time.Now()) {
		t.Errorf("expected copyFeedbackExpiry to be set in the future")
	}
}
