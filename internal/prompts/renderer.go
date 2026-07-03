package prompts

import (
	"bytes"
	"embed"
	"text/template"

	"github.com/anton415/mini-agent-orchestrator/internal/model"
	"github.com/anton415/mini-agent-orchestrator/internal/templates"
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
func RenderAll(project model.Project) ([]templates.Artifact, error) {
	artifacts := make([]templates.Artifact, 0, len(templateDefinitions))

	for _, def := range templateDefinitions {
		tmpl, err := template.ParseFS(promptFS, def.templatePath)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, project); err != nil {
			return nil, err
		}

		artifacts = append(artifacts, templates.Artifact{
			Filename: def.outputName,
			Content:  buf.String(),
		})
	}

	return artifacts, nil
}
