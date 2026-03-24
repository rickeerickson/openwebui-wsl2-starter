//go:build linux

package docker

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/exec"
	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

func newHealthClient(t *testing.T, calls []mockCall) (*Client, *mockRunner) {
	t.Helper()
	var buf bytes.Buffer
	logger, err := logging.NewLoggerWithWriter(&buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	mr := &mockRunner{calls: calls, t: t}
	return NewClient(mr, logger), mr
}

func TestCheckHTTPSucceeds(t *testing.T) {
	c, m := newHealthClient(t, []mockCall{
		{wantName: "curl", wantArgs: []string{"-sf", "http://localhost:3000/"},
			output: "", err: nil},
	})
	err := c.CheckHTTP(context.Background(), "localhost", 3000, "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestCheckHTTPFails(t *testing.T) {
	c, m := newHealthClient(t, []mockCall{
		{wantName: "curl", wantArgs: []string{"-sf", "http://localhost:3000/health"},
			output: "", err: fmt.Errorf("command \"curl\" failed: exit status 22")},
	})
	err := c.CheckHTTP(context.Background(), "localhost", 3000, "/health")
	if err == nil {
		t.Fatal("expected error for failed curl")
	}
	m.verify()
}

func TestWaitForHTTPRetriesUntilSuccess(t *testing.T) {
	c, m := newHealthClient(t, []mockCall{
		// First attempt: fail.
		{wantName: "curl", output: "", err: fmt.Errorf("command \"curl\" failed: connection refused")},
		// Second attempt: succeed.
		{wantName: "curl", output: "", err: nil},
	})

	opts := exec.RetryOpts{
		MaxAttempts: 3,
		InitialA:    time.Millisecond,
		InitialB:    time.Millisecond,
	}
	err := c.WaitForHTTP(context.Background(), "localhost", 3000, "/", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.verify()
}

func TestWaitForHTTPFailsAfterMaxAttempts(t *testing.T) {
	c, m := newHealthClient(t, []mockCall{
		{wantName: "curl", output: "", err: fmt.Errorf("command \"curl\" failed: connection refused")},
		{wantName: "curl", output: "", err: fmt.Errorf("command \"curl\" failed: connection refused")},
		{wantName: "curl", output: "", err: fmt.Errorf("command \"curl\" failed: connection refused")},
	})

	opts := exec.RetryOpts{
		MaxAttempts: 3,
		InitialA:    time.Millisecond,
		InitialB:    time.Millisecond,
	}
	err := c.WaitForHTTP(context.Background(), "localhost", 3000, "/", opts)
	if err == nil {
		t.Fatal("expected error after max attempts")
	}
	if !contains(err.Error(), "after 3 attempts") {
		t.Errorf("error = %q, want it to contain 'after 3 attempts'", err.Error())
	}
	m.verify()
}

func TestWaitForHTTPInvalidMaxAttempts(t *testing.T) {
	c, _ := newHealthClient(t, nil)
	opts := exec.RetryOpts{MaxAttempts: 0}
	err := c.WaitForHTTP(context.Background(), "localhost", 3000, "/", opts)
	if err == nil {
		t.Fatal("expected error for MaxAttempts=0")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
