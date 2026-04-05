//go:build windows

package main

import (
	"testing"
)

func TestWslSubcommandRegistration(t *testing.T) {
	subs := wslCmd.Commands()
	names := make(map[string]bool, len(subs))
	for _, s := range subs {
		names[s.Use] = true
	}

	for _, want := range []string{"install", "remove", "stop"} {
		if !names[want] {
			t.Errorf("missing subcommand %q in wsl command", want)
		}
	}
}

func TestWslCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "wsl" {
			found = true
			break
		}
	}
	if !found {
		t.Error("wsl command not registered on rootCmd")
	}
}
