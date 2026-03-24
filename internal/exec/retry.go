package exec

import "time"

// RetryOpts configures retry behavior with Fibonacci backoff.
type RetryOpts struct {
	MaxAttempts int
	InitialA    time.Duration
	InitialB    time.Duration
}

// DefaultRetryOpts returns the standard retry configuration:
// 5 attempts with Fibonacci delays starting at 10s, 10s.
// The resulting sequence is 10, 10, 20, 30, 50, 80 seconds.
func DefaultRetryOpts() RetryOpts {
	return RetryOpts{
		MaxAttempts: 5,
		InitialA:    10 * time.Second,
		InitialB:    10 * time.Second,
	}
}

// NextDelay returns the current delay and the next Fibonacci pair.
// Given (a, b), the delay is a, and the next pair is (b, a+b).
func NextDelay(a, b time.Duration) (delay, nextA, nextB time.Duration) {
	return a, b, a + b
}
