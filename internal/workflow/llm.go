package workflow

import (
	"context"
	"fmt"

	"github.com/anton415/mini-agent-orchestrator/internal/llm"
)

// LLMGenerator adapts workflow prompt execution to the provider boundary.
type LLMGenerator struct {
	Provider    llm.Provider
	Model       string
	Temperature *float64
	MaxTokens   *int
}

// Generate sends one workflow prompt through the configured provider.
func (generator LLMGenerator) Generate(ctx context.Context, prompt string) (llm.GenerateResponse, error) {
	if generator.Provider == nil {
		return llm.GenerateResponse{}, fmt.Errorf("llm provider is required")
	}
	if prompt == "" {
		return llm.GenerateResponse{}, fmt.Errorf("llm prompt is required")
	}

	response, err := generator.Provider.Generate(ctx, llm.GenerateRequest{
		Prompt:      prompt,
		Model:       generator.Model,
		Temperature: generator.Temperature,
		MaxTokens:   generator.MaxTokens,
	})
	if err != nil {
		return llm.GenerateResponse{}, fmt.Errorf("generate llm content: %w", err)
	}

	return response, nil
}
