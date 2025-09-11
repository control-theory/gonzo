package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// SkinColors defines the color scheme for the TUI with semantic naming
type SkinColors struct {
	// UI Component Colors
	Primary       string `yaml:"primary"`        // Main accent color (section borders, highlights)
	Secondary     string `yaml:"secondary"`      // Secondary accent color
	Background    string `yaml:"background"`     // Main background
	Surface       string `yaml:"surface"`        // Secondary background (modals, panels)
	Border        string `yaml:"border"`         // Default border color
	BorderActive  string `yaml:"border_active"`  // Active section border
	Text          string `yaml:"text"`           // Primary text color
	TextSecondary string `yaml:"text_secondary"` // Secondary/muted text
	TextInverse   string `yaml:"text_inverse"`   // Text on colored backgrounds

	// Chart and Data Colors
	ChartTitle  string `yaml:"chart_title"`  // Chart titles
	ChartBar    string `yaml:"chart_bar"`    // Bar chart bars
	ChartAccent string `yaml:"chart_accent"` // Chart accent elements

	// Log Entry Colors
	LogTimestamp  string `yaml:"log_timestamp"`  // Log timestamps
	LogMessage    string `yaml:"log_message"`    // Log message text
	LogBackground string `yaml:"log_background"` // Log entry background
	LogSelected   string `yaml:"log_selected"`   // Selected log entry

	// Severity Level Colors
	SeverityTrace string `yaml:"severity_trace"` // TRACE level logs
	SeverityDebug string `yaml:"severity_debug"` // DEBUG level logs
	SeverityInfo  string `yaml:"severity_info"`  // INFO level logs
	SeverityWarn  string `yaml:"severity_warn"`  // WARN level logs
	SeverityError string `yaml:"severity_error"` // ERROR level logs
	SeverityFatal string `yaml:"severity_fatal"` // FATAL/CRITICAL level logs

	// Status Colors
	Success string `yaml:"success"` // Success states
	Warning string `yaml:"warning"` // Warning states
	Error   string `yaml:"error"`   // Error states
	Info    string `yaml:"info"`    // Info states

	// Special Elements
	Help      string `yaml:"help"`      // Help text
	Highlight string `yaml:"highlight"` // Search highlights, emphasis
	Disabled  string `yaml:"disabled"`  // Disabled elements
}

// Skin represents a complete color scheme
type Skin struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description,omitempty"`
	Author      string     `yaml:"author,omitempty"`
	Colors      SkinColors `yaml:"colors"`
}

// CurrentSkin holds the active skin
var CurrentSkin *Skin

// DefaultSkin returns the default color scheme
func DefaultSkin() *Skin {
	return &Skin{
		Name:        "default",
		Description: "Default Gonzo color scheme - Dark theme",
		Author:      "ControlTheory",
		Colors: SkinColors{
			// UI Component Colors
			Primary:       "#0f93fc", // Blue
			Secondary:     "#49E209", // Green
			Background:    "#081C39", // Navy
			Surface:       "#2D2D2D", // Dark gray
			Border:        "#BCBEC0", // Gray
			BorderActive:  "#0f93fc", // Blue
			Text:          "#FFFFFF", // White
			TextSecondary: "#BCBEC0", // Gray
			TextInverse:   "#000000", // Black

			// Chart and Data Colors
			ChartTitle:  "#0f93fc", // Blue
			ChartBar:    "#49E209", // Green
			ChartAccent: "#FF8C42", // Orange

			// Log Entry Colors
			LogTimestamp:  "#BCBEC0", // Gray
			LogMessage:    "#FFFFFF", // White
			LogBackground: "#081C39", // Navy
			LogSelected:   "#0f93fc", // Blue

			// Severity Level Colors
			SeverityTrace: "#888888", // Light gray
			SeverityDebug: "#BCBEC0", // Gray
			SeverityInfo:  "#0f93fc", // Blue
			SeverityWarn:  "#FFD93D", // Yellow
			SeverityError: "#FF8888", // Light red
			SeverityFatal: "#FF6B6B", // Red

			// Status Colors
			Success: "#49E209", // Green
			Warning: "#FFD93D", // Yellow
			Error:   "#FF6B6B", // Red
			Info:    "#0f93fc", // Blue

			// Special Elements
			Help:      "#BCBEC0", // Gray
			Highlight: "#FF69B4", // Pink
			Disabled:  "#666666", // Dark gray
		},
	}
}

// LoadSkin loads a skin from a YAML file
func LoadSkin(path string) (*Skin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skin file: %w", err)
	}

	var skin Skin
	if err := yaml.Unmarshal(data, &skin); err != nil {
		return nil, fmt.Errorf("failed to parse skin file: %w", err)
	}

	// Apply defaults for any missing colors
	defaultSkin := DefaultSkin()
	applyDefaults(&skin.Colors, &defaultSkin.Colors)

	return &skin, nil
}

// applyDefaults fills in any missing colors with defaults
func applyDefaults(colors *SkinColors, defaults *SkinColors) {
	if colors.Primary == "" {
		colors.Primary = defaults.Primary
	}
	if colors.Secondary == "" {
		colors.Secondary = defaults.Secondary
	}
	if colors.Background == "" {
		colors.Background = defaults.Background
	}
	if colors.Surface == "" {
		colors.Surface = defaults.Surface
	}
	if colors.Border == "" {
		colors.Border = defaults.Border
	}
	if colors.BorderActive == "" {
		colors.BorderActive = defaults.BorderActive
	}
	if colors.Text == "" {
		colors.Text = defaults.Text
	}
	if colors.TextSecondary == "" {
		colors.TextSecondary = defaults.TextSecondary
	}
	if colors.TextInverse == "" {
		colors.TextInverse = defaults.TextInverse
	}

	if colors.ChartTitle == "" {
		colors.ChartTitle = defaults.ChartTitle
	}
	if colors.ChartBar == "" {
		colors.ChartBar = defaults.ChartBar
	}
	if colors.ChartAccent == "" {
		colors.ChartAccent = defaults.ChartAccent
	}

	if colors.LogTimestamp == "" {
		colors.LogTimestamp = defaults.LogTimestamp
	}
	if colors.LogMessage == "" {
		colors.LogMessage = defaults.LogMessage
	}
	if colors.LogBackground == "" {
		colors.LogBackground = defaults.LogBackground
	}
	if colors.LogSelected == "" {
		colors.LogSelected = defaults.LogSelected
	}

	if colors.SeverityTrace == "" {
		colors.SeverityTrace = defaults.SeverityTrace
	}
	if colors.SeverityDebug == "" {
		colors.SeverityDebug = defaults.SeverityDebug
	}
	if colors.SeverityInfo == "" {
		colors.SeverityInfo = defaults.SeverityInfo
	}
	if colors.SeverityWarn == "" {
		colors.SeverityWarn = defaults.SeverityWarn
	}
	if colors.SeverityError == "" {
		colors.SeverityError = defaults.SeverityError
	}
	if colors.SeverityFatal == "" {
		colors.SeverityFatal = defaults.SeverityFatal
	}

	if colors.Success == "" {
		colors.Success = defaults.Success
	}
	if colors.Warning == "" {
		colors.Warning = defaults.Warning
	}
	if colors.Error == "" {
		colors.Error = defaults.Error
	}
	if colors.Info == "" {
		colors.Info = defaults.Info
	}

	if colors.Help == "" {
		colors.Help = defaults.Help
	}
	if colors.Highlight == "" {
		colors.Highlight = defaults.Highlight
	}
	if colors.Disabled == "" {
		colors.Disabled = defaults.Disabled
	}
}

// LoadSkinByName loads a skin by name from the skins directory
func LoadSkinByName(name string, configDir string) (*Skin, error) {
	if name == "" || name == "default" {
		return DefaultSkin(), nil
	}

	// Check for .yaml extension
	if filepath.Ext(name) == "" {
		name = name + ".yaml"
	}

	// Try to load from skins directory
	skinsDir := filepath.Join(configDir, "skins")
	skinPath := filepath.Join(skinsDir, name)

	// Check if file exists
	if _, err := os.Stat(skinPath); os.IsNotExist(err) {
		// Try without .yaml if it was added
		if filepath.Ext(name) == ".yaml" {
			altPath := filepath.Join(skinsDir, name[:len(name)-5])
			if _, err := os.Stat(altPath); err == nil {
				skinPath = altPath
			}
		}
	}

	return LoadSkin(skinPath)
}

// InitializeSkin sets up the skin system with the specified skin
func InitializeSkin(skinName string, configDir string) error {
	skin, err := LoadSkinByName(skinName, configDir)
	if err != nil {
		// Fall back to default skin on error
		skin = DefaultSkin()
	}

	CurrentSkin = skin
	updateColorVariables()
	updateStyles()

	return err
}

// updateColorVariables updates the global color variables based on the current skin
func updateColorVariables() {
	if CurrentSkin == nil {
		CurrentSkin = DefaultSkin()
	}

	// Map semantic colors to legacy color variables for backward compatibility
	ColorBlue = lipgloss.Color(CurrentSkin.Colors.Primary)
	ColorGreen = lipgloss.Color(CurrentSkin.Colors.Success)
	ColorNavy = lipgloss.Color(CurrentSkin.Colors.Background)
	ColorGray = lipgloss.Color(CurrentSkin.Colors.TextSecondary)
	ColorDarkGray = lipgloss.Color(CurrentSkin.Colors.Surface)
	ColorBlack = lipgloss.Color(CurrentSkin.Colors.TextInverse)
	ColorWhite = lipgloss.Color(CurrentSkin.Colors.Text)
	ColorRed = lipgloss.Color(CurrentSkin.Colors.Error)
	ColorYellow = lipgloss.Color(CurrentSkin.Colors.Warning)
	ColorOrange = lipgloss.Color(CurrentSkin.Colors.ChartAccent)
	ColorPink = lipgloss.Color(CurrentSkin.Colors.Highlight)
}

// updateStyles recreates all styles with the new colors
func updateStyles() {
	sectionStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(CurrentSkin.Colors.Border)).
		Padding(0, 1).
		Margin(0)

	activeSectionStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(CurrentSkin.Colors.BorderActive)).
		Padding(0, 1).
		Margin(0)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(CurrentSkin.Colors.Help)).
		Italic(true).
		Padding(1)

	chartTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(CurrentSkin.Colors.ChartTitle)).
		Bold(true).
		Align(lipgloss.Center)
}

// GetSeverityColor returns the color for a given severity level
// This uses semantic colors if defined, otherwise falls back to defaults
func GetSeverityColor(severity string) lipgloss.Color {
	if CurrentSkin == nil {
		CurrentSkin = DefaultSkin()
	}

	switch severity {
	case "TRACE", "trace":
		return lipgloss.Color(CurrentSkin.Colors.SeverityTrace)
	case "DEBUG", "debug":
		return lipgloss.Color(CurrentSkin.Colors.SeverityDebug)
	case "INFO", "info":
		return lipgloss.Color(CurrentSkin.Colors.SeverityInfo)
	case "WARN", "warn", "WARNING", "warning":
		return lipgloss.Color(CurrentSkin.Colors.SeverityWarn)
	case "ERROR", "error":
		return lipgloss.Color(CurrentSkin.Colors.SeverityError)
	case "FATAL", "fatal", "CRITICAL", "critical":
		return lipgloss.Color(CurrentSkin.Colors.SeverityFatal)
	default:
		return lipgloss.Color(CurrentSkin.Colors.TextSecondary)
	}
}
