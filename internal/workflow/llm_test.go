package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/anton415/mini-agent-orchestrator/internal/llm"
)

type fakeLLMProvider struct {
	request  llm.GenerateRequest
	response llm.GenerateResponse
	err      error
}

var _ llm.Provider = (*fakeLLMProvider)(nil)

func (provider *fakeLLMProvider) Generate(ctx context.Context, req llm.GenerateRequest) (llm.GenerateResponse, error) {
	provider.request = req
	if provider.err != nil {
		return llm.GenerateResponse{}, provider.err
	}

	return provider.response, nil
}

func TestLLMGeneratorUsesProviderInterface(t *testing.T) {
	temperature := 0.2
	maxTokens := 800
	provider := &fakeLLMProvider{
		response: llm.GenerateResponse{
			Content:  "# Specification",
			Provider: "fake",
			Model:    "test-model",
		},
	}
	generator := LLMGenerator{
		Provider:    provider,
		Model:       "test-model",
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
	}

	response, err := generator.Generate(context.Background(), "Generate spec.md")
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if response.Content != "# Specification" {
		t.Fatalf("Content = %q, want %q", response.Content, "# Specification")
	}
	if provider.request.Prompt != "Generate spec.md" {
		t.Fatalf("Prompt = %q, want %q", provider.request.Prompt, "Generate spec.md")
	}
	if provider.request.Model != "test-model" {
		t.Fatalf("Model = %q, want %q", provider.request.Model, "test-model")
	}
	if provider.request.Temperature == nil || *provider.request.Temperature != temperature {
		t.Fatalf("Temperature = %#v, want %v", provider.request.Temperature, temperature)
	}
	if provider.request.MaxTokens == nil || *provider.request.MaxTokens != maxTokens {
		t.Fatalf("MaxTokens = %#v, want %d", provider.request.MaxTokens, maxTokens)
	}
}

func TestLLMGeneratorRequiresProvider(t *testing.T) {
	_, err := LLMGenerator{}.Generate(context.Background(), "Generate idea.md")
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !strings.Contains(err.Error(), "llm provider is required") {
		t.Fatalf("error = %q, want provider message", err.Error())
	}
}

func TestLLMGeneratorWrapsProviderError(t *testing.T) {
	providerErr := errors.New("provider failed")
	generator := LLMGenerator{
		Provider: &fakeLLMProvider{
			err: providerErr,
		},
	}

	_, err := generator.Generate(context.Background(), "Generate tasks.md")
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want wrapped provider error", err)
	}
	if !strings.Contains(err.Error(), "generate llm content") {
		t.Fatalf("error = %q, want generation context", err.Error())
	}
}
