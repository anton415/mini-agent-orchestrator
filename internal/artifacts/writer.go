package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

// Artifact is a rendered file ready to be written to disk.
type Artifact struct {
	Filename string
	Content  string
}

// WriteAll writes rendered project artifacts and project metadata into a project directory.
func WriteAll(outDir string, project model.Project, items []Artifact, force bool) error {
	// Create the project directory if it doesn't exist.
	projectDir := filepath.Join(outDir, project.Name)

	// 0755 means: owner can read/write/execute, group can read/execute, others can read/execute.
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// Write each artifact to the project directory.
	for _, item := range items {
		path := filepath.Join(projectDir, item.Filename)
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
