package provider

import (
	"context"
	"errors"
	"testing"
)

func TestModes(t *testing.T) {
	f := NewFake()
	ctx := context.Background()
	if r, err := f.Charge(ctx, "a", 1000); err != nil || !r.Approved {
		t.Fatalf("default mode should approve: %+v %v", r, err)
	}
	_ = f.SetMode(ModeDecline)
	if r, err := f.Charge(ctx, "b", 1000); err != nil || r.Approved || r.Reason != "card declined" {
		t.Fatalf("decline mode: %+v %v", r, err)
	}
	_ = f.SetMode(ModeDown)
	if _, err := f.Charge(ctx, "c", 1000); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("down mode should be an outage, got %v", err)
	}
	if err := f.SetMode("explode"); err == nil || f.Mode() != ModeDown {
		t.Fatal("an unknown mode must be rejected and change nothing")
	}
}

func TestSameKeyGetsTheSameAnswer(t *testing.T) {
	f := NewFake()
	ctx := context.Background()
	_ = f.SetMode(ModeDecline)
	first, _ := f.Charge(ctx, "order-1", 1000)
	_ = f.SetMode(ModeSucceed)
	again, _ := f.Charge(ctx, "order-1", 1000)
	if first != again || again.Approved {
		t.Fatalf("an idempotent retry must return the first answer, got %+v then %+v", first, again)
	}
	if r, _ := f.Charge(ctx, "order-2", 1000); !r.Approved {
		t.Fatal("a new key is charged under the new mode")
	}
}

func TestAnOutageIsNotRemembered(t *testing.T) {
	f := NewFake()
	ctx := context.Background()
	_ = f.SetMode(ModeDown)
	_, _ = f.Charge(ctx, "order-1", 1000)
	_ = f.SetMode(ModeSucceed)
	if r, err := f.Charge(ctx, "order-1", 1000); err != nil || !r.Approved {
		t.Fatalf("after the outage the same key must be chargeable: %+v %v", r, err)
	}
}
