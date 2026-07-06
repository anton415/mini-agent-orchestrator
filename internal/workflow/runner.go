package workflow

import (
	"fmt"

	"github.com/anton415/mini-agent-orchestrator/internal/artifacts"
	"github.com/anton415/mini-agent-orchestrator/internal/cli"
	"github.com/anton415/mini-agent-orchestrator/internal/input"
	"github.com/anton415/mini-agent-orchestrator/internal/model"
	"github.com/anton415/mini-agent-orchestrator/internal/prompts"
	"github.com/anton415/mini-agent-orchestrator/internal/templates"
)

// Run executes the workflow to generate artifacts based on the provided configuration.
func Run(cfg cli.RunConfig) error {
	// Read the idea text from the input source (file or stdin).
	ideaText, err := input.ReadIdea(cfg.Idea, cfg.Input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	// Create a new project model using the provided name and idea text.
	project := model.NewProject(cfg.Name, ideaText)
	if !cfg.CreatedAt.IsZero() {
		project = model.NewProjectAt(cfg.Name, ideaText, cfg.CreatedAt)
	}

	// Render all templates for the project.
	items, err := templates.RenderAll(project)
	if err != nil {
		return fmt.Errorf("render templates: %w", err)
	}

	if cfg.IncludePrompts {
		// Render companion prompt files for the manual LLM workflow.
		promptItems, err := prompts.RenderAll(project)
		if err != nil {
			return fmt.Errorf("render prompts: %w", err)
		}
		items = append(items, promptItems...)
	}

	// Handle dry run mode: if enabled, print the files that would be created without actually writing them to disk.
	if cfg.DryRun {
		fmt.Println("Dry run. Files that would be created:")
		for _, item := range items {
			fmt.Println("-", item.Filename)
		}
		fmt.Println("- metadata.json")
		return nil
	}

	// Write all generated artifacts to disk.
	if err := artifacts.WriteAll(cfg.Out, project, items, cfg.Force); err != nil {
		return fmt.Errorf("write artifacts: %w", err)
	}

	fmt.Printf("Artifacts created: %s/%s\n", cfg.Out, cfg.Name)
	return nil
}
