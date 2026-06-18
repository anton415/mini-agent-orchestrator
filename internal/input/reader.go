package input

import (
	"os"
	"strings"
)

// ReadIdea returns the trimmed idea text from either an inline --idea value or
// the markdown file pointed to by --input.
func ReadIdea(rawIdea string, inputPath string) (string, error) {
	// Prefer inline text when it is present; ParseRunArgs prevents callers of the
	// CLI from providing both sources at once.
	if rawIdea != "" {
		return strings.TrimSpace(rawIdea), nil
	}

	// File input lets longer ideas live in markdown while the rest of the
	// pipeline receives the same plain string shape as inline input.
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}
