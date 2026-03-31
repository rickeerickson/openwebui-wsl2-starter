//go:build linux

package main

import (
	"testing"
)

func TestDiagnoseCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "diagnose" {
			found = true
			break
		}
	}
	if !found {
		t.Error("diagnose command not registered on rootCmd")
	}
}
