package rate

import (
	"sync"
	"time"
)

// Limiter is a rate limiter for controlling request rates
type Limiter struct {
	rate   time.Duration
	burst  int
	tokens int
	last   time.Time
	mu     sync.Mutex
}

// NewLimiter creates a new rate limiter that allows events up to rate and can burst up to burst events
func NewLimiter(rate time.Duration, burst int) *Limiter {
	return &Limiter{
		rate:   rate,
		burst:  burst,
		tokens: burst,
		last:   time.Now(),
	}
}

// Every creates a rate duration from frequency (e.g., rate.Every(100*time.Millisecond))
func Every(interval time.Duration) time.Duration {
	return interval
}

// Allow returns true if a request should be allowed now
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last)
	l.last = now

	// Add tokens based on elapsed time
	l.tokens += int(elapsed / l.rate)
	if l.tokens > l.burst {
		l.tokens = l.burst
	}

	if l.tokens <= 0 {
		return false
	}

	l.tokens--
	return true
}

// Wait blocks until a request is allowed
func (l *Limiter) Wait() {
	for !l.Allow() {
		time.Sleep(l.rate)
	}
}
