// Package ratelimit is a per-key token bucket (one bucket per client IP).
package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type entry struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

type Limiter struct {
	mu    sync.Mutex
	m     map[string]*entry
	limit rate.Limit
	burst int
}

func New(limit rate.Limit, burst int) *Limiter {
	return &Limiter{m: map[string]*entry{}, limit: limit, burst: burst}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.m[key]
	if !ok {
		e = &entry{lim: rate.NewLimiter(l.limit, l.burst)}
		l.m[key] = e
	}
	e.lastSeen = time.Now()
	return e.lim.Allow()
}

// Sweep drops buckets idle for longer than maxIdle so the map cannot grow forever.
func (l *Limiter) Sweep(maxIdle time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, e := range l.m {
		if time.Since(e.lastSeen) > maxIdle {
			delete(l.m, k)
		}
	}
}

// RunSweeper calls Sweep every minute until ctx is cancelled.
func (l *Limiter) RunSweeper(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			l.Sweep(10 * time.Minute)
		}
	}
}
