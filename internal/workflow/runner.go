package workflow

import (
	"context"
	"fmt"

	"github.com/anton415/mini-agent-orchestrator/internal/artifacts"
	"github.com/anton415/mini-agent-orchestrator/internal/cli"
	"github.com/anton415/mini-agent-orchestrator/internal/input"
	"github.com/anton415/mini-agent-orchestrator/internal/llm"
	openaicompatible "github.com/anton415/mini-agent-orchestrator/internal/llm/openai-compatible"
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

	// Render the deterministic artifacts first. Besides being the default output,
	// their stable filenames describe the LLM output during a side-effect-free dry run.
	items, err := templates.RenderAll(project)
	if err != nil {
		return fmt.Errorf("render templates: %w", err)
	}

	var promptItems []artifacts.Artifact
	if cfg.IncludePrompts || (cfg.LLM && !cfg.DryRun) {
		promptItems, err = prompts.RenderAll(project)
		if err != nil {
			return fmt.Errorf("render prompts: %w", err)
		}
	}

	if cfg.LLM {
		providerConfig, err := llm.LoadConfigFromEnvWithOverrides(true, llm.ConfigOverrides{
			Provider: cfg.LLMProvider,
			BaseURL:  cfg.LLMBaseURL,
			Model:    cfg.LLMModel,
		})
		if err != nil {
			return fmt.Errorf("load LLM config: %w", err)
		}

		if !cfg.DryRun {
			plannedItems := append([]artifacts.Artifact(nil), items...)
			if cfg.IncludePrompts {
				plannedItems = append(plannedItems, promptItems...)
			}
			if err := artifacts.CheckWritable(cfg.Out, project, plannedItems, cfg.Force); err != nil {
				return fmt.Errorf("check artifact output: %w", err)
			}

			provider, err := newLLMProvider(providerConfig)
			if err != nil {
				return fmt.Errorf("create LLM provider: %w", err)
			}

			generatedItems, executedPrompts, err := executeLLMWorkflow(context.Background(), promptItems, LLMGenerator{
				Provider: provider,
				Model:    providerConfig.Model,
			})
			if err != nil {
				return err
			}

			items = generatedItems
			promptItems = executedPrompts
			project.Generation = &model.GenerationMetadata{
				Mode:     model.GenerationModeLLM,
				Provider: providerConfig.Provider,
				Model:    providerConfig.Model,
			}
		}
	}

	if cfg.IncludePrompts {
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

func newLLMProvider(cfg llm.Config) (llm.Provider, error) {
	switch cfg.Provider {
	case llm.ProviderOpenAICompatible:
		return openaicompatible.NewClient(cfg)
	default:
		return nil, fmt.Errorf("unsupported LLM provider %q", cfg.Provider)
	}
}
