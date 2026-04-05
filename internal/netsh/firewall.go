//go:build windows

package netsh

import (
	"context"
	"fmt"
	"strings"
)

// AddFirewallRule creates an inbound TCP firewall rule via netsh advfirewall.
// Skips if a rule with the same name already exists.
func (c *Client) AddFirewallRule(ctx context.Context, name string, port int) error {
	exists, err := c.FirewallRuleExists(ctx, name)
	if err != nil {
		c.Logger.Warn("could not check firewall rule: %v", err)
	}
	if exists {
		c.Logger.Info("firewall rule %q already exists, skipping", name)
		return nil
	}

	c.Logger.Info("adding firewall rule %q for TCP port %d", name, port)
	_, err = c.Runner.Run(ctx, "netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%s", name),
		"dir=in",
		"action=allow",
		"protocol=TCP",
		fmt.Sprintf("localport=%d", port))
	if err != nil {
		return fmt.Errorf("netsh add firewall rule: %w", err)
	}
	return nil
}

// RemoveFirewallRule deletes a firewall rule by name.
func (c *Client) RemoveFirewallRule(ctx context.Context, name string) error {
	c.Logger.Info("removing firewall rule %q", name)
	_, err := c.Runner.Run(ctx, "netsh", "advfirewall", "firewall", "delete", "rule",
		fmt.Sprintf("name=%s", name))
	if err != nil {
		return fmt.Errorf("netsh delete firewall rule: %w", err)
	}
	return nil
}

// FirewallRuleExists returns true if a firewall rule with the given name exists.
func (c *Client) FirewallRuleExists(ctx context.Context, name string) (bool, error) {
	out, err := c.Runner.Run(ctx, "netsh", "advfirewall", "firewall", "show", "rule",
		fmt.Sprintf("name=%s", name))
	if err != nil {
		// netsh exits non-zero when no matching rule is found.
		if strings.Contains(out, "No rules match") {
			return false, nil
		}
		return false, fmt.Errorf("netsh show firewall rule: %w", err)
	}
	return true, nil
}
