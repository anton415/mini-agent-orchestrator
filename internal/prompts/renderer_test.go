package prompts

import (
	"strings"
	"testing"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

func TestRenderAllOrder(t *testing.T) {
	want := []string{
		"prompts/01-normalize-idea.prompt.md",
		"prompts/02-generate-spec.prompt.md",
		"prompts/03-generate-tasks.prompt.md",
		"prompts/04-review-checklist.prompt.md",
	}

	project := model.NewProject("demo", "a small idea")

	for i := 0; i < 50; i++ {
		artifacts, err := RenderAll(project)
		if err != nil {
			t.Fatalf("RenderAll returned error: %v", err)
		}

		if len(artifacts) != len(want) {
			t.Fatalf("got %d artifacts, want %d", len(artifacts), len(want))
		}

		for j, name := range want {
			if artifacts[j].Filename != name {
				t.Fatalf("artifact[%d] = %q, want %q", j, artifacts[j].Filename, name)
			}
			if artifacts[j].Content == "" {
				t.Errorf("artifact %q has empty content", name)
			}
		}
	}
}

func TestRenderAllPromptsIncludeProjectDataAndRequiredSections(t *testing.T) {
	project := model.Project{
		Name:    "Prompt Renderer",
		RawIdea: "Generate copyable prompts for a manual LLM workflow.",
		Version: "v-test",
	}

	artifacts, err := RenderAll(project)
	if err != nil {
		t.Fatalf("RenderAll returned error: %v", err)
	}

	for _, artifact := range artifacts {
		for _, want := range []string{
			project.Name,
			project.RawIdea,
			"## Expected output",
			"## Constraints",
			"Do not add out-of-scope features.",
			"Do not use network access, API calls, or API keys.",
		} {
			if !strings.Contains(artifact.Content, want) {
				t.Errorf("%s does not contain %q\ncontent:\n%s", artifact.Filename, want, artifact.Content)
			}
		}
	}
}

func TestRenderAllPromptsAvoidGenericTODOs(t *testing.T) {
	project := model.NewProject("demo", "a small idea")

	artifacts, err := RenderAll(project)
	if err != nil {
		t.Fatalf("RenderAll returned error: %v", err)
	}

	for _, artifact := range artifacts {
		if strings.Contains(artifact.Content, "TODO") {
			t.Errorf("%s contains generic TODO marker\ncontent:\n%s", artifact.Filename, artifact.Content)
		}
	}
}
