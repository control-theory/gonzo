package ai

import (
	"net/http"
	"os"
	"time"
)

// LiteLLM proxy defaults. A LiteLLM proxy speaks the OpenAI wire format, so the
// LiteLLM provider reuses the tested OpenAIClient and only differs in its
// defaults and environment variables. Point it at your proxy to reach 100+
// providers (OpenAI, Anthropic, Gemini, Bedrock, Vertex AI, Azure, ...) through
// a single endpoint with unified auth, routing and observability.
const defaultLiteLLMBaseURL = "http://localhost:4000/v1"

// NewLiteLLMClient creates a client for a LiteLLM proxy.
//
// Configuration (mirrors the OpenAI provider, with LiteLLM-specific env vars):
//   - LITELLM_API_KEY:  the proxy master/virtual key (required)
//   - LITELLM_API_BASE: the proxy base URL (default http://localhost:4000/v1)
//
// LiteLLM is OpenAI-wire compatible, so this reuses OpenAIClient (and therefore
// its chat, model-discovery, validation and retry behavior) verbatim.
func NewLiteLLMClient(model string) *OpenAIClient {
	apiKey := os.Getenv("LITELLM_API_KEY")
	if apiKey == "" {
		return &OpenAIClient{
			Validated:     false,
			ValidationErr: "LITELLM_API_KEY environment variable not set",
			ServiceName:   "LiteLLM",
		}
	}

	baseURL := os.Getenv("LITELLM_API_BASE")
	if baseURL == "" {
		baseURL = defaultLiteLLMBaseURL
	}

	client := &OpenAIClient{
		APIKey:          apiKey,
		BaseURL:         baseURL,
		Model:           model,
		ServiceName:     "LiteLLM",
		AutoSelectModel: model == "",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Validate configuration and get available models (via the proxy's /models).
	client.ValidateConfiguration()

	return client
}
