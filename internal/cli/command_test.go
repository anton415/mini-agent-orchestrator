package cli

import (
	"strings"
	"testing"
	"time"
)

// TestParseRunArgsAcceptsOneInputSource checks the two valid ways to provide
// workflow input: inline text with --idea or a markdown file path with --input.
func TestParseRunArgsAcceptsOneInputSource(t *testing.T) {
	// Each case includes only one input source and relies on the parser to apply
	// default values for optional flags that are not provided.
	tests := []struct {
		name string
		args []string
		want RunConfig
	}{
		{
			name: "idea only",
			args: []string{"--idea", "Build a personal book library"},
			want: RunConfig{
				Idea: "Build a personal book library",
				Out:  "./artifacts",
				Name: "project",
			},
		},
		{
			name: "input only",
			args: []string{"--input", "docs/mvp.md"},
			want: RunConfig{
				Input: "docs/mvp.md",
				Out:   "./artifacts",
				Name:  "project",
			},
		},
	}

	// Run each valid argument set as a subtest so failures identify the input
	// source that stopped parsing correctly.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseRunArgs(test.args)
			if err != nil {
				t.Fatalf("ParseRunArgs returned an error: %v", err)
			}

			// A full struct comparison catches accidental changes to defaults as
			// well as fields set directly by the provided flags.
			assertRunConfig(t, got, test.want)
		})
	}
}

// TestParseRunArgsRejectsInvalidInput checks the parser boundary before the
// workflow starts. Invalid source combinations and positional args must fail.
func TestParseRunArgsRejectsInvalidInput(t *testing.T) {
	// Each case includes the arguments to parse and the message fragment that
	// should appear in the returned error.
	tests := []struct {
		name        string
		args        []string
		wantMessage string
	}{
		{
			name:        "missing idea and input",
			args:        nil,
			wantMessage: "either --idea or --input is required",
		},
		{
			name:        "idea and input together",
			args:        []string{"--idea", "demo", "--input", "docs/mvp.md"},
			wantMessage: "use either --idea or --input, not both",
		},
		{
			name:        "unexpected positional argument",
			args:        []string{"--idea", "demo", "extra"},
			wantMessage: "unexpected positional argument: extra",
		},
	}

	// Run each invalid argument set as a subtest so one failure does not hide
	// the rest of the validation coverage.
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseRunArgs(test.args)
			if err == nil {
				t.Fatal("ParseRunArgs returned nil error")
			}

			// Check for the important validation message without depending on the
			// exact formatting of errors returned by the flag package.
			if !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("error = %q, want message containing %q", err.Error(), test.wantMessage)
			}
		})
	}
}

// TestParseRunArgsParsesOptionalFlags verifies options that do not decide the
// input source but still need to be carried into the workflow config.
func TestParseRunArgsParsesOptionalFlags(t *testing.T) {
	got, err := ParseRunArgs([]string{
		"--idea", "Build a personal book library",
		"--out", "./artifacts/demo",
		"--name", "demo",
		"--created-at", "2026-06-25T11:23:10Z",
		"--force",
		"--dry-run",
		"--prompts",
	})
	if err != nil {
		t.Fatalf("ParseRunArgs returned an error: %v", err)
	}

	// This expected config combines the required --idea input with explicit
	// values for the optional output, name, force, and dry-run flags.
	want := RunConfig{
		Idea:      "Build a personal book library",
		Out:       "./artifacts/demo",
		Name:      "demo",
		CreatedAt: time.Date(2026, 6, 25, 11, 23, 10, 0, time.UTC),
		Force:     true,
		DryRun:    true,
		Prompts:   true,
	}
	assertRunConfig(t, got, want)
}

func TestParseRunArgsRejectsInvalidCreatedAt(t *testing.T) {
	_, err := ParseRunArgs([]string{
		"--idea", "Build a personal book library",
		"--created-at", "June 25",
	})
	if err == nil {
		t.Fatal("ParseRunArgs returned nil error")
	}
	if !strings.Contains(err.Error(), "invalid --created-at") {
		t.Fatalf("error = %q, want invalid --created-at message", err.Error())
	}
}

// assertRunConfig keeps individual tests focused on their argument setup while
// still reporting the complete parsed config when a field differs.
func assertRunConfig(t *testing.T, got RunConfig, want RunConfig) {
	t.Helper()

	if got != want {
		t.Fatalf("RunConfig = %#v, want %#v", got, want)
	}
}
