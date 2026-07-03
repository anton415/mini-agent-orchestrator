package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anton415/mini-agent-orchestrator/internal/cli"
)

func TestRunWritesPromptArtifacts(t *testing.T) {
	outDir := t.TempDir()
	cfg := cli.RunConfig{
		Idea:      "Build a personal book library",
		Out:       outDir,
		Name:      "book-library",
		CreatedAt: time.Date(2026, 6, 25, 11, 23, 10, 0, time.UTC),
		Prompts:   true,
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
