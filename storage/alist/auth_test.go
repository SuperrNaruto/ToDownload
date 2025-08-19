package alist

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	a := &Alist{}
	
	tests := []struct {
		name     string
		failures int
		minTime  time.Duration
		maxTime  time.Duration
	}{
		{"no failures", 0, 0, 0},
		{"first failure", 1, 750*time.Millisecond, 1500*time.Millisecond},
		{"second failure", 2, 1500*time.Millisecond, 3*time.Second}, 
		{"third failure", 3, 3*time.Second, 6*time.Second},
		{"max backoff", 10, 45*time.Second, 80*time.Second},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := a.calculateBackoff(tt.failures)
			if backoff < tt.minTime || backoff > tt.maxTime {
				t.Errorf("calculateBackoff(%d) = %v, want between %v and %v", 
					tt.failures, backoff, tt.minTime, tt.maxTime)
			}
		})
	}
}

func TestAuthState(t *testing.T) {
	a := &Alist{
		authState: authState{
			isAuthenticated:     false,
			consecutiveFailures: 0,
		},
	}
	
	// Test initial state
	if a.authState.isAuthenticated {
		t.Error("Expected initial auth state to be false")
	}
	
	if a.authState.consecutiveFailures != 0 {
		t.Errorf("Expected initial failure count to be 0, got %d", a.authState.consecutiveFailures)
	}
	
	// Test cooldown calculation
	if !a.authState.cooldownUntil.IsZero() {
		t.Error("Expected initial cooldown to be zero time")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{10, 10, 10},
		{0, -1, -1},
	}
	
	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
		}
	}
}