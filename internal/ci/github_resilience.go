package ci

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

var (
	ErrGitHubUnavailable = errors.New("github api unavailable")
	ErrGitHubCircuitOpen = errors.New("github circuit open")
)

const (
	githubDefaultMaxAttempts = 3
	githubCircuitThreshold   = 5
	githubCircuitCooldown    = 30 * time.Second
)

type githubHTTPError struct {
	StatusCode int
	Status     string
}

func (e *githubHTTPError) Error() string {
	if e == nil {
		return "github API error"
	}
	return fmt.Sprintf("github API returned %s", e.Status)
}

type githubCircuit struct {
	mu        sync.Mutex
	failures  int
	openUntil time.Time
}

func (c *githubCircuit) Allow() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.openUntil.IsZero() && time.Now().Before(c.openUntil) {
		return ErrGitHubCircuitOpen
	}
	return nil
}

func (c *githubCircuit) RecordSuccess() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.openUntil = time.Time{}
}

func (c *githubCircuit) RecordFailure() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	if c.failures >= githubCircuitThreshold {
		c.openUntil = time.Now().Add(githubCircuitCooldown)
	}
}

func (c *githubCircuit) resetForTest() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.openUntil = time.Time{}
}

var defaultGitHubCircuit = &githubCircuit{}

func isRetryableGitHubError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrGitHubCircuitOpen) {
		return false
	}
	var httpErr *githubHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == 429 || httpErr.StatusCode >= 500
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "tls handshake")
}

func githubBackoff(attempt int) time.Duration {
	// attempt is 0-based: 50ms, 100ms, 200ms
	d := 50 * time.Millisecond
	for i := 0; i < attempt; i++ {
		d *= 2
	}
	return d
}
