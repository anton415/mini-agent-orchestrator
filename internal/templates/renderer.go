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

// RenderAll renders the full set of project artifacts from the embedded templates.
func RenderAll(project model.Project) ([]Artifact, error) {
	// Keep output filenames separate from template paths so templates can stay
	// organized inside the package without changing the generated artifact names.
	files := map[string]string{
		"idea.md":             "files/idea.md.tmpl",
		"spec.md":             "files/spec.md.tmpl",
		"tasks.md":            "files/tasks.md.tmpl",
		"review-checklist.md": "files/review-checklist.md.tmpl",
	}

	var artifacts []Artifact

	for outputName, templatePath := range files {
		// Parse and execute each template independently so the returned error points
		// at the specific template that failed.
		tmpl, err := template.ParseFS(templateFS, templatePath)
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
			Filename: outputName,
			Content:  buf.String(),
		})
	}

	return artifacts, nil
}
