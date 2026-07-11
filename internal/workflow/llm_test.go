package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/anton415/mini-agent-orchestrator/internal/artifacts"
	"github.com/anton415/mini-agent-orchestrator/internal/llm"
)

type fakeLLMProvider struct {
	request   llm.GenerateRequest
	requests  []llm.GenerateRequest
	response  llm.GenerateResponse
	responses []llm.GenerateResponse
	err       error
	errors    map[int]error
}

var _ llm.Provider = (*fakeLLMProvider)(nil)

func (provider *fakeLLMProvider) Generate(ctx context.Context, req llm.GenerateRequest) (llm.GenerateResponse, error) {
	provider.request = req
	provider.requests = append(provider.requests, req)
	callIndex := len(provider.requests) - 1

	if err := provider.errors[callIndex]; err != nil {
		return llm.GenerateResponse{}, err
	}
	if provider.err != nil {
		return llm.GenerateResponse{}, provider.err
	}
	if callIndex < len(provider.responses) {
		return provider.responses[callIndex], nil
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

func TestLLMGeneratorRequiresNonBlankPrompt(t *testing.T) {
	provider := &fakeLLMProvider{}

	_, err := (LLMGenerator{Provider: provider}).Generate(context.Background(), " \n\t")
	if err == nil {
		t.Fatal("Generate returned nil error")
	}
	if !strings.Contains(err.Error(), "llm prompt is required") {
		t.Fatalf("error = %q, want prompt message", err.Error())
	}
	if len(provider.requests) != 0 {
		t.Fatalf("provider calls = %d, want 0", len(provider.requests))
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

func TestExecuteLLMWorkflowRunsFixedPipelineSequentially(t *testing.T) {
	promptItems := testWorkflowPrompts()
	responses := testWorkflowResponses()
	provider := &fakeLLMProvider{responses: responses}

	generatedItems, executedPrompts, err := executeLLMWorkflow(
		context.Background(),
		promptItems,
		LLMGenerator{Provider: provider, Model: "test-model"},
	)
	if err != nil {
		t.Fatalf("executeLLMWorkflow returned error: %v", err)
	}

	wantOutputNames := []string{"idea.md", "spec.md", "tasks.md", "review-checklist.md"}
	if len(generatedItems) != len(wantOutputNames) {
		t.Fatalf("generated artifacts = %d, want %d", len(generatedItems), len(wantOutputNames))
	}
	if len(executedPrompts) != len(promptItems) {
		t.Fatalf("executed prompts = %d, want %d", len(executedPrompts), len(promptItems))
	}
	if len(provider.requests) != len(promptItems) {
		t.Fatalf("provider calls = %d, want %d", len(provider.requests), len(promptItems))
	}

	for stageIndex := range wantOutputNames {
		if generatedItems[stageIndex].Filename != wantOutputNames[stageIndex] {
			t.Errorf(
				"generated artifact %d filename = %q, want %q",
				stageIndex,
				generatedItems[stageIndex].Filename,
				wantOutputNames[stageIndex],
			)
		}
		if generatedItems[stageIndex].Content != responses[stageIndex].Content {
			t.Errorf(
				"generated artifact %q content = %q, want exact provider content %q",
				wantOutputNames[stageIndex],
				generatedItems[stageIndex].Content,
				responses[stageIndex].Content,
			)
		}
		if executedPrompts[stageIndex].Filename != promptItems[stageIndex].Filename {
			t.Errorf(
				"executed prompt %d filename = %q, want %q",
				stageIndex,
				executedPrompts[stageIndex].Filename,
				promptItems[stageIndex].Filename,
			)
		}
		if executedPrompts[stageIndex].Content != provider.requests[stageIndex].Prompt {
			t.Errorf("executed prompt %d does not equal the prompt sent to the provider", stageIndex)
		}

		for upstreamIndex := 0; upstreamIndex < stageIndex; upstreamIndex++ {
			for _, want := range []string{
				wantOutputNames[upstreamIndex],
				responses[upstreamIndex].Content,
			} {
				if !strings.Contains(provider.requests[stageIndex].Prompt, want) {
					t.Errorf(
						"stage %q prompt does not contain upstream value %q\nprompt:\n%s",
						wantOutputNames[stageIndex],
						want,
						provider.requests[stageIndex].Prompt,
					)
				}
			}
		}
		for futureIndex := stageIndex; futureIndex < len(responses); futureIndex++ {
			if strings.Contains(provider.requests[stageIndex].Prompt, responses[futureIndex].Content) {
				t.Errorf(
					"stage %q prompt unexpectedly contains current or future output %q",
					wantOutputNames[stageIndex],
					responses[futureIndex].Content,
				)
			}
		}
	}

	if executedPrompts[0].Content != promptItems[0].Content {
		t.Fatalf("first executed prompt = %q, want unchanged base prompt %q", executedPrompts[0].Content, promptItems[0].Content)
	}
	for index, promptItem := range promptItems {
		if promptItem.Content != testWorkflowPrompts()[index].Content {
			t.Errorf("input prompt %d was mutated", index)
		}
	}
}

func TestExecuteLLMWorkflowWrapsProviderErrorWithStage(t *testing.T) {
	providerErr := errors.New("provider unavailable")
	provider := &fakeLLMProvider{
		responses: testWorkflowResponses(),
		errors: map[int]error{
			2: providerErr,
		},
	}

	generatedItems, executedPrompts, err := executeLLMWorkflow(
		context.Background(),
		testWorkflowPrompts(),
		LLMGenerator{Provider: provider},
	)
	if err == nil {
		t.Fatal("executeLLMWorkflow returned nil error")
	}
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v, want wrapped provider error", err)
	}
	if !strings.Contains(err.Error(), "tasks.md") {
		t.Fatalf("error = %q, want failing stage name", err.Error())
	}
	if generatedItems != nil || executedPrompts != nil {
		t.Fatalf("partial results = (%#v, %#v), want nil results", generatedItems, executedPrompts)
	}
	if len(provider.requests) != 3 {
		t.Fatalf("provider calls = %d, want 3", len(provider.requests))
	}
}

func TestExecuteLLMWorkflowRejectsEmptyGeneratedContentWithStage(t *testing.T) {
	responses := testWorkflowResponses()
	responses[1].Content = " \n\t"
	provider := &fakeLLMProvider{responses: responses}

	generatedItems, executedPrompts, err := executeLLMWorkflow(
		context.Background(),
		testWorkflowPrompts(),
		LLMGenerator{Provider: provider},
	)
	if err == nil {
		t.Fatal("executeLLMWorkflow returned nil error")
	}
	for _, want := range []string{"spec.md", "empty content"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
	if generatedItems != nil || executedPrompts != nil {
		t.Fatalf("partial results = (%#v, %#v), want nil results", generatedItems, executedPrompts)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(provider.requests))
	}
}

func TestExecuteLLMWorkflowValidatesFixedPromptSetBeforeProviderCalls(t *testing.T) {
	tests := []struct {
		name        string
		promptItems []artifacts.Artifact
		wantError   string
	}{
		{
			name:        "missing prompt",
			promptItems: testWorkflowPrompts()[:3],
			wantError:   "got 3 prompt artifacts, want 4",
		},
		{
			name: "wrong prompt order",
			promptItems: func() []artifacts.Artifact {
				items := testWorkflowPrompts()
				items[1], items[2] = items[2], items[1]
				return items
			}(),
			wantError: "spec.md",
		},
		{
			name: "empty prompt",
			promptItems: func() []artifacts.Artifact {
				items := testWorkflowPrompts()
				items[2].Content = " \n\t"
				return items
			}(),
			wantError: "tasks.md",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &fakeLLMProvider{responses: testWorkflowResponses()}

			generatedItems, executedPrompts, err := executeLLMWorkflow(
				context.Background(),
				test.promptItems,
				LLMGenerator{Provider: provider},
			)
			if err == nil {
				t.Fatal("executeLLMWorkflow returned nil error")
			}
			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %q, want message containing %q", err.Error(), test.wantError)
			}
			if generatedItems != nil || executedPrompts != nil {
				t.Fatalf("results = (%#v, %#v), want nil results", generatedItems, executedPrompts)
			}
			if len(provider.requests) != 0 {
				t.Fatalf("provider calls = %d, want 0", len(provider.requests))
			}
		})
	}
}

func testWorkflowPrompts() []artifacts.Artifact {
	return []artifacts.Artifact{
		{Filename: "prompts/01-normalize-idea.prompt.md", Content: "normalize idea prompt"},
		{Filename: "prompts/02-generate-spec.prompt.md", Content: "generate spec prompt\n"},
		{Filename: "prompts/03-generate-tasks.prompt.md", Content: "generate tasks prompt"},
		{Filename: "prompts/04-review-checklist.prompt.md", Content: "generate review checklist prompt\n"},
	}
}

func testWorkflowResponses() []llm.GenerateResponse {
	return []llm.GenerateResponse{
		{Content: "# Idea\n\nunique generated idea\n", Provider: "fake", Model: "test-model"},
		{Content: "\n# Specification\n\nunique generated specification\n", Provider: "fake", Model: "test-model"},
		{Content: "# Tasks\n\nunique generated tasks", Provider: "fake", Model: "test-model"},
		{Content: "# Review Checklist\n\nunique generated review checklist\n", Provider: "fake", Model: "test-model"},
	}
}
