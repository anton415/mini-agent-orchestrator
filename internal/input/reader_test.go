package input

import (
	"os"
	"testing"
)

// TestReadIdeaFromString tests the ReadIdea function when provided with a direct string input.
func TestReadIdeaFromString(t *testing.T) {
	got, err := ReadIdea("  build cli  ", "")
	if err != nil {
		t.Fatal(err)
	}

	if got != "build cli" {
		t.Fatalf("expected %q, got %q", "build cli", got)
	}
}

// TestReadIdeaFromFile tests the ReadIdea function when provided with a file input.
func TestReadIdeaFromFile(t *testing.T) {
	file, err := os.CreateTemp("", "idea-*.md")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	if _, err := file.WriteString("  build from file  "); err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := ReadIdea("", file.Name())
	if err != nil {
		t.Fatal(err)
	}

	if got != "build from file" {
		t.Fatalf("expected %q, got %q", "build from file", got)
	}
}