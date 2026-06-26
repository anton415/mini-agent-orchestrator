package templates

import (
	"testing"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

// TestRenderAllOrder verifies that RenderAll returns artifacts with the expected
// filenames in a stable, deterministic order across repeated invocations.
func TestRenderAllOrder(t *testing.T) {
	want := []string{
		"idea.md",
		"spec.md",
		"tasks.md",
		"review-checklist.md",
	}

	project := model.NewProject("demo", "a small idea")

	// Render multiple times; the order must be identical every time, which would
	// not hold if RenderAll iterated over a map.
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
