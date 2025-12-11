package ai

import "fmt"

// ProviderType represents the AI provider type
type ProviderType string

const (
	// ProviderOpenAI uses OpenAI-compatible APIs (OpenAI, Ollama, LM Studio, etc.)
	ProviderOpenAI ProviderType = "openai"

	// ProviderClaudeCode uses the Claude Code CLI
	ProviderClaudeCode ProviderType = "claude-code"

	// ProviderAuto auto-detects the provider based on environment.
	// This is the default and maintains backwards compatibility:
	// - If OPENAI_API_KEY is set, uses OpenAI client
	// - Otherwise returns nil (AI disabled)
	ProviderAuto ProviderType = ""
)

// NewClient creates an AI client based on the provider type and model.
//
// When provider is empty (ProviderAuto), it maintains backwards compatibility:
//   - If OPENAI_API_KEY is set and valid, returns an OpenAI client
//   - Otherwise returns (nil, nil) indicating AI features are disabled
//
// When provider is explicitly set:
//   - "openai": Returns OpenAI client (may have Validated=false if OPENAI_API_KEY not set)
//   - "claude-code": Returns Claude Code client (may have Validated=false if CLI not found)
//
// Explicit providers always return a client so the TUI can display validation errors.
// Returns an error only for invalid/unknown provider values.
func NewClient(provider ProviderType, model string) (Client, error) {
	switch provider {
	case ProviderOpenAI:
		// Explicitly requested OpenAI - returns client even if not validated
		// so TUI can display the validation error
		return NewOpenAIClient(model), nil

	case ProviderClaudeCode:
		// Explicitly requested Claude Code - returns client even if not validated
		// so TUI can display the validation error
		return NewClaudeCodeClient(model), nil

	case ProviderAuto:
		// Auto-detect: maintain backwards compatibility
		// Only try OpenAI (the previous default behavior)
		client := NewOpenAIClient(model)
		if client.Validated {
			return client, nil
		}
		// No OpenAI API key or validation failed - return nil (AI disabled, same as before)
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown AI provider: %q (valid options: %v)", provider, ValidProviders())
	}
}

// ValidProviders returns the list of valid provider type strings
func ValidProviders() []string {
	return []string{string(ProviderOpenAI), string(ProviderClaudeCode)}
}
