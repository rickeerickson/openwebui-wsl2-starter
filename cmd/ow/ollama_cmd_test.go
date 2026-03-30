//go:build linux

package main

import (
	"testing"
)

func TestOllamaRunRequiresExactlyOneArg(t *testing.T) {
	// The cobra command has Args: cobra.ExactArgs(1).
	// Verify the command is configured correctly.
	if ollamaRunCmd.Args == nil {
		t.Fatal("ollamaRunCmd.Args should not be nil")
	}

	// Zero args should fail.
	if err := ollamaRunCmd.Args(ollamaRunCmd, []string{}); err == nil {
		t.Error("expected error with zero args")
	}

	// One arg should pass.
	if err := ollamaRunCmd.Args(ollamaRunCmd, []string{"llama3.2:1b"}); err != nil {
		t.Errorf("expected no error with one arg, got: %v", err)
	}

	// Two args should fail.
	if err := ollamaRunCmd.Args(ollamaRunCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with two args")
	}
}

func TestOllamaPullAcceptsVariadicArgs(t *testing.T) {
	// ollamaPullCmd has no Args constraint, so any number should work.
	// With zero args it falls back to config models.
	// This test just verifies the command exists and is wired up.
	if ollamaPullCmd.Use != "pull" {
		t.Errorf("Use = %q, want %q", ollamaPullCmd.Use, "pull")
	}
}

func TestOllamaModelsCommandExists(t *testing.T) {
	if ollamaModelsCmd.Use != "models" {
		t.Errorf("Use = %q, want %q", ollamaModelsCmd.Use, "models")
	}
}

func TestOllamaSubcommandRegistration(t *testing.T) {
	subs := ollamaCmd.Commands()
	names := make(map[string]bool, len(subs))
	for _, s := range subs {
		names[s.Use] = true
	}

	for _, want := range []string{"pull", "models", "run [model]"} {
		if !names[want] {
			t.Errorf("missing subcommand %q in ollama command", want)
		}
	}
}
