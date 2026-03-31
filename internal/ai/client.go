package ai

// ValidationStatus contains the validation state of an AI client.
type ValidationStatus struct {
	Validated    bool
	ErrorMessage string
	ServiceName  string
	ModelName    string
}

// Client defines the interface for all AI providers.
// Implementations include OpenAI-compatible APIs and Claude Code CLI.
type Client interface {
	// AnalyzeLog sends a log message to the AI for analysis
	AnalyzeLog(logMessage, severity, timestamp string, attributes map[string]string) (string, error)

	// AnalyzeLogWithContext sends a log message with chat context for follow-up questions
	AnalyzeLogWithContext(logMessage, severity, timestamp string, attributes map[string]string, previousAnalysis, question string) (string, error)

	// GetValidationStatus returns the validation status of the client
	GetValidationStatus() ValidationStatus

	// GetAvailableModels fetches the list of available models from the provider (makes API call)
	GetAvailableModels() ([]string, error)

	// SetModel updates the active model being used
	SetModel(model string)

	// GetModel returns the current model being used
	GetModel() string

	// CachedModels returns the cached list of available models without making API calls.
	// This is used by the TUI for model selection.
	CachedModels() []string
}
