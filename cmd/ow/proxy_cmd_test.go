//go:build windows

package main

import (
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
)

func TestProxySubcommandRegistration(t *testing.T) {
	subs := proxyCmd.Commands()
	names := make(map[string]bool, len(subs))
	for _, s := range subs {
		names[s.Use] = true
	}

	for _, want := range []string{"enable", "remove", "show"} {
		if !names[want] {
			t.Errorf("missing subcommand %q in proxy command", want)
		}
	}
}

func TestProxyCommandRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "proxy" {
			found = true
			break
		}
	}
	if !found {
		t.Error("proxy command not registered on rootCmd")
	}
}

func TestProxyRuleFromConfig(t *testing.T) {
	cfg := config.Defaults()
	rule := proxyRuleFromConfig(cfg)

	if rule.ListenAddress != "0.0.0.0" {
		t.Errorf("ListenAddress = %q, want %q", rule.ListenAddress, "0.0.0.0")
	}
	if rule.ListenPort != 3000 {
		t.Errorf("ListenPort = %d, want %d", rule.ListenPort, 3000)
	}
	if rule.ConnectAddress != "127.0.0.1" {
		t.Errorf("ConnectAddress = %q, want %q", rule.ConnectAddress, "127.0.0.1")
	}
	if rule.ConnectPort != 3000 {
		t.Errorf("ConnectPort = %d, want %d", rule.ConnectPort, 3000)
	}
}

func TestFirewallRuleName(t *testing.T) {
	cfg := config.Defaults()
	name := firewallRuleName(cfg)
	want := "OpenWebUI-0.0.0.0-3000"
	if name != want {
		t.Errorf("firewallRuleName = %q, want %q", name, want)
	}
}
