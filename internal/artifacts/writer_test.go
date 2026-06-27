package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
	"github.com/anton415/mini-agent-orchestrator/internal/templates"
)

func TestWriteAllWritesArtifactsAndMetadata(t *testing.T) {
	// Use t.TempDir so the test writes into an isolated directory that Go cleans
	// up automatically. The nested path also proves WriteAll creates parent dirs.
	outDir := filepath.Join(t.TempDir(), "nested", "artifacts")
	project := testProject()
	items := testArtifacts()

	if err := WriteAll(outDir, project, items, false); err != nil {
		t.Fatalf("WriteAll returned error: %v", err)
	}

	// WriteAll should create one project directory inside the output directory.
	projectDir := filepath.Join(outDir, project.Name)
	info, err := os.Stat(projectDir)
	if err != nil {
		t.Fatalf("project directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("project path is not a directory: %s", projectDir)
	}

	// Every artifact passed to WriteAll should be written with its exact content.
	for _, item := range items {
		assertFileContent(t, filepath.Join(projectDir, item.Filename), item.Content)
	}

	// metadata.json is separate from the markdown artifacts. Decode it back into
	// a Project so the test verifies valid JSON and the important field values.
	metadataPath := filepath.Join(projectDir, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}

	var got model.Project
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("metadata.json is not valid JSON: %v", err)
	}
	assertProject(t, got, project)
}

func TestWriteAllRefusesOverwriteWithoutForce(t *testing.T) {
	// Check overwrite protection for both file categories WriteAll owns:
	// generated markdown artifacts and metadata.json.
	tests := []struct {
		name       string
		setupPath  string
		wantPath   string
		wantExists string
	}{
		{
			name:       "artifact",
			setupPath:  "idea.md",
			wantPath:   "idea.md",
			wantExists: "file already exists",
		},
		{
			name:       "metadata",
			setupPath:  "metadata.json",
			wantPath:   "metadata.json",
			wantExists: "file already exists",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			project := testProject()
			projectDir := filepath.Join(outDir, project.Name)

			// Pre-create the project directory and one existing file to simulate a
			// previous run of the generator.
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				t.Fatalf("create project directory: %v", err)
			}

			existingPath := filepath.Join(projectDir, test.setupPath)
			if err := os.WriteFile(existingPath, []byte("keep me"), 0644); err != nil {
				t.Fatalf("write existing file: %v", err)
			}

			// With force=false, WriteAll should fail instead of replacing the file.
			err := WriteAll(outDir, project, testArtifacts(), false)
			if err == nil {
				t.Fatal("WriteAll returned nil error")
			}
			if !strings.Contains(err.Error(), test.wantExists) || !strings.Contains(err.Error(), test.wantPath) {
				t.Fatalf("error = %q, want message containing %q and %q", err.Error(), test.wantExists, test.wantPath)
			}

			// The original contents must still be there after the failed write.
			assertFileContent(t, existingPath, "keep me")
		})
	}
}

func TestWriteAllOverwritesExistingFilesWithForce(t *testing.T) {
	outDir := t.TempDir()
	project := testProject()
	projectDir := filepath.Join(outDir, project.Name)

	// Start with files that already exist so the test proves force=true replaces
	// them instead of returning an overwrite error.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("create project directory: %v", err)
	}

	for _, name := range []string{"idea.md", "metadata.json"} {
		if err := os.WriteFile(filepath.Join(projectDir, name), []byte("old content"), 0644); err != nil {
			t.Fatalf("write existing %s: %v", name, err)
		}
	}

	items := testArtifacts()

	// force=true opts into overwriting existing generated files.
	if err := WriteAll(outDir, project, items, true); err != nil {
		t.Fatalf("WriteAll returned error: %v", err)
	}

	// Existing markdown files should now contain the newly generated content.
	for _, item := range items {
		assertFileContent(t, filepath.Join(projectDir, item.Filename), item.Content)
	}

	// Existing metadata.json should also be replaced with fresh project metadata.
	data, err := os.ReadFile(filepath.Join(projectDir, "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}

	var got model.Project
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("metadata.json is not valid JSON: %v", err)
	}
	assertProject(t, got, project)
}

func testProject() model.Project {
	// Use a fixed timestamp so metadata assertions are deterministic.
	return model.Project{
		Name:      "demo",
		RawIdea:   "build a tiny orchestrator",
		CreatedAt: time.Date(2026, 6, 27, 12, 30, 0, 0, time.UTC),
		Version:   "v0",
	}
}

func testArtifacts() []templates.Artifact {
	// These mirror the artifact filenames produced by templates.RenderAll without
	// depending on template rendering in this writer-focused test package.
	return []templates.Artifact{
		{Filename: "idea.md", Content: "# Idea\n"},
		{Filename: "spec.md", Content: "# Spec\n"},
		{Filename: "tasks.md", Content: "# Tasks\n"},
		{Filename: "review-checklist.md", Content: "# Review Checklist\n"},
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	// Marked as a helper so failures point at the test assertion, not this helper.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, string(data), want)
	}
}

func assertProject(t *testing.T, got model.Project, want model.Project) {
	t.Helper()

	// Compare fields individually so a failure explains which metadata value
	// changed, instead of dumping two full structs.
	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.RawIdea != want.RawIdea {
		t.Fatalf("RawIdea = %q, want %q", got.RawIdea, want.RawIdea)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if got.Version != want.Version {
		t.Fatalf("Version = %q, want %q", got.Version, want.Version)
	}
}
