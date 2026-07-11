package cli

import (
	"flag" // used to parse command-line flags.
	"fmt"
	"time"
)

// RunConfig contains the validated options for the `mao run` command.
type RunConfig struct {
	Idea           string
	Input          string
	Out            string
	Name           string
	CreatedAt      time.Time
	Force          bool
	DryRun         bool
	IncludePrompts bool
	LLM            bool
	LLMProvider    string
	LLMBaseURL     string
	LLMModel       string
}

// ParseRunArgs parses flags that appear after `mao run`.
func ParseRunArgs(args []string) (RunConfig, error) {
	// 1. Create flag set.
	// We use a separate FlagSet so that we can parse only the arguments after `run`.
	runCommandFlags := flag.NewFlagSet("run", flag.ContinueOnError)

	// 2. Register/bind flags.
	// Register the supported flags and tell the parser where to store their values.
	var runCommandConfig RunConfig
	runCommandFlags.StringVar(&runCommandConfig.Idea, "idea", "", "short project idea")
	runCommandFlags.StringVar(&runCommandConfig.Input, "input", "", "path to markdown input file")
	runCommandFlags.StringVar(&runCommandConfig.Out, "out", "./artifacts", "output directory")
	runCommandFlags.StringVar(&runCommandConfig.Name, "name", "project", "project artifact folder name")
	var createdAt string
	runCommandFlags.StringVar(&createdAt, "created-at", "", "fixed metadata creation time in RFC3339 format")
	runCommandFlags.BoolVar(&runCommandConfig.Force, "force", false, "overwrite existing files")
	runCommandFlags.BoolVar(&runCommandConfig.DryRun, "dry-run", false, "show what would be created without writing files")
	runCommandFlags.BoolVar(&runCommandConfig.IncludePrompts, "include-prompts", false, "generate copyable manual LLM prompt files")
	runCommandFlags.BoolVar(&runCommandConfig.LLM, "llm", false, "generate artifacts with an LLM")
	runCommandFlags.StringVar(&runCommandConfig.LLMProvider, "llm-provider", "", "LLM provider override")
	runCommandFlags.StringVar(&runCommandConfig.LLMBaseURL, "llm-base-url", "", "LLM API base URL override")
	runCommandFlags.StringVar(&runCommandConfig.LLMModel, "llm-model", "", "LLM model override")

	// 3. Parse user args into the struct fields.
	// Parse the flags from the provided arguments. If there's an error, return it.
	if err := runCommandFlags.Parse(args); err != nil {
		return RunConfig{}, err
	}

	// Check for unexpected positional arguments. The `run` command doesn't support any, so if we find any, it's an error.
	if runCommandFlags.NArg() > 0 {
		return RunConfig{}, fmt.Errorf("unexpected positional argument: %s", runCommandFlags.Arg(0))
	}

	// 4. Validate parsed values.
	// The run command needs exactly one source of input: inline text or a file.
	if runCommandConfig.Idea == "" && runCommandConfig.Input == "" {
		return RunConfig{}, fmt.Errorf("either --idea or --input is required")
	}

	// If both --idea and --input are provided, it's also an error because we don't know which one to use.
	if runCommandConfig.Idea != "" && runCommandConfig.Input != "" {
		return RunConfig{}, fmt.Errorf("use either --idea or --input, not both")
	}

	if createdAt != "" {
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return RunConfig{}, fmt.Errorf("invalid --created-at: use RFC3339 format, for example 2026-06-25T11:23:10Z")
		}
		runCommandConfig.CreatedAt = parsedCreatedAt
	}

	// If we got here, the configuration is valid. Return it.
	return runCommandConfig, nil
}
