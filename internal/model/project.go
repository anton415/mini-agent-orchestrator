package model

import "time"

// Project represents a project with its name, raw idea, creation time, and version.
type Project struct {
	Name      string
	RawIdea   string
	CreatedAt time.Time
	Version   string
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
