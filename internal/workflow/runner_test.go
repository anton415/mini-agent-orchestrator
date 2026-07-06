package workflow

import (
	"bytes"
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
