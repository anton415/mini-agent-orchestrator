package llm

import "context"

// Provider generates content for one fixed workflow prompt.
type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
}
