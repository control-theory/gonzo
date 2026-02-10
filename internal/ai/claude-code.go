package ai

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Available Claude models (shortnames supported by claude CLI)
var claudeModels = []string{
	"sonnet", // Default - Claude Sonnet (latest version)
	"haiku",  // Claude Haiku (fastest, most efficient)
	"opus",   // Claude Opus (most capable)
}

// buildCommand creates an exec.Cmd for running Claude with the given arguments.
// Handles both simple paths (e.g., "/usr/local/bin/claude") and complex commands
// (e.g., "podman exec -it container claude").
func (c *ClaudeCodeClient) buildCommand(args ...string) *exec.Cmd {
	// If ClaudePath contains spaces, it's likely a full command (e.g., "podman exec container claude")
	// Split it and use the first part as the command, rest as base args
	if strings.Contains(c.ClaudePath, " ") {
		parts := strings.Fields(c.ClaudePath)
		allArgs := append(parts[1:], args...)
		return exec.Command(parts[0], allArgs...)
	}

	// Simple path, just execute directly
	return exec.Command(c.ClaudePath, args...)
}

// ClaudeCodeClient handles Claude Code CLI requests
type ClaudeCodeClient struct {
	Model           string
	ClaudePath      string // Path to claude executable
	Validated       bool
	ValidationErr   string
	ServiceName     string
	AvailableModels []string
}

// NewClaudeCodeClient creates a new Claude Code client
func NewClaudeCodeClient(model string) *ClaudeCodeClient {
	// Check for custom Claude path from environment variable first
	// This allows running Claude in containers: GONZO_CLAUDE_PATH="podman exec -it container claude"
	claudePath := os.Getenv("GONZO_CLAUDE_PATH")

	if claudePath == "" {
		// Fall back to finding claude executable in PATH
		var err error
		claudePath, err = exec.LookPath("claude")
		if err != nil {
			return &ClaudeCodeClient{
				Validated:     false,
				ValidationErr: "claude CLI not found in PATH - install Claude Code from https://claude.ai/download or set GONZO_CLAUDE_PATH",
			}
		}
	}

	// Default to sonnet if no model specified
	if model == "" {
		model = "sonnet"
	}

	client := &ClaudeCodeClient{
		ClaudePath:      claudePath,
		Model:           model,
		ServiceName:     "Claude Code",
		AvailableModels: claudeModels,
	}

	// Validate configuration
	client.ValidateConfiguration()

	return client
}

// AnalyzeLog sends a log message to Claude Code for analysis
func (c *ClaudeCodeClient) AnalyzeLog(logMessage, severity, timestamp string, attributes map[string]string) (string, error) {
	if c == nil || !c.Validated {
		return "", fmt.Errorf("Claude Code client not configured: %s", c.ValidationErr)
	}

	prompt := c.buildAnalysisPrompt(logMessage, severity, timestamp, attributes)

	// Execute claude CLI in headless mode with model selection
	cmd := c.buildCommand("--model", c.Model, "-p", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("claude CLI execution failed: %s", errMsg)
	}

	response := strings.TrimSpace(stdout.String())
	if response == "" {
		return "", fmt.Errorf("claude CLI returned empty response")
	}

	return response, nil
}

// AnalyzeLogWithContext sends a log message to Claude Code with chat context
func (c *ClaudeCodeClient) AnalyzeLogWithContext(logMessage, severity, timestamp string, attributes map[string]string, previousAnalysis string, question string) (string, error) {
	if c == nil || !c.Validated {
		return "", fmt.Errorf("Claude Code client not configured: %s", c.ValidationErr)
	}

	// Build context-aware prompt
	prompt := fmt.Sprintf(`Previous analysis of log entry:
%s

User's follow-up question: %s

Log Details (for reference):
- Timestamp: %s
- Severity: %s
- Message: %s`,
		previousAnalysis, question, timestamp, severity, logMessage)

	if len(attributes) > 0 {
		prompt += "\n- Attributes:"
		for key, value := range attributes {
			prompt += fmt.Sprintf("\n  %s: %s", key, value)
		}
	}

	prompt += "\n\nPlease answer the user's specific question about this log entry. Be concise and helpful."

	// Execute claude CLI in headless mode with model selection
	cmd := c.buildCommand("--model", c.Model, "-p", prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return "", fmt.Errorf("claude CLI execution failed: %s", errMsg)
	}

	response := strings.TrimSpace(stdout.String())
	if response == "" {
		return "", fmt.Errorf("claude CLI returned empty response")
	}

	return response, nil
}

// buildAnalysisPrompt creates the analysis prompt for the log message
func (c *ClaudeCodeClient) buildAnalysisPrompt(logMessage, severity, timestamp string, attributes map[string]string) string {
	prompt := `You are an expert log analyst. Help me understand what this log message means and its implications.

Log Details:
- Timestamp: ` + timestamp + `
- Severity: ` + severity + `
- Message: ` + logMessage

	if len(attributes) > 0 {
		prompt += `
- Attributes:`
		for key, value := range attributes {
			prompt += fmt.Sprintf(`
  %s: %s`, key, value)
		}
	}

	prompt += `

Please provide:
1. What this log message indicates (what happened)
2. Whether this is normal/expected or indicates a problem
3. If it's a problem, what might be the root cause
4. Any recommended actions or things to investigate
5. Context about what this type of log typically means in applications

Keep your response concise but informative. Focus on practical insights that would help a developer or operator understand and respond to this log entry.`

	return prompt
}

// ValidateConfiguration checks if the Claude Code CLI is properly configured
func (c *ClaudeCodeClient) ValidateConfiguration() {
	if c == nil {
		return
	}

	// Check if we found the claude executable
	if c.ClaudePath == "" {
		c.Validated = false
		c.ValidationErr = "claude CLI not found in PATH"
		return
	}

	// Try to execute claude with --version to verify it works
	cmd := c.buildCommand("--version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		c.Validated = false
		c.ValidationErr = fmt.Sprintf("claude CLI found but not executable: %v", err)
		return
	}

	// Claude Code CLI is available and working
	c.Validated = true
	c.ValidationErr = ""

	// Model and AvailableModels are already set in NewClaudeCodeClient
	// Validate that the selected model is in the available list
	validModel := false
	for _, m := range c.AvailableModels {
		if c.Model == m {
			validModel = true
			break
		}
	}

	// If model is not valid, default to sonnet
	if !validModel {
		c.Model = "sonnet"
	}
}

// GetAvailableModels returns a list of Claude models
// Note: Claude Code doesn't expose an API to list models, so we return a static list
func (c *ClaudeCodeClient) GetAvailableModels() ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	if !c.Validated {
		return nil, fmt.Errorf("client not validated: %s", c.ValidationErr)
	}

	return c.AvailableModels, nil
}

// GetValidationStatus returns the validation status of the client
func (c *ClaudeCodeClient) GetValidationStatus() ValidationStatus {
	if c == nil {
		return ValidationStatus{
			Validated:    false,
			ErrorMessage: "Client not initialized",
			ServiceName:  "None",
			ModelName:    "",
		}
	}
	return ValidationStatus{
		Validated:    c.Validated,
		ErrorMessage: c.ValidationErr,
		ServiceName:  c.ServiceName,
		ModelName:    c.Model,
	}
}

// SetModel updates the model being used
func (c *ClaudeCodeClient) SetModel(model string) {
	if c != nil {
		c.Model = model
	}
}

// GetModel returns the current model being used
func (c *ClaudeCodeClient) GetModel() string {
	if c == nil {
		return ""
	}
	return c.Model
}

// CachedModels returns the cached list of available models without making API calls
func (c *ClaudeCodeClient) CachedModels() []string {
	if c == nil {
		return nil
	}
	return c.AvailableModels
}
