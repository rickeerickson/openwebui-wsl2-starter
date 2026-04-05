//go:build windows

package netsh

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

type mockCall struct {
	Name string
	Args []string
}

type mockRunner struct {
	calls   []mockCall
	outputs map[string]string
	errors  map[string]error
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (r *mockRunner) key(name string, args ...string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func (r *mockRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	r.calls = append(r.calls, mockCall{Name: name, Args: args})
	k := r.key(name, args...)
	if err, ok := r.errors[k]; ok {
		return r.outputs[k], err
	}
	return r.outputs[k], nil
}

func (r *mockRunner) RunWithRetry(ctx context.Context, _ exec.RetryOpts, name string, args ...string) (string, error) {
	return r.Run(ctx, name, args...)
}

func (r *mockRunner) called(name string, args ...string) bool {
	k := r.key(name, args...)
	for _, c := range r.calls {
		ck := r.key(c.Name, c.Args...)
		if ck == k {
			return true
		}
	}
	return false
}

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	var buf bytes.Buffer
	l, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	return l
}

func TestParseProxyRulesEmpty(t *testing.T) {
	rules := parseProxyRules("")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestParseProxyRulesTypicalOutput(t *testing.T) {
	output := `
Listen on ipv4:             Connect to ipv4:

Address         Port        Address         Port
--------------- ----------  --------------- ----------
0.0.0.0         3000        127.0.0.1       3000
192.168.1.1     8080        10.0.0.1        80
`
	rules := parseProxyRules(output)
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	if rules[0].ListenAddress != "0.0.0.0" || rules[0].ListenPort != 3000 {
		t.Errorf("rule 0 listen: got %s:%d", rules[0].ListenAddress, rules[0].ListenPort)
	}
	if rules[0].ConnectAddress != "127.0.0.1" || rules[0].ConnectPort != 3000 {
		t.Errorf("rule 0 connect: got %s:%d", rules[0].ConnectAddress, rules[0].ConnectPort)
	}

	if rules[1].ListenAddress != "192.168.1.1" || rules[1].ListenPort != 8080 {
		t.Errorf("rule 1 listen: got %s:%d", rules[1].ListenAddress, rules[1].ListenPort)
	}
}

func TestParseProxyRulesHeaderOnly(t *testing.T) {
	output := `Listen on ipv4:             Connect to ipv4:

Address         Port        Address         Port
--------------- ----------  --------------- ----------
`
	rules := parseProxyRules(output)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestAddPortProxySkipsWhenExists(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "interface", "portproxy", "show", "v4tov4")
	r.outputs[showKey] = "0.0.0.0         3000        127.0.0.1       3000\n"

	c := NewClient(r, newTestLogger(t))
	err := c.AddPortProxy(context.Background(), ProxyRule{
		ListenAddress:  "0.0.0.0",
		ListenPort:     3000,
		ConnectAddress: "127.0.0.1",
		ConnectPort:    3000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT have called add.
	if r.called("netsh", "interface", "portproxy", "add", "v4tov4",
		"listenaddress=0.0.0.0", "listenport=3000",
		"connectaddress=127.0.0.1", "connectport=3000") {
		t.Error("should not add when rule already exists")
	}
}

func TestAddPortProxyCreatesNewRule(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "interface", "portproxy", "show", "v4tov4")
	r.outputs[showKey] = "" // no existing rules

	c := NewClient(r, newTestLogger(t))
	err := c.AddPortProxy(context.Background(), ProxyRule{
		ListenAddress:  "0.0.0.0",
		ListenPort:     3000,
		ConnectAddress: "127.0.0.1",
		ConnectPort:    3000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("netsh", "interface", "portproxy", "add", "v4tov4",
		"listenaddress=0.0.0.0", "listenport=3000",
		"connectaddress=127.0.0.1", "connectport=3000") {
		t.Error("expected netsh add portproxy call")
	}
}

func TestAddPortProxyRemovesConflicting(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "interface", "portproxy", "show", "v4tov4")
	// Existing rule has different connect address.
	r.outputs[showKey] = "0.0.0.0         3000        10.0.0.1        3000\n"

	c := NewClient(r, newTestLogger(t))
	err := c.AddPortProxy(context.Background(), ProxyRule{
		ListenAddress:  "0.0.0.0",
		ListenPort:     3000,
		ConnectAddress: "127.0.0.1",
		ConnectPort:    3000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called delete first.
	if !r.called("netsh", "interface", "portproxy", "delete", "v4tov4",
		"listenaddress=0.0.0.0", "listenport=3000") {
		t.Error("expected delete of conflicting rule")
	}
}

func TestRemovePortProxy(t *testing.T) {
	r := newMockRunner()
	c := NewClient(r, newTestLogger(t))

	err := c.RemovePortProxy(context.Background(), "0.0.0.0", 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("netsh", "interface", "portproxy", "delete", "v4tov4",
		"listenaddress=0.0.0.0", "listenport=3000") {
		t.Error("expected netsh delete call")
	}
}

func TestPortProxyExistsTrue(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "interface", "portproxy", "show", "v4tov4")
	r.outputs[showKey] = "0.0.0.0         3000        127.0.0.1       3000\n"

	c := NewClient(r, newTestLogger(t))
	exists, err := c.PortProxyExists(context.Background(), "0.0.0.0", 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestPortProxyExistsFalse(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "interface", "portproxy", "show", "v4tov4")
	r.outputs[showKey] = ""

	c := NewClient(r, newTestLogger(t))
	exists, err := c.PortProxyExists(context.Background(), "0.0.0.0", 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false")
	}
}

func TestAddFirewallRuleSkipsWhenExists(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "advfirewall", "firewall", "show", "rule", "name=OpenWebUI-3000")
	r.outputs[showKey] = "Rule Name: OpenWebUI-3000\n"

	c := NewClient(r, newTestLogger(t))
	err := c.AddFirewallRule(context.Background(), "OpenWebUI-3000", 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT have called add.
	for _, call := range r.calls {
		if call.Name == "netsh" && len(call.Args) > 2 && call.Args[2] == "add" {
			t.Error("should not add firewall rule when it already exists")
		}
	}
}

func TestAddFirewallRuleCreatesNew(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "advfirewall", "firewall", "show", "rule", "name=OpenWebUI-3000")
	r.outputs[showKey] = "No rules match the specified criteria."
	r.errors[showKey] = fmt.Errorf("exit status 1")

	c := NewClient(r, newTestLogger(t))
	err := c.AddFirewallRule(context.Background(), "OpenWebUI-3000", 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("netsh", "advfirewall", "firewall", "add", "rule",
		"name=OpenWebUI-3000", "dir=in", "action=allow", "protocol=TCP", "localport=3000") {
		t.Error("expected netsh add firewall rule call")
	}
}

func TestRemoveFirewallRule(t *testing.T) {
	r := newMockRunner()
	c := NewClient(r, newTestLogger(t))

	err := c.RemoveFirewallRule(context.Background(), "OpenWebUI-3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !r.called("netsh", "advfirewall", "firewall", "delete", "rule", "name=OpenWebUI-3000") {
		t.Error("expected netsh delete firewall rule call")
	}
}

func TestFirewallRuleExistsTrue(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "advfirewall", "firewall", "show", "rule", "name=Test")
	r.outputs[showKey] = "Rule Name: Test\nEnabled: Yes\n"

	c := NewClient(r, newTestLogger(t))
	exists, err := c.FirewallRuleExists(context.Background(), "Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected true")
	}
}

func TestFirewallRuleExistsFalse(t *testing.T) {
	r := newMockRunner()
	showKey := r.key("netsh", "advfirewall", "firewall", "show", "rule", "name=Missing")
	r.outputs[showKey] = "No rules match the specified criteria."
	r.errors[showKey] = fmt.Errorf("exit status 1")

	c := NewClient(r, newTestLogger(t))
	exists, err := c.FirewallRuleExists(context.Background(), "Missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected false")
	}
}
