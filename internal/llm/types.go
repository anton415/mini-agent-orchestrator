package llm

// GenerateRequest contains provider-agnostic settings for one generation call.
type GenerateRequest struct {
	Prompt      string
	Model       string
	Temperature *float64
	MaxTokens   *int
}

// GenerateResponse contains generated content and optional provider metadata.
type GenerateResponse struct {
	Content  string
	Provider string
	Model    string
	Usage    *Usage
}

// Usage contains token accounting when a provider returns it.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}
