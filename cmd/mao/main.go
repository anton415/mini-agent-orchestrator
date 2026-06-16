package main

import (
	"fmt"	// used to print messages.
	"os"	// used to read command-line arguments and exit with an error code.
)

// Go CLI entrypoint for the Mini Agent Orchestrator (MAO).
func main() {
	// If the user runs mao without a command, it prints usage instructions and exits with status 1.
	if len(os.Args) < 2 {
		fmt.Println("usage: mao run --idea \"...\" --out ./artifacts/demo")
		os.Exit(1)
	}

	// The program only accepts one command right now: run. 
	// Anything else, like mao test, prints unknown command: test.
	if os.Args[1] != "run" {
		fmt.Printf("unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	// If the command is valid, it prints: "Mini Agent Orchestrator v0".
	fmt.Println("Mini Agent Orchestrator v0")
}