package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/anton415/mini-agent-orchestrator/internal/cli"
	"github.com/anton415/mini-agent-orchestrator/internal/llm"
	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

func TestRunWritesPromptArtifacts(t *testing.T) {
	setInvalidLLMEnv(t)

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea:           "Build a personal book library",
		Out:            outDir,
		Name:           "book-library",
		CreatedAt:      time.Date(2026, 6, 25, 11, 23, 10, 0, time.UTC),
		IncludePrompts: true,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	projectDir := filepath.Join(outDir, cfg.Name)
	promptFiles := []string{
		"01-normalize-idea.prompt.md",
		"02-generate-spec.prompt.md",
		"03-generate-tasks.prompt.md",
		"04-review-checklist.prompt.md",
	}

	for _, filename := range promptFiles {
		path := filepath.Join(projectDir, "prompts", filename)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}

		content := string(data)
		for _, want := range []string{cfg.Name, cfg.Idea, "## Expected output", "## Constraints"} {
			if !strings.Contains(content, want) {
				t.Errorf("%s does not contain %q\ncontent:\n%s", filename, want, content)
			}
		}
	}
}

func TestRunOmitsPromptArtifactsByDefault(t *testing.T) {
	setInvalidLLMEnv(t)

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea:      "Build a personal book library",
		Out:       outDir,
		Name:      "book-library",
		CreatedAt: time.Date(2026, 6, 25, 11, 23, 10, 0, time.UTC),
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	promptsDir := filepath.Join(outDir, cfg.Name, "prompts")
	if _, err := os.Stat(promptsDir); !os.IsNotExist(err) {
		t.Fatalf("prompts directory stat error = %v, want not exist", err)
	}
}

func TestRunExecutesLLMWorkflowAndWritesExecutedPrompts(t *testing.T) {
	const apiKey = "sk-metadata-must-not-contain-this-secret"
	responses := []string{
		"# Generated Idea\n\nUNIQUE_IDEA_OUTPUT",
		"# Generated Specification\n\nUNIQUE_SPEC_OUTPUT",
		"# Generated Tasks\n\nUNIQUE_TASKS_OUTPUT",
		"# Generated Review Checklist\n\nUNIQUE_REVIEW_OUTPUT",
	}
	server, capturedPrompts := newWorkflowLLMServer(t, responses, -1)

	clearLLMEnv(t)
	t.Setenv(llm.EnvAPIKey, apiKey)

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea:           "Build a personal book library",
		Out:            outDir,
		Name:           "book-library",
		CreatedAt:      time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC),
		IncludePrompts: true,
		LLM:            true,
		LLMProvider:    llm.ProviderOpenAICompatible,
		LLMBaseURL:     server.URL + "/v1",
		LLMModel:       "workflow-test-model",
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	promptsSent := capturedPrompts()
	if len(promptsSent) != len(responses) {
		t.Fatalf("provider calls = %d, want %d", len(promptsSent), len(responses))
	}

	projectDir := filepath.Join(outDir, cfg.Name)
	outputFiles := []string{"idea.md", "spec.md", "tasks.md", "review-checklist.md"}
	promptFiles := []string{
		"01-normalize-idea.prompt.md",
		"02-generate-spec.prompt.md",
		"03-generate-tasks.prompt.md",
		"04-review-checklist.prompt.md",
	}

	for index, filename := range outputFiles {
		assertWorkflowFileContent(t, filepath.Join(projectDir, filename), responses[index])
		assertWorkflowFileContent(
			t,
			filepath.Join(projectDir, "prompts", promptFiles[index]),
			promptsSent[index],
		)

		for upstreamIndex := 0; upstreamIndex < index; upstreamIndex++ {
			if !strings.Contains(promptsSent[index], responses[upstreamIndex]) {
				t.Errorf("stage %s prompt does not contain upstream %s output", filename, outputFiles[upstreamIndex])
			}
		}
		for futureIndex := index; futureIndex < len(responses); futureIndex++ {
			if strings.Contains(promptsSent[index], responses[futureIndex]) {
				t.Errorf("stage %s prompt unexpectedly contains current or future %s output", filename, outputFiles[futureIndex])
			}
		}
	}

	metadataBytes, err := os.ReadFile(filepath.Join(projectDir, "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}
	for _, forbidden := range []string{apiKey, server.URL} {
		if strings.Contains(string(metadataBytes), forbidden) {
			t.Fatalf("metadata.json contains secret provider configuration %q", forbidden)
		}
	}

	var metadata model.Project
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		t.Fatalf("decode metadata.json: %v", err)
	}
	if metadata.Generation == nil {
		t.Fatal("Generation = nil, want LLM generation metadata")
	}
	if metadata.Generation.Mode != model.GenerationModeLLM ||
		metadata.Generation.Provider != llm.ProviderOpenAICompatible ||
		metadata.Generation.Model != cfg.LLMModel {
		t.Fatalf("Generation = %#v, want safe LLM provider metadata", metadata.Generation)
	}
	if metadata.RawIdea != cfg.Idea {
		t.Fatalf("RawIdea = %q, want %q", metadata.RawIdea, cfg.Idea)
	}
}

func TestRunLLMOmitsPromptArtifactsByDefault(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{
		"# Generated Idea",
		"# Generated Specification",
		"# Generated Tasks",
		"# Generated Review Checklist",
	}, -1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea: "Build a personal book library",
		Out:  outDir,
		Name: "book-library",
		LLM:  true,
	}

	if err := Run(cfg); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := len(capturedPrompts()); got != 4 {
		t.Fatalf("provider calls = %d, want 4", got)
	}
	if _, err := os.Stat(filepath.Join(outDir, cfg.Name, "prompts")); !os.IsNotExist(err) {
		t.Fatalf("prompts directory stat error = %v, want not exist", err)
	}
}

func TestRunLLMModeFailsClearlyForInvalidConfigWithoutWriting(t *testing.T) {
	clearLLMEnv(t)

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea: "Build a personal book library",
		Out:  outDir,
		Name: "book-library",
		LLM:  true,
	}

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run returned nil error for invalid LLM config")
	}
	for _, want := range []string{"load LLM config", llm.EnvProvider, llm.EnvBaseURL, llm.EnvModel, llm.EnvAPIKey} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}

	projectDir := filepath.Join(outDir, cfg.Name)
	if _, statErr := os.Stat(projectDir); !os.IsNotExist(statErr) {
		t.Fatalf("project directory stat error = %v, want not exist", statErr)
	}
}

func TestRunLLMProviderFailureNamesStageAndDoesNotWritePartialArtifacts(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{
		"# Generated Idea\n\nfirst stage completed",
		"unused",
	}, 1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea: "Build a personal book library",
		Out:  outDir,
		Name: "book-library",
		LLM:  true,
	}

	err := Run(cfg)
	if err == nil {
		t.Fatal("Run returned nil error for provider failure")
	}
	for _, want := range []string{"spec.md", "503 Service Unavailable"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
	if got := len(capturedPrompts()); got != 2 {
		t.Fatalf("provider calls = %d, want 2", got)
	}

	projectDir := filepath.Join(outDir, cfg.Name)
	if _, statErr := os.Stat(projectDir); !os.IsNotExist(statErr) {
		t.Fatalf("project directory stat error = %v, want not exist", statErr)
	}
}

func TestRunLLMChecksOutputBeforeProviderCalls(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{
		"# Generated Idea",
		"# Generated Specification",
		"# Generated Tasks",
		"# Generated Review Checklist",
	}, -1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	outDir := t.TempDir()
	projectDir := filepath.Join(outDir, "book-library")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("create project directory: %v", err)
	}
	existingPath := filepath.Join(projectDir, "spec.md")
	if err := os.WriteFile(existingPath, []byte("keep existing specification"), 0644); err != nil {
		t.Fatalf("write existing artifact: %v", err)
	}

	err := Run(cli.RunConfig{
		Idea: "Build a personal book library",
		Out:  outDir,
		Name: "book-library",
		LLM:  true,
	})
	if err == nil {
		t.Fatal("Run returned nil error for existing output")
	}
	for _, want := range []string{"check artifact output", "spec.md", "--force"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
	if got := len(capturedPrompts()); got != 0 {
		t.Fatalf("provider calls = %d, want 0 when output preflight fails", got)
	}
	assertWorkflowFileContent(t, existingPath, "keep existing specification")
	if _, err := os.Stat(filepath.Join(projectDir, "idea.md")); !os.IsNotExist(err) {
		t.Fatalf("idea.md stat error = %v, want not exist", err)
	}
}

func TestRunLLMForceChecksInvalidOutputBeforeProviderCalls(t *testing.T) {
	tests := []struct {
		name           string
		includePrompts bool
		setup          func(t *testing.T, projectDir string)
		wantPath       string
	}{
		{
			name: "project directory is a file",
			setup: func(t *testing.T, projectDir string) {
				t.Helper()
				if err := os.WriteFile(projectDir, []byte("not a directory"), 0644); err != nil {
					t.Fatalf("write blocking project path: %v", err)
				}
			},
			wantPath: "book-library",
		},
		{
			name:           "prompt parent is a file",
			includePrompts: true,
			setup: func(t *testing.T, projectDir string) {
				t.Helper()
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("create project directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(projectDir, "prompts"), []byte("not a directory"), 0644); err != nil {
					t.Fatalf("write blocking prompt path: %v", err)
				}
			},
			wantPath: "prompts",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capturedPrompts := newWorkflowLLMServer(t, []string{
				"# Generated Idea",
				"# Generated Specification",
				"# Generated Tasks",
				"# Generated Review Checklist",
			}, -1)
			setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

			outDir := t.TempDir()
			projectDir := filepath.Join(outDir, "book-library")
			test.setup(t, projectDir)

			err := Run(cli.RunConfig{
				Idea:           "Build a personal book library",
				Out:            outDir,
				Name:           "book-library",
				Force:          true,
				IncludePrompts: test.includePrompts,
				LLM:            true,
			})
			if err == nil {
				t.Fatal("Run returned nil error for invalid forced output")
			}
			for _, want := range []string{"check artifact output", test.wantPath} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %q, want message containing %q", err.Error(), want)
				}
			}
			if got := len(capturedPrompts()); got != 0 {
				t.Fatalf("provider calls = %d, want 0 when forced output preflight fails", got)
			}
		})
	}
}

func TestRunLLMChecksDirectoryWritabilityBeforeProviderCalls(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{
		"# Generated Idea",
		"# Generated Specification",
		"# Generated Tasks",
		"# Generated Review Checklist",
	}, -1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	outDir := t.TempDir()
	projectDir := filepath.Join(outDir, "book-library")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("create project directory: %v", err)
	}
	makeWorkflowDirectoryUnwritable(t, projectDir)

	err := Run(cli.RunConfig{
		Idea:  "Build a personal book library",
		Out:   outDir,
		Name:  "book-library",
		Force: true,
		LLM:   true,
	})
	if err == nil {
		t.Fatal("Run returned nil error for unwritable project directory")
	}
	for _, want := range []string{"check artifact output", "check output parent", "book-library"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
	if got := len(capturedPrompts()); got != 0 {
		t.Fatalf("provider calls = %d, want 0 when directory writability preflight fails", got)
	}
	entries, readErr := os.ReadDir(projectDir)
	if readErr != nil {
		t.Fatalf("read project directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("project directory contains %d entries, want no artifacts or probe residue", len(entries))
	}
}

func TestRunLLMRejectsSymlinkedOutputBeforeProviderCalls(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{
		"# Generated Idea",
		"# Generated Specification",
		"# Generated Tasks",
		"# Generated Review Checklist",
	}, -1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	outDir := t.TempDir()
	projectDir := filepath.Join(outDir, "book-library")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("create project directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside-idea.md")
	if err := os.Symlink(outsidePath, filepath.Join(projectDir, "idea.md")); err != nil {
		t.Skipf("symlinks are not available in this test environment: %v", err)
	}

	err := Run(cli.RunConfig{
		Idea:  "Build a personal book library",
		Out:   outDir,
		Name:  "book-library",
		Force: true,
		LLM:   true,
	})
	if err == nil {
		t.Fatal("Run returned nil error for symlinked output")
	}
	for _, want := range []string{"check artifact output", "symbolic links are not allowed", "idea.md"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
	if got := len(capturedPrompts()); got != 0 {
		t.Fatalf("provider calls = %d, want 0 when output symlink preflight fails", got)
	}
	if _, statErr := os.Stat(outsidePath); !os.IsNotExist(statErr) {
		t.Fatalf("outside target stat error = %v, want not exist", statErr)
	}
}

func TestRunLLMRejectsSymlinkedOutputAncestorBeforeProviderCalls(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{
		"# Generated Idea",
		"# Generated Specification",
		"# Generated Tasks",
		"# Generated Review Checklist",
	}, -1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outsideDir, "existing"), 0755); err != nil {
		t.Fatalf("create existing outside descendant: %v", err)
	}
	linkPath := filepath.Join(baseDir, "linked-parent")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Skipf("symlinks are not available in this test environment: %v", err)
	}
	outDir := filepath.Join(linkPath, "existing", "artifacts")

	err := Run(cli.RunConfig{
		Idea:  "Build a personal book library",
		Out:   outDir,
		Name:  "book-library",
		Force: true,
		LLM:   true,
	})
	if err == nil {
		t.Fatal("Run returned nil error for symlinked output ancestor")
	}
	for _, want := range []string{"check artifact output", "symbolic links are not allowed", "linked-parent"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want message containing %q", err.Error(), want)
		}
	}
	if got := len(capturedPrompts()); got != 0 {
		t.Fatalf("provider calls = %d, want 0 when output ancestor preflight fails", got)
	}
	if _, statErr := os.Stat(filepath.Join(outsideDir, "existing", "artifacts")); !os.IsNotExist(statErr) {
		t.Fatalf("outside output directory stat error = %v, want not exist", statErr)
	}
}

func TestRunDryRunListsPromptArtifactsWhenIncluded(t *testing.T) {
	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea:           "Build a personal book library",
		Out:            outDir,
		Name:           "book-library",
		DryRun:         true,
		IncludePrompts: true,
	}

	output, err := captureStdout(func() error {
		return Run(cfg)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, want := range []string{
		"- idea.md",
		"- spec.md",
		"- tasks.md",
		"- review-checklist.md",
		"- prompts/01-normalize-idea.prompt.md",
		"- prompts/02-generate-spec.prompt.md",
		"- prompts/03-generate-tasks.prompt.md",
		"- prompts/04-review-checklist.prompt.md",
		"- metadata.json",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("dry-run output does not contain %q\noutput:\n%s", want, output)
		}
	}

	if _, err := os.Stat(filepath.Join(outDir, cfg.Name)); !os.IsNotExist(err) {
		t.Fatalf("project directory stat error = %v, want not exist", err)
	}
}

func TestRunLLMDryRunValidatesConfigWithoutCallingProvider(t *testing.T) {
	server, capturedPrompts := newWorkflowLLMServer(t, []string{"unexpected provider call"}, -1)
	setValidWorkflowLLMEnv(t, server.URL+"/v1", "sk-test-secret")

	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea:   "Build a personal book library",
		Out:    outDir,
		Name:   "book-library",
		DryRun: true,
		LLM:    true,
	}

	output, err := captureStdout(func() error {
		return Run(cfg)
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := len(capturedPrompts()); got != 0 {
		t.Fatalf("provider calls = %d, want 0 during dry run", got)
	}
	for _, filename := range []string{"idea.md", "spec.md", "tasks.md", "review-checklist.md", "metadata.json"} {
		if !strings.Contains(output, "- "+filename) {
			t.Errorf("dry-run output does not contain %q\noutput:\n%s", filename, output)
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, cfg.Name)); !os.IsNotExist(err) {
		t.Fatalf("project directory stat error = %v, want not exist", err)
	}
}

func TestRunLLMDryRunRejectsInvalidProviderConfiguration(t *testing.T) {
	clearLLMEnv(t)

	err := Run(cli.RunConfig{
		Idea:   "Build a personal book library",
		Out:    t.TempDir(),
		Name:   "book-library",
		DryRun: true,
		LLM:    true,
	})
	if err == nil {
		t.Fatal("Run returned nil error for invalid LLM config during dry run")
	}
	if !strings.Contains(err.Error(), llm.EnvAPIKey) {
		t.Fatalf("error = %q, want missing provider configuration", err.Error())
	}
}

func newWorkflowLLMServer(
	t *testing.T,
	responses []string,
	failAt int,
) (*httptest.Server, func() []string) {
	t.Helper()

	var mu sync.Mutex
	var capturedPrompts []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if len(request.Messages) != 1 || request.Messages[0].Role != "user" {
			http.Error(w, "unexpected messages", http.StatusBadRequest)
			return
		}

		mu.Lock()
		callIndex := len(capturedPrompts)
		capturedPrompts = append(capturedPrompts, request.Messages[0].Content)
		mu.Unlock()

		if callIndex == failAt {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"provider temporarily unavailable"}}`))
			return
		}
		if callIndex >= len(responses) {
			http.Error(w, fmt.Sprintf("unexpected call %d", callIndex+1), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": request.Model,
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": responses[callIndex],
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	snapshot := func() []string {
		mu.Lock()
		defer mu.Unlock()

		return append([]string(nil), capturedPrompts...)
	}

	return server, snapshot
}

func setValidWorkflowLLMEnv(t *testing.T, baseURL string, apiKey string) {
	t.Helper()

	t.Setenv(llm.EnvProvider, llm.ProviderOpenAICompatible)
	t.Setenv(llm.EnvBaseURL, baseURL)
	t.Setenv(llm.EnvModel, "workflow-test-model")
	t.Setenv(llm.EnvAPIKey, apiKey)
}

func setInvalidLLMEnv(t *testing.T) {
	t.Helper()

	t.Setenv(llm.EnvProvider, "unsupported-provider")
	t.Setenv(llm.EnvBaseURL, "://invalid")
	t.Setenv(llm.EnvModel, "")
	t.Setenv(llm.EnvAPIKey, "")
}

func clearLLMEnv(t *testing.T) {
	t.Helper()

	t.Setenv(llm.EnvProvider, "")
	t.Setenv(llm.EnvBaseURL, "")
	t.Setenv(llm.EnvModel, "")
	t.Setenv(llm.EnvAPIKey, "")
}

func assertWorkflowFileContent(t *testing.T, path string, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, string(data), want)
	}
}

func makeWorkflowDirectoryUnwritable(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat directory before chmod: %v", err)
	}
	if err := os.Chmod(path, 0555); err != nil {
		t.Fatalf("make directory unwritable: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(path, info.Mode().Perm())
	})

	probe, err := os.CreateTemp(path, ".permission-control-*")
	if err != nil {
		return
	}
	probePath := probe.Name()
	if closeErr := probe.Close(); closeErr != nil {
		t.Fatalf("close permission control file: %v", closeErr)
	}
	if removeErr := os.Remove(probePath); removeErr != nil {
		t.Fatalf("remove permission control file: %v", removeErr)
	}
	t.Skip("filesystem or effective user does not enforce directory write mode bits")
}

func captureStdout(fn func() error) (string, error) {
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = writer

	runErr := fn()

	if err := writer.Close(); err != nil && runErr == nil {
		runErr = err
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil && runErr == nil {
		runErr = err
	}
	if err := reader.Close(); err != nil && runErr == nil {
		runErr = err
	}

	return buf.String(), runErr
}
