package api

import (
	"strings"
	"testing"
)

func TestRateLimitError_WithRetryAfter(t *testing.T) {
	err := &RateLimitError{
		Detail:            "Request was throttled.",
		RetryAfterSeconds: 42,
	}

	msg := err.Error()
	if !strings.Contains(msg, "Rate limited") {
		t.Errorf("Error() = %q, want to contain 'Rate limited'", msg)
	}
	if !strings.Contains(msg, "retry in 42s") {
		t.Errorf("Error() = %q, want to contain 'retry in 42s'", msg)
	}
}

func TestRateLimitError_WithoutRetryAfter(t *testing.T) {
	err := &RateLimitError{
		Detail: "Too many requests.",
	}

	msg := err.Error()
	if !strings.Contains(msg, "Rate limited") {
		t.Errorf("Error() = %q, want to contain 'Rate limited'", msg)
	}
	if strings.Contains(msg, "retry in") {
		t.Errorf("Error() = %q, should not contain 'retry in' when RetryAfterSeconds is 0", msg)
	}
}
