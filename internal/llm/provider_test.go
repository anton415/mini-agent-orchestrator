package llm

import (
	"context"
	"testing"
)

type staticProvider struct {
	response GenerateResponse
}

var _ Provider = staticProvider{}

func (provider staticProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	return provider.response, nil
}

func TestProviderInterfaceUsesProviderAgnosticTypes(t *testing.T) {
	provider := staticProvider{
		response: GenerateResponse{
			Content:  "# Idea",
			Provider: "test-provider",
			Model:    "test-model",
			Usage: &Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	response, err := provider.Generate(context.Background(), GenerateRequest{
		Prompt: "Generate idea.md",
		Model:  "test-model",
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if response.Content != "# Idea" {
		t.Fatalf("Content = %q, want %q", response.Content, "# Idea")
	}
	if response.Provider != "test-provider" {
		t.Fatalf("Provider = %q, want %q", response.Provider, "test-provider")
	}
	if response.Model != "test-model" {
		t.Fatalf("Model = %q, want %q", response.Model, "test-model")
	}
	if response.Usage == nil {
		t.Fatal("Usage = nil, want token metadata")
	}
	if response.Usage.TotalTokens != 30 {
		t.Fatalf("TotalTokens = %d, want %d", response.Usage.TotalTokens, 30)
	}
}
