// Package clock gives code the current time through a value that a test can
// replace.
package clock

import (
	"sync"
	"time"

	"go.uber.org/fx"
)

// Clock tells the current time.
type Clock interface {
	Now() time.Time
}

// Module provides the real Clock.
var Module = fx.Module("clock",
	fx.Provide(func() Clock { return Real{} }),
)

// Real reads the system clock.
type Real struct{}

// Now returns time.Now.
func (Real) Now() time.Time { return time.Now() }

// Fixed is a Clock that a test sets and moves. It is safe for concurrent
// use.
type Fixed struct {
	mu  sync.Mutex
	now time.Time
}

// NewFixed returns a Fixed clock at now.
func NewFixed(now time.Time) *Fixed {
	return &Fixed{now: now}
}

// Now returns the time the clock is at.
func (c *Fixed) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *Fixed) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
