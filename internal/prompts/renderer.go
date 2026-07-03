package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"text/template"

	"github.com/anton415/mini-agent-orchestrator/internal/artifacts"
	"github.com/anton415/mini-agent-orchestrator/internal/model"
)

// promptFS contains manual LLM workflow prompt templates bundled into the binary.
//
//go:embed files/*.tmpl
var promptFS embed.FS

// templateDefinition pairs an output filename with its embedded template path.
type templateDefinition struct {
	outputName   string
	templatePath string
}

// templateDefinitions lists prompt artifacts in the fixed workflow order.
var templateDefinitions = []templateDefinition{
	{outputName: "prompts/01-normalize-idea.prompt.md", templatePath: "files/01-normalize-idea.prompt.md.tmpl"},
	{outputName: "prompts/02-generate-spec.prompt.md", templatePath: "files/02-generate-spec.prompt.md.tmpl"},
	{outputName: "prompts/03-generate-tasks.prompt.md", templatePath: "files/03-generate-tasks.prompt.md.tmpl"},
	{outputName: "prompts/04-review-checklist.prompt.md", templatePath: "files/04-review-checklist.prompt.md.tmpl"},
}

// RenderAll renders manual LLM workflow prompts from the embedded templates.
func RenderAll(project model.Project) ([]artifacts.Artifact, error) {
	return renderAll(promptFS, templateDefinitions, project)
}

func renderAll(fsys fs.FS, definitions []templateDefinition, project model.Project) ([]artifacts.Artifact, error) {
	rendered := make([]artifacts.Artifact, 0, len(definitions))

	for _, def := range definitions {
		tmpl, err := template.ParseFS(fsys, def.templatePath)
		if err != nil {
			return nil, fmt.Errorf("parse prompt template %q for output %q: %w", def.templatePath, def.outputName, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, project); err != nil {
			return nil, fmt.Errorf("execute prompt template %q for output %q: %w", def.templatePath, def.outputName, err)
		}

		rendered = append(rendered, artifacts.Artifact{
			Filename: def.outputName,
			Content:  buf.String(),
		})
	}

	return rendered, nil
}
