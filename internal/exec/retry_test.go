package exec

import (
	"math"
	"testing"
	"time"
)

func TestDefaultRetryOpts(t *testing.T) {
	opts := DefaultRetryOpts()

	if opts.MaxAttempts != 5 {
		t.Errorf("MaxAttempts = %d, want 5", opts.MaxAttempts)
	}
	if opts.InitialA != 10*time.Second {
		t.Errorf("InitialA = %v, want 10s", opts.InitialA)
	}
	if opts.InitialB != 10*time.Second {
		t.Errorf("InitialB = %v, want 10s", opts.InitialB)
	}
}

func TestFibonacciSequence(t *testing.T) {
	// The bash implementation uses: 10, 10, 20, 30, 50, 80
	expected := []time.Duration{
		10 * time.Second,
		10 * time.Second,
		20 * time.Second,
		30 * time.Second,
		50 * time.Second,
		80 * time.Second,
	}

	a := 10 * time.Second
	b := 10 * time.Second

	for i, want := range expected {
		var delay time.Duration
		delay, a, b = NextDelay(a, b)
		if delay != want {
			t.Errorf("iteration %d: delay = %v, want %v", i, delay, want)
		}
	}
}

func TestNextDelayZeroDurations(t *testing.T) {
	delay, nextA, nextB := NextDelay(0, 0)

	if delay != 0 {
		t.Errorf("delay = %v, want 0", delay)
	}
	if nextA != 0 {
		t.Errorf("nextA = %v, want 0", nextA)
	}
	if nextB != 0 {
		t.Errorf("nextB = %v, want 0", nextB)
	}
}

func TestNextDelayAsymmetricValues(t *testing.T) {
	// Start with (1s, 3s) and verify the sequence: 1, 3, 4, 7, 11
	expected := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		4 * time.Second,
		7 * time.Second,
		11 * time.Second,
	}

	a := 1 * time.Second
	b := 3 * time.Second

	for i, want := range expected {
		var delay time.Duration
		delay, a, b = NextDelay(a, b)
		if delay != want {
			t.Errorf("iteration %d: delay = %v, want %v", i, delay, want)
		}
	}
}

func TestNextDelaySingleIteration(t *testing.T) {
	delay, nextA, nextB := NextDelay(10*time.Second, 10*time.Second)

	if delay != 10*time.Second {
		t.Errorf("delay = %v, want 10s", delay)
	}
	if nextA != 10*time.Second {
		t.Errorf("nextA = %v, want 10s", nextA)
	}
	if nextB != 20*time.Second {
		t.Errorf("nextB = %v, want 20s", nextB)
	}
}

func TestNextDelayLargeValues(t *testing.T) {
	// Use large durations near the max int64 nanosecond range.
	// This should not panic.
	large := time.Duration(math.MaxInt64 / 4)
	delay, nextA, nextB := NextDelay(large, large)

	if delay != large {
		t.Errorf("delay = %v, want %v", delay, large)
	}
	if nextA != large {
		t.Errorf("nextA = %v, want %v", nextA, large)
	}
	if nextB != 2*large {
		t.Errorf("nextB = %v, want %v", nextB, 2*large)
	}
}
