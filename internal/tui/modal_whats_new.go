package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/control-theory/gonzo/internal/state"
)

// gitHashRe matches full 40-character git commit hashes.
var gitHashRe = regexp.MustCompile(`\b([0-9a-f]{40})\b`)

// renderWhatsNewModal renders the "What's New" modal with GitHub release notes.
func (m *DashboardModel) renderWhatsNewModal() string {
	modalWidth := m.width - 8
	modalHeight := m.height - 4

	contentWidth := modalWidth - 4
	contentHeight := modalHeight - 4

	m.infoViewport.Width = contentWidth
	m.infoViewport.Height = contentHeight

	// Only re-render content if width changed or cache is empty
	if m.whatsNewRenderedCache == "" || m.whatsNewCacheWidth != contentWidth {
		m.whatsNewRenderedCache = m.buildWhatsNewContent(contentWidth)
		m.whatsNewCacheWidth = contentWidth
		m.infoViewport.SetContent(m.whatsNewRenderedCache)
	}

	contentPane := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorGray).
		Render(m.infoViewport.View())

	header := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(ColorGreen).
		Bold(true).
		Render("🚀 What's New in Gonzo")

	statusBar := lipgloss.NewStyle().
		Foreground(ColorGray).
		Render("↑↓/Wheel: Scroll • ESC: Close")

	modal := lipgloss.JoinVertical(lipgloss.Left, header, contentPane, statusBar)

	finalModal := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorGreen).
		Render(modal)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, finalModal)
}

// buildWhatsNewContent renders all release notes via glamour. Called once and cached.
func (m *DashboardModel) buildWhatsNewContent(width int) string {
	if m.releasesFetcher == nil {
		return "Loading release notes..."
	}

	releases := m.releasesFetcher.GetReleasesNonBlocking()
	if len(releases) == 0 {
		return "Could not load release notes. Check https://github.com/control-theory/gonzo/releases"
	}

	renderWidth := max(40, width-4)
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(renderWidth),
	)

	var sb strings.Builder

	for i, rel := range releases {
		version := rel.TagName
		name := rel.Name
		if name == "" {
			name = version
		}

		dateStr := ""
		if t, parseErr := time.Parse(time.RFC3339, rel.PublishedAt); parseErr == nil {
			dateStr = t.Format("January 2, 2006")
		}

		if i == 0 && m.currentVersion != "" && ("v"+m.currentVersion == version || m.currentVersion == version || baseVersion("v"+m.currentVersion) == version) {
			badge := lipgloss.NewStyle().Bold(true).Foreground(ColorGreen).Render(fmt.Sprintf("★ %s (current)", name))
			if dateStr != "" {
				badge += lipgloss.NewStyle().Foreground(ColorGray).Render("  " + dateStr)
			}
			sb.WriteString(badge)
		} else {
			badge := lipgloss.NewStyle().Bold(true).Foreground(ColorBlue).Render(name)
			if dateStr != "" {
				badge += lipgloss.NewStyle().Foreground(ColorGray).Render("  " + dateStr)
			}
			sb.WriteString(badge)
		}
		sb.WriteString("\n")

		body := gitHashRe.ReplaceAllStringFunc(strings.TrimSpace(rel.Body), func(hash string) string {
			return hash[:7]
		})
		if body != "" && err == nil {
			rendered, renderErr := renderer.Render(body)
			if renderErr == nil {
				sb.WriteString(rendered)
			} else {
				sb.WriteString(body)
				sb.WriteString("\n")
			}
		} else if body != "" {
			sb.WriteString(body)
			sb.WriteString("\n")
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorGray).Render("  No release notes."))
			sb.WriteString("\n")
		}

		if i < len(releases)-1 {
			sep := lipgloss.NewStyle().Foreground(ColorGray).Render(strings.Repeat("─", min(width-4, 60)))
			sb.WriteString("\n" + sep + "\n\n")
		}
	}

	return sb.String()
}

// saveWhatsNewState saves the current version as "last seen" so the modal won't auto-show again.
func (m *DashboardModel) saveWhatsNewState() tea.Cmd {
	base := baseVersion(m.currentVersion)
	return func() tea.Msg {
		state.Save(state.AppState{LastSeenVersion: base})
		return nil
	}
}

// baseVersion strips git describe suffixes like "-13-g58cce74-dirty" from a version string.
func baseVersion(v string) string {
	s := v
	prefix := ""
	if strings.HasPrefix(s, "v") {
		prefix = "v"
		s = s[1:]
	}
	parts := strings.SplitN(s, "-", 2)
	return prefix + parts[0]
}

// checkWhatsNewAutoShow checks if we should auto-show the what's-new modal.
func (m *DashboardModel) checkWhatsNewAutoShow() {
	if m.whatsNewCheckDone || m.currentVersion == "" || m.currentVersion == "dev" {
		m.whatsNewCheckDone = true
		return
	}

	if m.releasesFetcher == nil {
		m.whatsNewCheckDone = true
		return
	}

	base := baseVersion(m.currentVersion)
	if baseVersion(m.lastSeenVersion) == base {
		m.whatsNewCheckDone = true
		return
	}

	rels := m.releasesFetcher.GetReleasesNonBlocking()
	if rels == nil {
		m.whatsNewRetries++
		if m.whatsNewRetries > 5 {
			m.whatsNewCheckDone = true
		}
		return
	}
	m.whatsNewCheckDone = true

	if len(rels) > 0 {
		m.showWhatsNewModal = true
		m.whatsNewRenderedCache = "" // force re-render
		m.infoViewport.GotoTop()
	}
}
