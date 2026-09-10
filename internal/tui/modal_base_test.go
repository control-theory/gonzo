package tui

import (
	"strings"
	"testing"
)

func TestModalStatusBar_ChatVisible(t *testing.T) {
	m := &DashboardModel{
		currentLogEntry: &LogEntry{},
		chatPaneVisible: true,
	}
	got := m.renderModalStatusBar()

	if !strings.Contains(got, "x: Hide chat") {
		t.Errorf("expected status bar to contain 'x: Hide chat', got:\n%s", got)
	}
	if !strings.Contains(got, "Tab/Click: Switch panes") {
		t.Errorf("expected status bar to contain 'Tab/Click: Switch panes', got:\n%s", got)
	}
}

func TestModalStatusBar_ChatHidden(t *testing.T) {
	m := &DashboardModel{
		currentLogEntry: &LogEntry{},
		chatPaneVisible: false,
	}
	got := m.renderModalStatusBar()

	if !strings.Contains(got, "x: Show chat pane") {
		t.Errorf("expected status bar to contain 'x: Show chat pane', got:\n%s", got)
	}
	if strings.Contains(got, "Tab/Click") {
		t.Errorf("expected status bar NOT to contain 'Tab/Click' when chat is hidden, got:\n%s", got)
	}
}

func TestModalStatusBar_ChatActiveShowsSendHint(t *testing.T) {
	m := &DashboardModel{
		currentLogEntry:    &LogEntry{},
		chatPaneVisible:    true,
		modalActiveSection: "chat",
		chatActive:         true,
	}
	got := m.renderModalStatusBar()

	if !strings.Contains(got, "Enter: Send message") {
		t.Errorf("expected status bar to contain 'Enter: Send message' when chat is active, got:\n%s", got)
	}
	if !strings.Contains(got, "ESC: Stop typing") {
		t.Errorf("expected status bar to contain 'ESC: Stop typing' when chat is active, got:\n%s", got)
	}
}

func TestModalStatusBar_SingleModal(t *testing.T) {
	// No currentLogEntry = single modal (like Top Values)
	m := &DashboardModel{
		currentLogEntry: nil,
		chatPaneVisible: true,
	}
	got := m.renderModalStatusBar()

	if strings.Contains(got, "x:") {
		t.Errorf("expected no 'x:' hint in single modal, got:\n%s", got)
	}
	if !strings.Contains(got, "ESC: Close") {
		t.Errorf("expected 'ESC: Close' in status bar, got:\n%s", got)
	}
}
