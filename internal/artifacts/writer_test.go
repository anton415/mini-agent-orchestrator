package artifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
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
		{
			name:       "prompt",
			setupPath:  filepath.Join("prompts", "01-normalize-idea.prompt.md"),
			wantPath:   filepath.Join("prompts", "01-normalize-idea.prompt.md"),
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
			if err := os.MkdirAll(filepath.Dir(existingPath), 0755); err != nil {
				t.Fatalf("create existing file parent directory: %v", err)
			}
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

func TestWriteAllPreflightPreventsPartialWritesForLateCollisions(t *testing.T) {
	tests := []struct {
		name         string
		existingPath string
	}{
		{
			name:         "late artifact",
			existingPath: "tasks.md",
		},
		{
			name:         "metadata",
			existingPath: "metadata.json",
		},
		{
			name:         "late prompt",
			existingPath: filepath.Join("prompts", "01-normalize-idea.prompt.md"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			project := testProject()
			projectDir := filepath.Join(outDir, project.Name)
			existingPath := filepath.Join(projectDir, test.existingPath)

			if err := os.MkdirAll(filepath.Dir(existingPath), 0755); err != nil {
				t.Fatalf("create existing file parent directory: %v", err)
			}
			if err := os.WriteFile(existingPath, []byte("keep me"), 0644); err != nil {
				t.Fatalf("write existing file: %v", err)
			}

			err := WriteAll(outDir, project, testArtifacts(), false)
			if err == nil {
				t.Fatal("WriteAll returned nil error")
			}
			if !strings.Contains(err.Error(), "file already exists") || !strings.Contains(err.Error(), test.existingPath) {
				t.Fatalf("error = %q, want collision for %q", err.Error(), test.existingPath)
			}

			assertFileContent(t, existingPath, "keep me")

			outputPaths := []string{"metadata.json"}
			for _, item := range testArtifacts() {
				outputPaths = append(outputPaths, item.Filename)
			}
			for _, outputPath := range outputPaths {
				if outputPath == test.existingPath {
					continue
				}
				path := filepath.Join(projectDir, outputPath)
				if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
					t.Fatalf("unexpected output %s stat error = %v, want not exist", outputPath, statErr)
				}
			}
		})
	}
}

func TestCheckWritableDoesNotCreateProjectDirectory(t *testing.T) {
	rootDir := t.TempDir()
	outDir := filepath.Join(rootDir, "nested", "artifacts")
	project := testProject()

	if err := CheckWritable(outDir, project, testArtifacts(), false); err != nil {
		t.Fatalf("CheckWritable returned error: %v", err)
	}

	projectDir := filepath.Join(outDir, project.Name)
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Fatalf("project directory stat error = %v, want not exist", err)
	}
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		t.Fatalf("read probe parent directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("probe parent contains %d entries, want no temporary residue", len(entries))
	}
}

func TestCheckWritableForceStillRejectsInvalidOutputPaths(t *testing.T) {
	t.Run("project directory is a file", func(t *testing.T) {
		outDir := t.TempDir()
		project := testProject()
		projectPath := filepath.Join(outDir, project.Name)
		if err := os.WriteFile(projectPath, []byte("not a directory"), 0644); err != nil {
			t.Fatalf("write blocking project path: %v", err)
		}

		err := CheckWritable(outDir, project, testArtifacts(), true)
		if err == nil {
			t.Fatal("CheckWritable returned nil error")
		}
		if !strings.Contains(err.Error(), "check output path") || !strings.Contains(err.Error(), project.Name) {
			t.Fatalf("error = %q, want invalid project directory context", err.Error())
		}
	})

	t.Run("prompt parent is a file", func(t *testing.T) {
		outDir := t.TempDir()
		project := testProject()
		projectDir := filepath.Join(outDir, project.Name)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			t.Fatalf("create project directory: %v", err)
		}
		promptsPath := filepath.Join(projectDir, "prompts")
		if err := os.WriteFile(promptsPath, []byte("not a directory"), 0644); err != nil {
			t.Fatalf("write blocking prompt path: %v", err)
		}

		err := CheckWritable(outDir, project, testArtifacts(), true)
		if err == nil {
			t.Fatal("CheckWritable returned nil error")
		}
		if !strings.Contains(err.Error(), "check output path") || !strings.Contains(err.Error(), "prompts") {
			t.Fatalf("error = %q, want invalid prompt parent context", err.Error())
		}
	})

	t.Run("artifact target is a directory", func(t *testing.T) {
		outDir := t.TempDir()
		project := testProject()
		artifactPath := filepath.Join(outDir, project.Name, "idea.md")
		if err := os.MkdirAll(artifactPath, 0755); err != nil {
			t.Fatalf("create blocking artifact directory: %v", err)
		}

		err := CheckWritable(outDir, project, testArtifacts(), true)
		if err == nil {
			t.Fatal("CheckWritable returned nil error")
		}
		if !strings.Contains(err.Error(), "existing target must be a regular file") {
			t.Fatalf("error = %q, want invalid target type", err.Error())
		}
	})
}

func TestCheckWritableRejectsUnwritableOutputParents(t *testing.T) {
	tests := []struct {
		name     string
		prepare  func(t *testing.T, outDir string, project model.Project) string
		wantPath string
	}{
		{
			name: "selected output directory",
			prepare: func(t *testing.T, outDir string, project model.Project) string {
				t.Helper()
				if err := os.MkdirAll(outDir, 0755); err != nil {
					t.Fatalf("create output directory: %v", err)
				}
				makeDirectoryUnwritable(t, outDir)
				return outDir
			},
			wantPath: "artifacts",
		},
		{
			name: "project directory",
			prepare: func(t *testing.T, outDir string, project model.Project) string {
				t.Helper()
				projectDir := filepath.Join(outDir, project.Name)
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("create project directory: %v", err)
				}
				makeDirectoryUnwritable(t, projectDir)
				return projectDir
			},
			wantPath: "demo",
		},
		{
			name: "prompt directory",
			prepare: func(t *testing.T, outDir string, project model.Project) string {
				t.Helper()
				promptsDir := filepath.Join(outDir, project.Name, "prompts")
				if err := os.MkdirAll(promptsDir, 0755); err != nil {
					t.Fatalf("create prompt directory: %v", err)
				}
				makeDirectoryUnwritable(t, promptsDir)
				return promptsDir
			},
			wantPath: "prompts",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outDir := filepath.Join(t.TempDir(), "artifacts")
			project := testProject()
			blockedDir := test.prepare(t, outDir, project)

			err := CheckWritable(outDir, project, testArtifacts(), true)
			if err == nil {
				t.Fatal("CheckWritable returned nil error")
			}
			for _, want := range []string{"check output parent", test.wantPath} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error = %q, want message containing %q", err.Error(), want)
				}
			}

			entries, readErr := os.ReadDir(blockedDir)
			if readErr != nil {
				t.Fatalf("read blocked directory: %v", readErr)
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".mao-write-check-") {
					t.Fatalf("temporary probe entry was not removed: %s", entry.Name())
				}
			}
		})
	}
}

func TestCheckWritableForceRejectsUnwritableExistingFile(t *testing.T) {
	outDir := t.TempDir()
	project := testProject()
	projectDir := filepath.Join(outDir, project.Name)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("create project directory: %v", err)
	}
	existingPath := filepath.Join(projectDir, "idea.md")
	if err := os.WriteFile(existingPath, []byte("keep me"), 0644); err != nil {
		t.Fatalf("write existing artifact: %v", err)
	}
	makeFileUnwritable(t, existingPath)

	err := CheckWritable(outDir, project, []Artifact{{Filename: "idea.md", Content: "replacement"}}, true)
	if err == nil {
		t.Fatal("CheckWritable returned nil error")
	}
	if !strings.Contains(err.Error(), "check existing output file") || !strings.Contains(err.Error(), "idea.md") {
		t.Fatalf("error = %q, want unwritable existing file context", err.Error())
	}
	assertFileContent(t, existingPath, "keep me")
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

	for _, name := range []string{"idea.md", "metadata.json", filepath.Join("prompts", "01-normalize-idea.prompt.md")} {
		path := filepath.Join(projectDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("create existing file parent directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("old content"), 0644); err != nil {
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

func TestWriteAllRejectsUnsafeArtifactPaths(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "empty",
			filename: "",
		},
		{
			name:     "absolute",
			filename: filepath.Join(string(os.PathSeparator), "tmp", "artifact.md"),
		},
		{
			name:     "parent prefix",
			filename: "../outside.md",
		},
		{
			name:     "parent nested",
			filename: "prompts/../outside.md",
		},
		{
			name:     "windows rooted",
			filename: `\tmp\artifact.md`,
		},
		{
			name:     "windows drive",
			filename: `C:\tmp\artifact.md`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outDir := t.TempDir()
			project := testProject()
			items := []Artifact{
				{Filename: test.filename, Content: "# Unsafe\n"},
			}

			err := WriteAll(outDir, project, items, false)
			if err == nil {
				t.Fatal("WriteAll returned nil error")
			}
			if !strings.Contains(err.Error(), "invalid artifact path") {
				t.Fatalf("error = %q, want invalid artifact path", err.Error())
			}

			projectDir := filepath.Join(outDir, project.Name)
			if _, err := os.Stat(filepath.Join(projectDir, "metadata.json")); !os.IsNotExist(err) {
				t.Fatalf("metadata.json stat error = %v, want not exist", err)
			}
		})
	}
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

func testArtifacts() []Artifact {
	// These mirror the artifact filenames produced by templates.RenderAll without
	// depending on template rendering in this writer-focused test package.
	return []Artifact{
		{Filename: "idea.md", Content: "# Idea\n"},
		{Filename: "spec.md", Content: "# Spec\n"},
		{Filename: "tasks.md", Content: "# Tasks\n"},
		{Filename: "review-checklist.md", Content: "# Review Checklist\n"},
		{Filename: filepath.Join("prompts", "01-normalize-idea.prompt.md"), Content: "# Prompt\n"},
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

func makeDirectoryUnwritable(t *testing.T, path string) {
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

func makeFileUnwritable(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file before chmod: %v", err)
	}
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatalf("make file unwritable: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(path, info.Mode().Perm())
	})

	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	if closeErr := file.Close(); closeErr != nil {
		t.Fatalf("close permission control file: %v", closeErr)
	}
	t.Skip("filesystem or effective user does not enforce file write mode bits")
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
