package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

// Artifact is a rendered file ready to be written to disk.
type Artifact struct {
	Filename string
	Content  string
}

// WriteAll writes rendered project artifacts and project metadata into a project directory.
func WriteAll(outDir string, project model.Project, items []Artifact, force bool) error {
	if err := CheckWritable(outDir, project, items, force); err != nil {
		return err
	}

	// Create the project directory if it doesn't exist.
	projectDir := filepath.Join(outDir, project.Name)

	// 0755 means: owner can read/write/execute, group can read/execute, others can read/execute.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// Write each artifact to the project directory.
	for _, item := range items {
		path, err := artifactPath(projectDir, item.Filename)
		if err != nil {
			return err
		}

		parentDir := filepath.Dir(path)

		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return err
		}

		if !force {
			// Refuse to overwrite generated artifacts unless the caller explicitly opts in.
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("file already exists: %s; use --force to overwrite", path)
			}
		}

		// 0644 means: owner can read/write, group can read, others can read.
		if err := os.WriteFile(path, []byte(item.Content), 0644); err != nil {
			return err
		}
	}

	// Write the project metadata to a JSON file in the project directory.
	metadataPath := filepath.Join(projectDir, "metadata.json")

	if !force {
		// Apply the same overwrite protection to metadata as the markdown artifacts.
		if _, err := os.Stat(metadataPath); err == nil {
			return fmt.Errorf("file already exists: %s; use --force to overwrite", metadataPath)
		}
	}

	// Marshal the project metadata to JSON with indentation for readability.
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	// Write the metadata JSON file with a final newline like the markdown artifacts.
	return os.WriteFile(metadataPath, append(data, '\n'), 0644)
}

// CheckWritable validates every output path and checks overwrite collisions
// without creating directories or writing files.
func CheckWritable(outDir string, project model.Project, items []Artifact, force bool) error {
	projectDir := filepath.Join(outDir, project.Name)
	paths := make([]string, 0, len(items)+1)

	for _, item := range items {
		path, err := artifactPath(projectDir, item.Filename)
		if err != nil {
			return err
		}
		paths = append(paths, path)
	}

	metadataPath, err := artifactPath(projectDir, "metadata.json")
	if err != nil {
		return err
	}
	paths = append(paths, metadataPath)

	if force {
		return nil
	}

	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if _, exists := seen[path]; exists {
			return fmt.Errorf("duplicate output path: %s", path)
		}
		seen[path] = struct{}{}

		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file already exists: %s; use --force to overwrite", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check output path %s: %w", path, err)
		}
	}

	return nil
}

func artifactPath(projectDir string, filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("invalid artifact path %q: path must be relative", filename)
	}
	if isAbsoluteArtifactPath(filename) {
		return "", fmt.Errorf("invalid artifact path %q: absolute paths are not allowed", filename)
	}
	if hasParentDirSegment(filename) {
		return "", fmt.Errorf("invalid artifact path %q: parent directory references are not allowed", filename)
	}

	cleaned := filepath.Clean(filename)
	if cleaned == "." {
		return "", fmt.Errorf("invalid artifact path %q: path must name a file", filename)
	}

	path := filepath.Join(projectDir, cleaned)
	rel, err := filepath.Rel(projectDir, path)
	if err != nil {
		return "", fmt.Errorf("invalid artifact path %q: %w", filename, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid artifact path %q: path escapes project directory", filename)
	}

	return path, nil
}

func isAbsoluteArtifactPath(filename string) bool {
	if filepath.IsAbs(filename) || strings.HasPrefix(filename, `\`) {
		return true
	}

	return len(filename) >= 2 &&
		isASCIILetter(filename[0]) &&
		filename[1] == ':'
}

func hasParentDirSegment(filename string) bool {
	parts := strings.FieldsFunc(filename, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}

	return false
}

func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
