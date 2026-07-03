package main

import (
	"fmt"
	"os"

	"github.com/anton415/mini-agent-orchestrator/internal/cli"
	"github.com/anton415/mini-agent-orchestrator/internal/workflow"
)

// main is the entry point of the mini-agent-orchestrator application.
func main() {
	// Check if at least one command-line argument is provided (the command to run).
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: mao run --idea \"...\" --out ./artifacts --name demo [--prompts]")
		os.Exit(1)
	}

	// Switch on the first command-line argument to determine which command to execute.
	switch os.Args[1] {
	case "run":
		cfg, err := cli.ParseRunArgs(os.Args[2:])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

		if err := workflow.Run(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
