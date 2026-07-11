package model

import "time"

// GenerationModeLLM identifies artifacts produced through LLM execution.
const GenerationModeLLM = "llm"

// GenerationMetadata identifies how project artifacts were generated without
// including provider credentials or other secret configuration.
type GenerationMetadata struct {
	Mode     string
	Provider string
	Model    string
}

// Project represents a project with its name, raw idea, creation time, and version.
type Project struct {
	Name       string
	RawIdea    string
	CreatedAt  time.Time
	Version    string
	Generation *GenerationMetadata `json:",omitempty"`
}

// NewProject creates a new Project instance with the given name and raw idea, setting the creation time to now and version to "v0".
func NewProject(name string, rawIdea string) Project {
	return NewProjectAt(name, rawIdea, time.Now())
}

// NewProjectAt creates a Project with an explicit creation time.
func NewProjectAt(name string, rawIdea string, createdAt time.Time) Project {
	return Project{
		Name:      name,
		RawIdea:   rawIdea,
		CreatedAt: createdAt,
		Version:   "v0",
	}
}
