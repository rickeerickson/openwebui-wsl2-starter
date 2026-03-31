//go:build linux

package main

import (
	"testing"
)

func TestSetupCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "setup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("setup command not registered on rootCmd")
	}
}

func TestSetupCommandDescription(t *testing.T) {
	if setupCmd.Short == "" {
		t.Error("setup command has no short description")
	}
	if setupCmd.Long == "" {
		t.Error("setup command has no long description")
	}
}
