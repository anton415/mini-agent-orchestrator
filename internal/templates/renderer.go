package templates

import (
	"bytes"
	"embed"
	"text/template"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

// templateFS contains the markdown templates bundled into the binary at build time.
//go:embed files/*.tmpl
var templateFS embed.FS

// Artifact is a rendered file ready to be written to disk.
type Artifact struct {
	Filename string
	Content  string
}

// templateDefinition pairs an output filename with its embedded template path.
type templateDefinition struct {
	outputName   string
	templatePath string
}

// templateDefinitions lists the artifacts to render in a fixed order. A slice is
// used instead of a map so RenderAll produces a deterministic order: Go map
// iteration order is intentionally randomized, which would make dry-run output
// and tests unstable. Output filenames are kept separate from template paths so
// templates can stay organized inside the package without changing the generated
// artifact names.
var templateDefinitions = []templateDefinition{
	{outputName: "idea.md", templatePath: "files/idea.md.tmpl"},
	{outputName: "spec.md", templatePath: "files/spec.md.tmpl"},
	{outputName: "tasks.md", templatePath: "files/tasks.md.tmpl"},
	{outputName: "review-checklist.md", templatePath: "files/review-checklist.md.tmpl"},
}

// RenderAll renders the full set of project artifacts from the embedded
// templates, returning them in the order declared by templateDefinitions.
func RenderAll(project model.Project) ([]Artifact, error) {
	artifacts := make([]Artifact, 0, len(templateDefinitions))

	for _, def := range templateDefinitions {
		// Parse and execute each template independently so the returned error points
		// at the specific template that failed.
		tmpl, err := template.ParseFS(templateFS, def.templatePath)
		if err != nil {
			return nil, err
		}

		// Execute the template with the project data and capture the output in a buffer.
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, project); err != nil {
			return nil, err
		}

		// Append the rendered content to the list of artifacts.
		artifacts = append(artifacts, Artifact{
			Filename: def.outputName,
			Content:  buf.String(),
		})
	}

	return artifacts, nil
}
