package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Helper to build a tea.KeyMsg for a rune key
func runeKey(r string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)}
}

func TestHandleKeyPress_XKeyHidesChatPane(t *testing.T) {
	m := &DashboardModel{
		showModal:       true,
		currentLogEntry: &LogEntry{},
		chatPaneVisible: true,
	}
	m.handleKeyPress(runeKey("x"))

	if m.chatPaneVisible {
		t.Error("expected chatPaneVisible to be false after pressing x")
	}
}

func TestHandleKeyPress_XKeyShowsChatPane(t *testing.T) {
	m := &DashboardModel{
		showModal:       true,
		currentLogEntry: &LogEntry{},
		chatPaneVisible: false,
	}
	m.handleKeyPress(runeKey("x"))

	if !m.chatPaneVisible {
		t.Error("expected chatPaneVisible to be true after pressing x")
	}
}

func TestHandleKeyPress_XKeySwitchesFocusToInfoWhenHidingFromChat(t *testing.T) {
	m := &DashboardModel{
		showModal:          true,
		currentLogEntry:    &LogEntry{},
		chatPaneVisible:    true,
		modalActiveSection: "chat",
	}
	m.handleKeyPress(runeKey("x"))

	if m.chatPaneVisible {
		t.Error("expected chatPaneVisible to be false")
	}
	if m.modalActiveSection != "info" {
		t.Errorf("expected modalActiveSection to be 'info', got %q", m.modalActiveSection)
	}
}

func TestHandleKeyPress_XKeyPreservesFocusWhenAlreadyOnInfo(t *testing.T) {
	m := &DashboardModel{
		showModal:          true,
		currentLogEntry:    &LogEntry{},
		chatPaneVisible:    true,
		modalActiveSection: "info",
	}
	m.handleKeyPress(runeKey("x"))

	if m.chatPaneVisible {
		t.Error("expected chatPaneVisible to be false")
	}
	if m.modalActiveSection != "info" {
		t.Errorf("expected modalActiveSection to stay 'info', got %q", m.modalActiveSection)
	}
}

func TestHandleKeyPress_TabIgnoredWhenChatHidden(t *testing.T) {
	m := &DashboardModel{
		showModal:          true,
		currentLogEntry:    &LogEntry{},
		chatPaneVisible:    false,
		modalActiveSection: "info",
		width:              120,
		height:             40,
	}
	m.handleKeyPress(tea.KeyMsg{Type: tea.KeyTab})

	if m.modalActiveSection != "info" {
		t.Errorf("expected modalActiveSection to stay 'info' when chat is hidden, got %q", m.modalActiveSection)
	}
}

func TestHandleKeyPress_TabSwitchesToChatWhenVisible(t *testing.T) {
	m := &DashboardModel{
		showModal:          true,
		currentLogEntry:    &LogEntry{},
		chatPaneVisible:    true,
		modalActiveSection: "info",
		width:              120,
		height:             40,
	}
	m.handleKeyPress(tea.KeyMsg{Type: tea.KeyTab})

	if m.modalActiveSection != "chat" {
		t.Errorf("expected modalActiveSection to be 'chat', got %q", m.modalActiveSection)
	}
}

func TestHandleKeyPress_XKeyIgnoredWhenNoModal(t *testing.T) {
	m := &DashboardModel{
		showModal:       false,
		currentLogEntry: nil,
		chatPaneVisible: true,
	}
	m.handleKeyPress(runeKey("x"))

	// Should not toggle — x is only handled in the modal context
	if !m.chatPaneVisible {
		t.Error("expected chatPaneVisible to remain true when no modal is open")
	}
}

func TestHandleKeyPress_XKeyIgnoredInSingleModal(t *testing.T) {
	m := &DashboardModel{
		showModal:       true,
		currentLogEntry: nil, // single modal, not log detail
		chatPaneVisible: true,
	}
	m.handleKeyPress(runeKey("x"))

	if !m.chatPaneVisible {
		t.Error("expected chatPaneVisible to remain true in single modal (no log entry)")
	}
}
