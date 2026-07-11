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

// CheckWritable validates every output path, checks overwrite collisions, and
// probes the filesystem operations needed to create or replace the outputs.
// A successful check leaves no temporary probe entries behind.
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

	seen := make(map[string]struct{}, len(paths))
	existingPaths := make([]string, 0, len(paths))
	missingPaths := make([]string, 0, len(paths))

	for _, path := range paths {
		if err := rejectSymlinkedOutputPath(outDir, path); err != nil {
			return err
		}
		if _, exists := seen[path]; exists {
			return fmt.Errorf("duplicate output path: %s", path)
		}
		seen[path] = struct{}{}

		info, err := os.Stat(path)
		if err == nil {
			if !force {
				return fmt.Errorf("file already exists: %s; use --force to overwrite", path)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("invalid output path %s: existing target must be a regular file", path)
			}
			existingPaths = append(existingPaths, path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check output path %s: %w", path, err)
		} else {
			missingPaths = append(missingPaths, path)
		}
	}

	for _, path := range existingPaths {
		if err := checkExistingFileWritable(path); err != nil {
			return err
		}
	}

	type parentProbe struct {
		directory string
		exact     bool
	}
	probedParents := make(map[parentProbe]struct{}, len(missingPaths))
	for _, path := range missingPaths {
		parentDir, exact, err := nearestExistingDirectory(filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("check output parent for %s: %w", path, err)
		}

		probeKey := parentProbe{directory: parentDir, exact: exact}
		if _, ok := probedParents[probeKey]; ok {
			continue
		}
		if err := probeOutputCreation(parentDir, exact); err != nil {
			return fmt.Errorf("check output parent for %s: %w", path, err)
		}
		probedParents[probeKey] = struct{}{}
	}

	return nil
}

func rejectSymlinkedOutputPath(outputRoot string, path string) error {
	relative, err := filepath.Rel(outputRoot, path)
	if err != nil {
		return fmt.Errorf("check output path %s relative to output root: %w", path, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
		return fmt.Errorf("invalid output path %s: path escapes output root", path)
	}

	current := filepath.Clean(outputRoot)
	components := []string{current}
	if relative != "." {
		for _, component := range strings.Split(relative, string(os.PathSeparator)) {
			current = filepath.Join(current, component)
			components = append(components, current)
		}
	}

	for _, component := range components {
		info, lstatErr := os.Lstat(component)
		if lstatErr == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("invalid output path %s: symbolic links are not allowed", component)
			}
			continue
		}
		if os.IsNotExist(lstatErr) {
			return nil
		}
		return fmt.Errorf("check output path component %s: %w", component, lstatErr)
	}

	return nil
}

func checkExistingFileWritable(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("check existing output file %s is writable: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close writable output file %s: %w", path, err)
	}

	return nil
}

func nearestExistingDirectory(path string) (directory string, exact bool, err error) {
	wanted := filepath.Clean(path)
	current := wanted

	for {
		info, statErr := os.Stat(current)
		if statErr == nil {
			if !info.IsDir() {
				return "", false, fmt.Errorf("invalid output parent %s: existing path must be a directory", current)
			}
			return current, current == wanted, nil
		}
		if !os.IsNotExist(statErr) {
			return "", false, fmt.Errorf("check output parent %s: %w", current, statErr)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, fmt.Errorf("no existing directory found for %s", wanted)
		}
		current = parent
	}
}

func probeOutputCreation(parentDir string, parentExists bool) error {
	if parentExists {
		return probeFileCreation(parentDir)
	}

	probeDir, err := os.MkdirTemp(parentDir, ".mao-write-check-*")
	if err != nil {
		return fmt.Errorf("create temporary directory in %s: %w", parentDir, err)
	}

	if err := probeFileCreation(probeDir); err != nil {
		_ = os.RemoveAll(probeDir)
		return err
	}
	if err := os.Remove(probeDir); err != nil {
		return fmt.Errorf("remove temporary directory %s: %w", probeDir, err)
	}

	return nil
}

func probeFileCreation(parentDir string) error {
	probe, err := os.CreateTemp(parentDir, ".mao-write-check-*")
	if err != nil {
		return fmt.Errorf("create temporary file in %s: %w", parentDir, err)
	}
	probePath := probe.Name()

	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return fmt.Errorf("close temporary file %s: %w", probePath, err)
	}
	if err := os.Remove(probePath); err != nil {
		return fmt.Errorf("remove temporary file %s: %w", probePath, err)
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
