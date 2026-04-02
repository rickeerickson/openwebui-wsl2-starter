//go:build windows

package main

import (
	"fmt"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/config"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/netsh"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage port proxy and firewall rules",
}

var proxyEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Create port proxy and firewall rule for OpenWebUI access",
	RunE:  runProxyEnable,
}

var proxyRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove port proxy rules",
	RunE:  runProxyRemove,
}

var proxyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current port proxy rules",
	RunE:  runProxyShow,
}

func init() {
	rootCmd.AddCommand(proxyCmd)
	proxyCmd.AddCommand(proxyEnableCmd)
	proxyCmd.AddCommand(proxyRemoveCmd)
	proxyCmd.AddCommand(proxyShowCmd)
}

func newNetshClient() (*netsh.Client, error) {
	logger, err := logging.NewLogger("", logging.Info)
	if err != nil {
		return nil, fmt.Errorf("creating logger: %w", err)
	}
	runner := &exec.RealRunner{Logger: logger}
	return netsh.NewClient(runner, logger), nil
}

func proxyRuleFromConfig(cfg config.Config) netsh.ProxyRule {
	return netsh.ProxyRule{
		ListenAddress:  cfg.Proxy.ListenAddress,
		ListenPort:     cfg.Proxy.ListenPort,
		ConnectAddress: cfg.Proxy.ConnectAddress,
		ConnectPort:    cfg.Proxy.ConnectPort,
	}
}

func firewallRuleName(cfg config.Config) string {
	return fmt.Sprintf("OpenWebUI-%s-%d", cfg.Proxy.ListenAddress, cfg.Proxy.ListenPort)
}

func runProxyEnable(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client, err := newNetshClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	rule := proxyRuleFromConfig(cfg)

	if err := client.AddPortProxy(ctx, rule); err != nil {
		return fmt.Errorf("add port proxy: %w", err)
	}

	if err := client.AddFirewallRule(ctx, firewallRuleName(cfg), cfg.Proxy.ListenPort); err != nil {
		return fmt.Errorf("add firewall rule: %w", err)
	}

	fmt.Printf("Port proxy enabled: %s:%d -> %s:%d\n",
		rule.ListenAddress, rule.ListenPort,
		rule.ConnectAddress, rule.ConnectPort)
	return nil
}

func runProxyRemove(cmd *cobra.Command, args []string) error {
	cfg, err := config.Resolve(nil)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	client, err := newNetshClient()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	if err := client.RemovePortProxy(ctx, cfg.Proxy.ListenAddress, cfg.Proxy.ListenPort); err != nil {
		return fmt.Errorf("remove port proxy: %w", err)
	}

	if err := client.RemoveFirewallRule(ctx, firewallRuleName(cfg)); err != nil {
		return fmt.Errorf("remove firewall rule: %w", err)
	}

	return nil
}

func runProxyShow(cmd *cobra.Command, args []string) error {
	client, err := newNetshClient()
	if err != nil {
		return err
	}

	rules, err := client.ListPortProxy(cmd.Context())
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		fmt.Println("No port proxy rules found.")
		return nil
	}

	fmt.Printf("%-20s %-10s %-20s %-10s\n", "Listen Address", "Port", "Connect Address", "Port")
	fmt.Printf("%-20s %-10s %-20s %-10s\n", "----", "----", "----", "----")
	for _, r := range rules {
		fmt.Printf("%-20s %-10d %-20s %-10d\n",
			r.ListenAddress, r.ListenPort,
			r.ConnectAddress, r.ConnectPort)
	}
	return nil
}
