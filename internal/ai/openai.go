// Package ai provides AI-powered log analysis features.
package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAIClient handles OpenAI API compatible requests
type OpenAIClient struct {
	APIKey          string
	BaseURL         string
	Model           string
	HTTPClient      *http.Client
	Validated       bool
	ValidationErr   string
	ServiceName     string
	AvailableModels []string
	AutoSelectModel bool // True if no model was specified, should auto-select
}

// OpenAIRequest represents the request structure
type OpenAIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the API response with flexible error handling
type OpenAIResponse struct {
	Choices []Choice `json:"choices"`
	Error   any      `json:"error,omitempty"` // Can be string or object
}

// Choice represents a response choice
type Choice struct {
	Message Message `json:"message"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ModelListResponse represents the /v1/models endpoint response
type ModelListResponse struct {
	Data []ModelInfo `json:"data"`
}

// ModelInfo represents a single model in the list
type ModelInfo struct {
	ID     string `json:"id"`
	Object string `json:"object"`
}

// OllamaTagsResponse represents Ollama's /api/tags response
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaModel represents a model in Ollama's response
type OllamaModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// OllamaGenerateRequest represents Ollama's /api/generate request
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaGenerateResponse represents Ollama's /api/generate response
type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// getErrorMessage extracts error message from flexible error field
func (r *OpenAIResponse) getErrorMessage() string {
	if r.Error == nil {
		return ""
	}

	// Handle string error format (like LM Studio)
	if errStr, ok := r.Error.(string); ok {
		return errStr
	}

	// Handle object error format (like OpenAI)
	if errMap, ok := r.Error.(map[string]any); ok {
		if message, exists := errMap["message"]; exists {
			if msgStr, ok := message.(string); ok {
				return msgStr
			}
		}
		// Fallback to JSON representation
		jsonBytes, _ := json.Marshal(r.Error)
		return string(jsonBytes)
	}

	// Fallback to string representation
	return fmt.Sprintf("%v", r.Error)
}

// NewOpenAIClient creates a new OpenAI client with environment variable detection
func NewOpenAIClient(model string) *OpenAIClient {
	// Detect API key from standard environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil // No API key found
	}

	// Detect base URL from environment variable, default to OpenAI's endpoint
	baseURL := os.Getenv("OPENAI_API_BASE")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// We'll set the default model after getting available models in ValidateConfiguration
	// For now, keep track if no model was specified
	autoSelectModel := model == ""

	// Determine service name from base URL
	serviceName := "OpenAI"
	if baseURL != "https://api.openai.com/v1" {
		if strings.Contains(baseURL, "localhost") || strings.Contains(baseURL, "127.0.0.1") {
			if strings.Contains(baseURL, "1234") {
				serviceName = "LM Studio"
			} else if strings.Contains(baseURL, "11434") {
				serviceName = "Ollama"
			} else {
				serviceName = "Local AI"
			}
		} else {
			serviceName = "Custom API"
		}
	}

	client := &OpenAIClient{
		APIKey:          apiKey,
		BaseURL:         baseURL,
		Model:           model,
		ServiceName:     serviceName,
		AutoSelectModel: autoSelectModel,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Validate configuration and get available models
	client.ValidateConfiguration()

	return client
}

// AnalyzeLog sends a log message to the AI for analysis
func (c *OpenAIClient) AnalyzeLog(logMessage, severity, timestamp string, attributes map[string]string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("OpenAI client not configured (missing OPENAI_API_KEY)")
	}

	prompt := c.buildAnalysisPrompt(logMessage, severity, timestamp, attributes)

	// Try Ollama native API first if we detect it's Ollama
	if c.ServiceName == "Ollama" {
		result, err := c.analyzeWithOllama(prompt)
		if err == nil {
			return result, nil
		}
		// If Ollama native API fails, continue to try OpenAI-compatible API
	}

	request := OpenAIRequest{
		Model: c.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for flexible parsing
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Try parsing as standard OpenAI response
	var response OpenAIResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		// Try parsing as flexible structure for compatibility
		var flexResponse map[string]any
		if flexErr := json.Unmarshal(bodyBytes, &flexResponse); flexErr != nil {
			return "", fmt.Errorf("failed to decode response: %v (body: %s)", err, string(bodyBytes))
		}

		// Try to extract response manually from flexible structure
		if choices, ok := flexResponse["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if message, ok := choice["message"].(map[string]any); ok {
					if content, ok := message["content"].(string); ok {
						return content, nil
					}
				}
			}
		}

		return "", fmt.Errorf("failed to parse response structure: %v (body: %s)", err, string(bodyBytes))
	}

	if errorMsg := response.getErrorMessage(); errorMsg != "" {
		return "", fmt.Errorf("AI API error (model=%s, url=%s): %s", c.Model, c.BaseURL, errorMsg)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return response.Choices[0].Message.Content, nil
}

// AnalyzeLogWithContext sends a log message to the AI with chat context
func (c *OpenAIClient) AnalyzeLogWithContext(logMessage, severity, timestamp string, attributes map[string]string, previousAnalysis string, question string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("OpenAI client not configured (missing OPENAI_API_KEY)")
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

	// Try Ollama native API first if we detect it's Ollama
	if c.ServiceName == "Ollama" {
		result, err := c.analyzeWithOllama(prompt)
		if err == nil {
			return result, nil
		}
		// If Ollama native API fails, continue to try OpenAI-compatible API
	}

	request := OpenAIRequest{
		Model: c.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for flexible parsing
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Try parsing as standard OpenAI response
	var response OpenAIResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		// Try parsing as flexible structure for compatibility
		var flexResponse map[string]any
		if flexErr := json.Unmarshal(bodyBytes, &flexResponse); flexErr != nil {
			return "", fmt.Errorf("failed to decode response: %v (body: %s)", err, string(bodyBytes))
		}

		// Try to extract response manually from flexible structure
		if choices, ok := flexResponse["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				if message, ok := choice["message"].(map[string]any); ok {
					if content, ok := message["content"].(string); ok {
						return content, nil
					}
				}
			}
		}

		return "", fmt.Errorf("failed to parse response structure: %v (body: %s)", err, string(bodyBytes))
	}

	if errorMsg := response.getErrorMessage(); errorMsg != "" {
		return "", fmt.Errorf("AI API error (model=%s, url=%s): %s", c.Model, c.BaseURL, errorMsg)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return response.Choices[0].Message.Content, nil
}

// buildAnalysisPrompt creates the analysis prompt for the log message
func (c *OpenAIClient) buildAnalysisPrompt(logMessage, severity, timestamp string, attributes map[string]string) string {
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

// ValidateConfiguration checks if the AI client is properly configured
func (c *OpenAIClient) ValidateConfiguration() {
	if c == nil {
		return
	}

	// Get available models to validate the endpoint
	models, err := c.GetAvailableModels()
	if err != nil {
		c.Validated = false
		c.ValidationErr = fmt.Sprintf("Failed to connect: %v", err)
		return
	}

	c.AvailableModels = models

	// Handle auto-selection when no model was specified
	if c.AutoSelectModel {
		if len(models) == 0 {
			c.Validated = false
			c.ValidationErr = "No models available from AI service"
			return
		}

		// Smart model selection: prefer common models or pick first available
		selectedModel := c.selectBestDefaultModel(models)
		c.Model = selectedModel
		c.AutoSelectModel = false // Reset flag after selection
	} else {
		// Check if the specified model exists, or find a close match
		originalModel := c.Model
		matchedModel := c.findBestModelMatch(c.Model, models)
		if matchedModel == "" {
			c.Validated = false
			c.ValidationErr = fmt.Sprintf("Model '%s' not found", c.Model)
			return
		}

		// Update to matched model if different
		if matchedModel != originalModel {
			c.Model = matchedModel
		}
	}

	c.Validated = true
	c.ValidationErr = ""
}

// GetAvailableModels fetches the list of available models from the API
func (c *OpenAIClient) GetAvailableModels() ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	// Try Ollama native API first if we detect it's Ollama
	if c.ServiceName == "Ollama" {
		models, err := c.getOllamaModels()
		if err == nil && len(models) > 0 {
			return models, nil
		}
		// If Ollama native API fails, continue to try OpenAI-compatible API
	}

	req, err := http.NewRequest("GET", c.BaseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var modelList ModelListResponse
	if err := json.Unmarshal(bodyBytes, &modelList); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var models []string
	for _, model := range modelList.Data {
		models = append(models, model.ID)
	}

	return models, nil
}

// getOllamaModels fetches models using Ollama's native /api/tags endpoint
func (c *OpenAIClient) getOllamaModels() ([]string, error) {
	// Remove /v1 suffix if present for Ollama native API
	baseURL := strings.TrimSuffix(c.BaseURL, "/v1")

	req, err := http.NewRequest("GET", baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var tagsResponse OllamaTagsResponse
	if err := json.Unmarshal(bodyBytes, &tagsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %v", err)
	}

	var models []string
	for _, model := range tagsResponse.Models {
		models = append(models, model.Name)
	}

	return models, nil
}

// analyzeWithOllama uses Ollama's native /api/generate endpoint
func (c *OpenAIClient) analyzeWithOllama(prompt string) (string, error) {
	// Remove /v1 suffix if present for Ollama native API
	baseURL := strings.TrimSuffix(c.BaseURL, "/v1")

	request := OllamaGenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var response OllamaGenerateResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %v", err)
	}

	return response.Response, nil
}

// selectBestDefaultModel selects the best default model from available models
func (c *OpenAIClient) selectBestDefaultModel(availableModels []string) string {
	if len(availableModels) == 0 {
		return ""
	}

	// Preferred models in order of preference
	preferredModels := []string{
		"gpt-4", "gpt-4-turbo", "gpt-4o", "gpt-4o-mini", // OpenAI GPT-4 variants
		"gpt-3.5-turbo", "gpt-3.5-turbo-16k", // OpenAI GPT-3.5 variants
		"gpt-oss:20b", "gpt-oss:7b", "gpt-oss", // OSS GPT models (common in Ollama)
		"llama3", "llama3.1", "llama3:8b", "llama3:70b", // Ollama Llama variants
		"mistral", "mistral:7b", "mistral:latest", // Ollama Mistral variants
		"codellama", "codellama:7b", "codellama:13b", // Ollama CodeLlama variants
	}

	// First, try to find any preferred model (exact match)
	for _, preferred := range preferredModels {
		for _, available := range availableModels {
			if available == preferred {
				return available
			}
		}
	}

	// Second, try case-insensitive partial matches with preferred models
	for _, preferred := range preferredModels {
		lowerPreferred := strings.ToLower(preferred)
		for _, available := range availableModels {
			lowerAvailable := strings.ToLower(available)
			if strings.Contains(lowerAvailable, lowerPreferred) {
				return available
			}
		}
	}

	// Fallback: return the first available model
	return availableModels[0]
}

// findBestModelMatch finds the best matching model from available models
func (c *OpenAIClient) findBestModelMatch(requestedModel string, availableModels []string) string {
	// First try exact match
	for _, model := range availableModels {
		if model == requestedModel {
			return model
		}
	}

	// Try case-insensitive exact match
	lowerRequested := strings.ToLower(requestedModel)
	for _, model := range availableModels {
		if strings.ToLower(model) == lowerRequested {
			return model
		}
	}

	// Try partial match (requested model is contained in available model)
	for _, model := range availableModels {
		if strings.Contains(strings.ToLower(model), lowerRequested) {
			return model
		}
	}

	// Try partial match (available model is contained in requested model)
	for _, model := range availableModels {
		if strings.Contains(lowerRequested, strings.ToLower(model)) {
			return model
		}
	}

	// No match found
	return ""
}

// GetValidationStatus returns the validation status and any error message
func (c *OpenAIClient) GetValidationStatus() (bool, string, string, string) {
	if c == nil {
		return false, "No API key configured", "None", ""
	}
	return c.Validated, c.ValidationErr, c.ServiceName, c.Model
}
