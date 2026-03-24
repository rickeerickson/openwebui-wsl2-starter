package exec

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rickeerickson/openwebui-wsl2-starter/internal/logging"
)

// testBins is an allowlist that includes common test utilities.
var testBins = map[string]bool{
	"echo":  true,
	"false": true,
	"sleep": true,
	"sh":    true,
}

func newTestRunner(t *testing.T, buf *bytes.Buffer) *RealRunner {
	t.Helper()
	logger, err := logging.NewLoggerWithWriter(buf, "", logging.Debug2)
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	return &RealRunner{
		Logger:      logger,
		AllowedBins: testBins,
	}
}

func TestRunAllowedBinary(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	out, err := r.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(out), "hello")
	}
}

func TestRunDisallowedBinary(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	_, err := r.Run(context.Background(), "curl", "http://example.com")
	if err == nil {
		t.Fatal("expected error for disallowed binary")
	}
	if !strings.Contains(err.Error(), "not in the allowlist") {
		t.Errorf("error = %q, want it to contain 'not in the allowlist'", err.Error())
	}
}

func TestRunLogsCommand(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	_, _ = r.Run(context.Background(), "echo", "logged")

	logged := buf.String()
	if !strings.Contains(logged, "exec: echo") {
		t.Errorf("log output missing command, got: %s", logged)
	}
}

func TestRunReturnsCombinedOutput(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	out, err := r.Run(context.Background(), "sh", "-c", "echo stdout; echo stderr >&2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "stdout") {
		t.Errorf("output missing stdout, got: %q", out)
	}
	if !strings.Contains(out, "stderr") {
		t.Errorf("output missing stderr, got: %q", out)
	}
}

func TestRunNonzeroExitCode(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	_, err := r.Run(context.Background(), "false")
	if err == nil {
		t.Fatal("expected error for nonzero exit code")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("error = %q, want it to contain 'failed'", err.Error())
	}
}

func TestRunRespectsContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := r.Run(ctx, "sleep", "10")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestRunWithRetrySucceedsFirstAttempt(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	opts := RetryOpts{MaxAttempts: 3, InitialA: time.Millisecond, InitialB: time.Millisecond}
	out, err := r.RunWithRetry(context.Background(), opts, "echo", "ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "ok" {
		t.Errorf("output = %q, want %q", strings.TrimSpace(out), "ok")
	}
	// No retry log lines should appear.
	if strings.Contains(buf.String(), "retry") {
		t.Errorf("unexpected retry log: %s", buf.String())
	}
}

func TestRunWithRetryRetriesOnFailure(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	// sh -c script that fails twice then succeeds using a temp file as counter.
	// We use RunWithRetry on "false" with 2 attempts to confirm retry happens.
	opts := RetryOpts{MaxAttempts: 2, InitialA: time.Millisecond, InitialB: time.Millisecond}
	_, err := r.RunWithRetry(context.Background(), opts, "false")
	if err == nil {
		t.Fatal("expected error after retries")
	}
	logged := buf.String()
	if !strings.Contains(logged, "retry 1/2") {
		t.Errorf("expected retry log, got: %s", logged)
	}
}

func TestRunWithRetryExhaustsAttempts(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	opts := RetryOpts{MaxAttempts: 3, InitialA: time.Millisecond, InitialB: time.Millisecond}
	_, err := r.RunWithRetry(context.Background(), opts, "false")
	if err == nil {
		t.Fatal("expected error after exhausted attempts")
	}
	if !strings.Contains(err.Error(), "after 3 attempts") {
		t.Errorf("error = %q, want it to contain 'after 3 attempts'", err.Error())
	}
}

func TestRunWithRetryRespectsContextDuringSleep(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay so the first attempt runs, then cancellation
	// hits during the retry sleep.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	opts := RetryOpts{MaxAttempts: 5, InitialA: 10 * time.Second, InitialB: 10 * time.Second}
	_, err := r.RunWithRetry(ctx, opts, "false")
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}
	if !strings.Contains(err.Error(), "context cancelled") {
		t.Errorf("error = %q, want it to contain 'context cancelled'", err.Error())
	}
}

func TestAllowlistIsCaseSensitive(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	// "echo" is allowed, "Echo" is not.
	_, err := r.Run(context.Background(), "Echo", "hello")
	if err == nil {
		t.Fatal("expected error for case-mismatched binary")
	}
	if !strings.Contains(err.Error(), "not in the allowlist") {
		t.Errorf("error = %q, want allowlist error", err.Error())
	}
}

func TestRunEmptyCommandName(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRunner(t, &buf)

	_, err := r.Run(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty command name")
	}
	if !strings.Contains(err.Error(), "empty command name") {
		t.Errorf("error = %q, want it to contain 'empty command name'", err.Error())
	}
}
