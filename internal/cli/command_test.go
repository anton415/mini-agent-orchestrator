package cli

import (
	"strings"	// used to check error messages in tests.
	"testing"	// used to write unit tests for the ParseRunArgs function.
)

// TestParseRunArgsParsesRunCommandFlags tests that ParseRunArgs correctly parses valid flags for the `run` command and populates the RunConfig struct.
func TestParseRunArgsParsesRunCommandFlags(t *testing.T) {
	// We call ParseRunArgs with a set of valid flags that we expect users to provide when running the `mao run` command. 
	// We check that the returned RunConfig struct has the expected values.
	runCommandConfig, err := ParseRunArgs([] string {
		"--idea", "Build a personal book library",
		"--out", "./artifacts/demo",
		"--name", "demo",
		"--force",
		"--dry-run",
	})

	// If ParseRunArgs returns an error, the test fails because we provided valid input.
	if err != nil {
		t.Fatalf("ParseRunArgs returned an error: %v", err)
	}

	// We check that each field in the returned RunConfig struct matches the expected value based on the flags we provided.
	if runCommandConfig.Idea != "Build a personal book library" {
		t.Fatalf("Idea = %q, want %q", runCommandConfig.Idea, "Build a personal book library")
	}

	// Since we didn't provide an --input flag, we expect the Input field to be an empty string. If it's not, the test fails.
	if runCommandConfig.Input != "" {
		t.Fatalf("Input = %q, want empty string", runCommandConfig.Input)
	}

	// We check that the Out field is set to "./artifacts/demo" as specified by the --out flag. If it's not, the test fails.
	if runCommandConfig.Out != "./artifacts/demo" {
		t.Fatalf("Out = %q, want %q", runCommandConfig.Out, "./artifacts/demo")
	}

	// We check that the Name field is set to "demo" as specified by the --name flag. If it's not, the test fails.
	if runCommandConfig.Name != "demo" {
		t.Fatalf("Name = %q, want %q", runCommandConfig.Name, "demo")
	}

	// We check that the Force field is set to true because we provided the --force flag. If it's false, the test fails.
	if !runCommandConfig.Force {
		t.Fatal("Force = false, want true")
	}

	// We check that the DryRun field is set to true because we provided the --dry-run flag. If it's false, the test fails.
	if !runCommandConfig.DryRun {
		t.Fatal("DryRun = false, want true")
	}
}

// TestParseRunArgsRejectsInvalidInputSources tests that ParseRunArgs returns an error when the user provides invalid combinations of input flags for the `run` command, 
// such as missing both --idea and --input, providing both at the same time, or including unexpected positional arguments.
func TestParseRunArgsRejectsInvalidInputSources(t *testing.T) {
	// We define a set of test cases that represent different invalid input scenarios. 
	// Each test case includes a name, the arguments to pass to ParseRunArgs, and the expected error message that should be returned.
	tests := []struct {
		name             string
		args             []string
		wantErrorMessage string
	}{
		{
			name:             "missing idea and input",
			args:             nil,
			wantErrorMessage: "either --idea or --input is required",
		},
		{
			name:             "idea and input together",
			args:             []string{"--idea", "demo", "--input", "docs/mvp.md"},
			wantErrorMessage: "use either --idea or --input, not both",
		},
		{
			name:             "unexpected positional argument",
			args:             []string{"--idea", "demo", "extra"},
			wantErrorMessage: "unexpected positional argument: extra",
		},
	}

	// We iterate over each test case and run it as a subtest using t.Run. 
	// For each test, we call ParseRunArgs with the specified arguments and check that it returns an error. 
	// We then check that the error message contains the expected substring defined in wantErrorMessage. 
	// If ParseRunArgs does not return an error or if the error message does not contain the expected substring, the test fails.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// We call ParseRunArgs with the test arguments and expect it to return an error. If it returns nil, the test fails.
			_, err := ParseRunArgs(test.args)
			// We check that the error message contains the expected substring. If it doesn't, the test fails.
			if err == nil {
				t.Fatal("ParseRunArgs returned nil error")
			}
			// We check that the error message contains the expected substring defined in wantErrorMessage. If it doesn't, the test fails.
			if !strings.Contains(err.Error(), test.wantErrorMessage) {
				t.Fatalf("error = %q, want message containing %q", err.Error(), test.wantErrorMessage)
			}
		})
	}
}
