package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProjectOmitsGenerationMetadataWhenUnset(t *testing.T) {
	project := NewProjectAt(
		"demo",
		"build a small tool",
		time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC),
	)

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	const want = `{"Name":"demo","RawIdea":"build a small tool","CreatedAt":"2026-07-11T10:00:00Z","Version":"v0"}`
	if string(data) != want {
		t.Fatalf("project JSON = %s, want unchanged deterministic JSON %s", data, want)
	}
}

func TestProjectIncludesSafeLLMGenerationMetadata(t *testing.T) {
	project := NewProjectAt(
		"demo",
		"build a small tool",
		time.Date(2026, 7, 11, 10, 0, 0, 0, time.UTC),
	)
	project.Generation = &GenerationMetadata{
		Mode:     GenerationModeLLM,
		Provider: "openai-compatible",
		Model:    "test-model",
	}

	data, err := json.Marshal(project)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var decoded struct {
		Generation *GenerationMetadata
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if decoded.Generation == nil {
		t.Fatal("Generation = nil, want LLM generation metadata")
	}
	if *decoded.Generation != *project.Generation {
		t.Fatalf("Generation = %#v, want %#v", decoded.Generation, project.Generation)
	}
}
