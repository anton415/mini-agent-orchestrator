package main

import (
	"fmt"
	"os"

	"github.com/anton415/mini-agent-orchestrator/internal/cli"
)

// Go CLI entrypoint for the Mini Agent Orchestrator (MAO).
func main() {
	// If the user runs mao without a command, it prints usage instructions and exits with status 1.
	if len(os.Args) < 2 {
		fmt.Println("usage: mao run --idea \"...\" --out ./artifacts/demo")
		os.Exit(1)
	}

	// Switch statement to handle different commands.
	switch os.Args[1] {
	// If the command is "run", we call the ParseRunArgs function to parse the flags that come after "run".
	case "run":
		// Pass only the arguments after `run` to the run-command parser.
		runCommandConfig, err := cli.ParseRunArgs(os.Args[2:])
		// If there's an error parsing the arguments, print the error message and exit with status 1.
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		// For demonstration purposes, we print the parsed configuration. In a real application, we would use this configuration to execute the command's logic.
		fmt.Printf("idea=%q input=%q out=%q name=%q force=%v dryRun=%v\n",
			runCommandConfig.Idea,
			runCommandConfig.Input,
			runCommandConfig.Out,
			runCommandConfig.Name,
			runCommandConfig.Force,
			runCommandConfig.DryRun)
	// Here we would call the function that executes the main logic of the `run` command, passing the validated configuration.
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
