package openaicompatible

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anton415/mini-agent-orchestrator/internal/llm"
)

const (
	providerName   = llm.ProviderOpenAICompatible
	defaultTimeout = 60 * time.Second
)

// Client sends non-streaming chat completion requests to an OpenAI-compatible API.
type Client struct {
	baseURL    *url.URL
	model      string
	apiKey     string
	httpClient *http.Client
}

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for provider requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		if httpClient != nil {
			client.httpClient = httpClient
		}
	}
}

// NewClient builds an OpenAI-compatible LLM provider from validated LLM config.
func NewClient(cfg llm.Config, options ...Option) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
	if err != nil {
		return nil, fmt.Errorf("parse LLM base URL: %w", err)
	}

	client := &Client{
		baseURL: baseURL,
		model:   strings.TrimSpace(cfg.Model),
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, option := range options {
		option(client)
	}

	return client, nil
}

var _ llm.Provider = (*Client)(nil)

// Generate sends one non-streaming chat completion request and returns the assistant content.
func (client *Client) Generate(ctx context.Context, req llm.GenerateRequest) (llm.GenerateResponse, error) {
	if client == nil {
		return llm.GenerateResponse{}, fmt.Errorf("openai-compatible client is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = client.model
	}
	if model == "" {
		return llm.GenerateResponse{}, fmt.Errorf("openai-compatible model is required")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return llm.GenerateResponse{}, fmt.Errorf("openai-compatible prompt is required")
	}

	payload := chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: req.Prompt,
			},
		},
		Stream:      false,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return llm.GenerateResponse{}, fmt.Errorf("encode chat completion request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, client.chatCompletionsURL(), &body)
	if err != nil {
		return llm.GenerateResponse{}, fmt.Errorf("create chat completion request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+client.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := client.httpClient.Do(httpReq)
	if err != nil {
		return llm.GenerateResponse{}, fmt.Errorf("send chat completion request: %w", err)
	}
	defer httpResp.Body.Close()

	responseBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return llm.GenerateResponse{}, fmt.Errorf("read chat completion response: %w", err)
	}

	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return llm.GenerateResponse{}, fmt.Errorf("openai-compatible chat completion failed: %s: %s", httpResp.Status, client.errorMessage(responseBody))
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return llm.GenerateResponse{}, fmt.Errorf("decode chat completion response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return llm.GenerateResponse{}, fmt.Errorf("decode chat completion response: missing choices")
	}

	content := parsed.Choices[0].Message.Content
	if content == "" {
		return llm.GenerateResponse{}, fmt.Errorf("decode chat completion response: missing assistant content")
	}

	responseModel := parsed.Model
	if responseModel == "" {
		responseModel = model
	}

	return llm.GenerateResponse{
		Content:  content,
		Provider: providerName,
		Model:    responseModel,
		Usage:    parsed.Usage.toLLMUsage(),
	}, nil
}

func (client *Client) chatCompletionsURL() string {
	base := *client.baseURL
	base.Path = strings.TrimRight(base.Path, "/") + "/chat/completions"
	base.RawQuery = ""
	base.Fragment = ""

	return base.String()
}

func (client *Client) errorMessage(body []byte) string {
	var errorResponse errorResponse
	if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Error.Message != "" {
		return client.redact(errorResponse.Error.Message)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty response body"
	}

	return client.redact(message)
}

func (client *Client) redact(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 4096 {
		message = message[:4096] + "...[truncated]"
	}
	if client.apiKey != "" {
		message = strings.ReplaceAll(message, client.apiKey, "[redacted]")
	}

	return message
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   *usage   `json:"usage,omitempty"`
}

type choice struct {
	Message chatMessage `json:"message"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (u *usage) toLLMUsage() *llm.Usage {
	if u == nil {
		return nil
	}

	return &llm.Usage{
		InputTokens:  u.PromptTokens,
		OutputTokens: u.CompletionTokens,
		TotalTokens:  u.TotalTokens,
	}
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
