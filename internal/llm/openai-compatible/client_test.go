package openaicompatible

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anton415/mini-agent-orchestrator/internal/llm"
)

func TestGenerateSendsChatCompletionRequestAndParsesResponse(t *testing.T) {
	temperature := 0.2
	maxTokens := 800

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test-secret" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var body chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Model != "configured-model" {
			t.Fatalf("model = %q, want configured-model", body.Model)
		}
		if len(body.Messages) != 1 {
			t.Fatalf("messages length = %d, want 1", len(body.Messages))
		}
		if body.Messages[0].Role != "user" {
			t.Fatalf("message role = %q, want user", body.Messages[0].Role)
		}
		if body.Messages[0].Content != "Generate spec.md" {
			t.Fatalf("message content = %q, want prompt", body.Messages[0].Content)
		}
		if body.Stream {
			t.Fatal("stream = true, want false")
		}
		if body.Temperature == nil || *body.Temperature != temperature {
			t.Fatalf("temperature = %#v, want %v", body.Temperature, temperature)
		}
		if body.MaxTokens == nil || *body.MaxTokens != maxTokens {
			t.Fatalf("max_tokens = %#v, want %d", body.MaxTokens, maxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"model": "configured-model",
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "# Specification\n\nGenerated markdown."
					}
				}
			],
			"usage": {
				"prompt_tokens": 11,
				"completion_tokens": 22,
				"total_tokens": 33
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL+"/v1")

	got, err := client.Generate(context.Background(), llm.GenerateRequest{
		Prompt:      "Generate spec.md",
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	if got.Content != "# Specification\n\nGenerated markdown." {
		t.Fatalf("Content = %q, want generated markdown", got.Content)
	}
	if got.Provider != llm.ProviderOpenAICompatible {
		t.Fatalf("Provider = %q, want %q", got.Provider, llm.ProviderOpenAICompatible)
	}
	if got.Model != "configured-model" {
		t.Fatalf("Model = %q, want configured-model", got.Model)
	}
	if got.Usage == nil {
		t.Fatal("Usage = nil, want token usage")
	}
	if got.Usage.InputTokens != 11 || got.Usage.OutputTokens != 22 || got.Usage.TotalTokens != 33 {
		t.Fatalf("Usage = %#v, want mapped OpenAI token usage", got.Usage)
	}
}

func TestGenerateAllowsRequestModelOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body.Model != "request-model" {
			t.Fatalf("model = %q, want request-model", body.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [
				{"message": {"role": "assistant", "content": "# Tasks"}}
			]
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	got, err := client.Generate(context.Background(), llm.GenerateRequest{
		Prompt: "Generate tasks.md",
		Model:  "request-model",
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if got.Model != "request-model" {
		t.Fatalf("Model = %q, want fallback to request model", got.Model)
	}
}

func TestGenerateReturnsUsefulErrorForNon2xxResponse(t *testing.T) {
	const secret = "sk-test-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"error": {
				"message": "invalid API key sk-test-secret"
			}
		}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.Generate(context.Background(), llm.GenerateRequest{
		Prompt: "Generate idea.md",
	})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}

	message := err.Error()
	for _, want := range []string{"401 Unauthorized", "invalid API key", "[redacted]"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error = %q, want message containing %q", message, want)
		}
	}
	if strings.Contains(message, secret) {
		t.Fatalf("error leaked API key: %q", message)
	}
}

func TestGenerateRedactsSecretBeforeTruncatingLongNon2xxResponse(t *testing.T) {
	const secret = "secret-value-that-crosses-boundary"
	longPrefix := strings.Repeat("x", 4090)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": longPrefix + secret,
			},
		})
	}))
	defer server.Close()

	client := newTestClientWithAPIKey(t, server.URL, secret)

	_, err := client.Generate(context.Background(), llm.GenerateRequest{
		Prompt: "Generate idea.md",
	})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}

	message := err.Error()
	if strings.Contains(message, secret) {
		t.Fatalf("error leaked full API key: %q", message)
	}
	if leakedPrefix := secret[:6]; strings.Contains(message, leakedPrefix) {
		t.Fatalf("error leaked API key prefix %q: %q", leakedPrefix, message)
	}
	if !strings.Contains(message, "...[truncated]") {
		t.Fatalf("error = %q, want truncated marker", message)
	}
}

func TestGenerateRejectsMalformedSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	_, err := client.Generate(context.Background(), llm.GenerateRequest{
		Prompt: "Generate checklist",
	})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !strings.Contains(err.Error(), "missing choices") {
		t.Fatalf("error = %q, want missing choices message", err.Error())
	}
}

func TestGenerateRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(200 * time.Millisecond):
			_, _ = w.Write([]byte(`{
				"choices": [
					{"message": {"role": "assistant", "content": "# Late"}}
				]
			}`))
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Generate(ctx, llm.GenerateRequest{
		Prompt: "Generate idea.md",
	})
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want context deadline exceeded", err)
	}
}

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	return newTestClientWithAPIKey(t, baseURL, "sk-test-secret")
}

func newTestClientWithAPIKey(t *testing.T, baseURL string, apiKey string) *Client {
	t.Helper()

	client, err := NewClient(llm.Config{
		Enabled:  true,
		Provider: llm.ProviderOpenAICompatible,
		BaseURL:  baseURL,
		Model:    "configured-model",
		APIKey:   apiKey,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	return client
}
