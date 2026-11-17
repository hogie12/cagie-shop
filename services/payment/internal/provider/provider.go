// Package provider is a fake payment provider you can switch into failure
// modes at runtime. A real provider (Midtrans, Xendit, Stripe) would replace it.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Mode string

const (
	ModeSucceed Mode = "succeed" // every charge is approved
	ModeDecline Mode = "decline" // a final answer: the card is declined
	ModeDown    Mode = "down"    // an outage: no answer at all, callers must retry
)

var ErrUnavailable = errors.New("payment provider unavailable")

type Result struct {
	Approved bool
	Reason   string
}

// Fake remembers the answer for each idempotency key, like real providers do:
// charging the same key twice returns the first answer and takes no new money.
// The memory is unbounded, which is fine for a fake.
type Fake struct {
	mu      sync.Mutex
	mode    Mode
	charges map[string]Result
}

func NewFake() *Fake { return &Fake{mode: ModeSucceed, charges: map[string]Result{}} }

func (f *Fake) Mode() Mode {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mode
}

func (f *Fake) SetMode(m Mode) error {
	switch m {
	case ModeSucceed, ModeDecline, ModeDown:
		f.mu.Lock()
		f.mode = m
		f.mu.Unlock()
		return nil
	}
	return fmt.Errorf("unknown mode %q", m)
}

func (f *Fake) Charge(_ context.Context, idempotencyKey string, amountIDR int64) (Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.charges[idempotencyKey]; ok {
		return r, nil
	}
	var r Result
	switch f.mode {
	case ModeDown:
		return Result{}, ErrUnavailable // not remembered: nothing happened
	case ModeDecline:
		r = Result{Approved: false, Reason: "card declined"}
	default:
		r = Result{Approved: true}
	}
	f.charges[idempotencyKey] = r
	return r, nil
}
