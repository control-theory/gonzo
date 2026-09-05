package ai

import (
	"slices"
	"testing"
)

func TestNewLiteLLMClient_NoAPIKeyReturnsUnvalidated(t *testing.T) {
	t.Setenv("LITELLM_API_KEY", "")

	client := NewLiteLLMClient("")

	if client.Validated {
		t.Fatal("expected unvalidated client when LITELLM_API_KEY is unset")
	}
	if client.ServiceName != "LiteLLM" {
		t.Fatalf("expected ServiceName %q, got %q", "LiteLLM", client.ServiceName)
	}
	if client.ValidationErr == "" {
		t.Fatal("expected a validation error message when LITELLM_API_KEY is unset")
	}
}

func TestNewLiteLLMClient_DefaultBaseURL(t *testing.T) {
	// Set a key so the constructor proceeds past the no-key guard. Validation
	// makes a network call and may fail (no proxy running), but BaseURL/defaults
	// are assigned before that.
	t.Setenv("LITELLM_API_KEY", "sk-test")
	t.Setenv("LITELLM_API_BASE", "")

	client := NewLiteLLMClient("some-model")

	if client.BaseURL != defaultLiteLLMBaseURL {
		t.Fatalf("expected default BaseURL %q, got %q", defaultLiteLLMBaseURL, client.BaseURL)
	}
	if client.APIKey != "sk-test" {
		t.Fatalf("expected APIKey to be forwarded, got %q", client.APIKey)
	}
	if client.ServiceName != "LiteLLM" {
		t.Fatalf("expected ServiceName %q, got %q", "LiteLLM", client.ServiceName)
	}
	if client.Model != "some-model" {
		t.Fatalf("expected Model %q, got %q", "some-model", client.Model)
	}
}

func TestNewLiteLLMClient_CustomBaseURL(t *testing.T) {
	t.Setenv("LITELLM_API_KEY", "sk-test")
	t.Setenv("LITELLM_API_BASE", "http://127.0.0.1:4000/v1")

	client := NewLiteLLMClient("m")

	if client.BaseURL != "http://127.0.0.1:4000/v1" {
		t.Fatalf("expected custom BaseURL to be honored, got %q", client.BaseURL)
	}
}

func TestNewClient_LiteLLMProvider(t *testing.T) {
	t.Setenv("LITELLM_API_KEY", "")

	client, err := NewClient(ProviderLiteLLM, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected a non-nil client for explicit litellm provider")
	}
	if got := client.GetValidationStatus().ServiceName; got != "LiteLLM" {
		t.Fatalf("expected ServiceName %q, got %q", "LiteLLM", got)
	}
}

func TestValidProviders_IncludesLiteLLM(t *testing.T) {
	if !slices.Contains(ValidProviders(), "litellm") {
		t.Fatalf("expected ValidProviders to include %q, got %v", "litellm", ValidProviders())
	}
}
