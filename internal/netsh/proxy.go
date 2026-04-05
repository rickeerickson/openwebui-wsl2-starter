//go:build windows

// Package netsh manages Windows port proxy and firewall rules via netsh.
package netsh

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// ProxyRule represents a single v4tov4 port proxy rule.
type ProxyRule struct {
	ListenAddress  string
	ListenPort     int
	ConnectAddress string
	ConnectPort    int
}

// Client manages netsh operations.
type Client struct {
	Runner exec.Runner
	Logger *logging.Logger
}

// NewClient creates a Client with the given runner and logger.
func NewClient(runner exec.Runner, logger *logging.Logger) *Client {
	return &Client{Runner: runner, Logger: logger}
}

// AddPortProxy creates a v4tov4 port proxy rule. If a rule already exists
// for the same listen address and port, it is removed first.
func (c *Client) AddPortProxy(ctx context.Context, rule ProxyRule) error {
	// Check for existing rule and remove if different.
	existing, err := c.ListPortProxy(ctx)
	if err != nil {
		c.Logger.Warn("could not list existing proxy rules: %v", err)
	}

	for _, r := range existing {
		if r.ListenAddress == rule.ListenAddress && r.ListenPort == rule.ListenPort {
			if r.ConnectAddress == rule.ConnectAddress && r.ConnectPort == rule.ConnectPort {
				c.Logger.Info("port proxy rule already exists, skipping")
				return nil
			}
			c.Logger.Info("removing conflicting proxy rule for %s:%d",
				rule.ListenAddress, rule.ListenPort)
			if err := c.RemovePortProxy(ctx, rule.ListenAddress, rule.ListenPort); err != nil {
				return fmt.Errorf("remove conflicting rule: %w", err)
			}
		}
	}

	c.Logger.Info("adding port proxy %s:%d -> %s:%d",
		rule.ListenAddress, rule.ListenPort,
		rule.ConnectAddress, rule.ConnectPort)

	_, err = c.Runner.Run(ctx, "netsh", "interface", "portproxy", "add", "v4tov4",
		fmt.Sprintf("listenaddress=%s", rule.ListenAddress),
		fmt.Sprintf("listenport=%d", rule.ListenPort),
		fmt.Sprintf("connectaddress=%s", rule.ConnectAddress),
		fmt.Sprintf("connectport=%d", rule.ConnectPort))
	if err != nil {
		return fmt.Errorf("netsh add portproxy: %w", err)
	}

	return nil
}

// RemovePortProxy deletes a v4tov4 port proxy rule.
func (c *Client) RemovePortProxy(ctx context.Context, listenAddr string, listenPort int) error {
	c.Logger.Info("removing port proxy for %s:%d", listenAddr, listenPort)
	_, err := c.Runner.Run(ctx, "netsh", "interface", "portproxy", "delete", "v4tov4",
		fmt.Sprintf("listenaddress=%s", listenAddr),
		fmt.Sprintf("listenport=%d", listenPort))
	if err != nil {
		return fmt.Errorf("netsh delete portproxy: %w", err)
	}
	return nil
}

// ListPortProxy returns all v4tov4 port proxy rules.
func (c *Client) ListPortProxy(ctx context.Context) ([]ProxyRule, error) {
	out, err := c.Runner.Run(ctx, "netsh", "interface", "portproxy", "show", "v4tov4")
	if err != nil {
		return nil, fmt.Errorf("netsh show portproxy: %w", err)
	}
	return parseProxyRules(out), nil
}

// PortProxyExists returns true if a proxy rule exists for the given listen
// address and port.
func (c *Client) PortProxyExists(ctx context.Context, listenAddr string, listenPort int) (bool, error) {
	rules, err := c.ListPortProxy(ctx)
	if err != nil {
		return false, err
	}
	for _, r := range rules {
		if r.ListenAddress == listenAddr && r.ListenPort == listenPort {
			return true, nil
		}
	}
	return false, nil
}

// proxyPattern matches netsh portproxy output lines like:
// "0.0.0.0         3000         127.0.0.1       3000"
// or the colon-separated format:
// "0.0.0.0:3000        127.0.0.1:3000"
var proxyPattern = regexp.MustCompile(
	`^(\d{1,3}(?:\.\d{1,3}){3})\s+(\d+)\s+(\d{1,3}(?:\.\d{1,3}){3})\s+(\d+)$`)

// parseProxyRules extracts ProxyRule entries from netsh portproxy output.
// The output format is a header followed by data lines:
//
//	Listen on ipv4:             Connect to ipv4:
//
//	Address         Port        Address         Port
//	--------------- ----------  --------------- ----------
//	0.0.0.0         3000        127.0.0.1       3000
func parseProxyRules(output string) []ProxyRule {
	var rules []ProxyRule
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		m := proxyPattern.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		var lport, cport int
		fmt.Sscanf(m[2], "%d", &lport)
		fmt.Sscanf(m[4], "%d", &cport)
		rules = append(rules, ProxyRule{
			ListenAddress:  m[1],
			ListenPort:     lport,
			ConnectAddress: m[3],
			ConnectPort:    cport,
		})
	}
	return rules
}
