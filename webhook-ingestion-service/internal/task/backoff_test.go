package task

import (
	"math/rand"
	"testing"
	"time"
)

func TestNextRetryAt_MonotonicBounds(t *testing.T) {
	cfg := BackoffConfig{BaseDelay: 1 * time.Second, MaxDelay: 60 * time.Second}
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	// deterministic RNG
	rng := rand.New(rand.NewSource(1))

	// attempt=1 => delay in [0..1s]
	t1 := NextRetryAt(now, 1, cfg, rng)
	if t1.Before(now) || t1.After(now.Add(1*time.Second)) {
		t.Fatalf("attempt 1 out of range: %s", t1.Sub(now))
	}

	// attempt=2 => [0..2s]
	rng = rand.New(rand.NewSource(1))
	t2 := NextRetryAt(now, 2, cfg, rng)
	if t2.Before(now) || t2.After(now.Add(2*time.Second)) {
		t.Fatalf("attempt 2 out of range: %s", t2.Sub(now))
	}

	// attempt=6 => base*32 => [0..32s]
	rng = rand.New(rand.NewSource(1))
	t6 := NextRetryAt(now, 6, cfg, rng)
	if t6.Before(now) || t6.After(now.Add(32*time.Second)) {
		t.Fatalf("attempt 6 out of range: %s", t6.Sub(now))
	}
}

func TestNextRetryAt_Capped(t *testing.T) {
	cfg := BackoffConfig{BaseDelay: 10 * time.Second, MaxDelay: 60 * time.Second}
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	// attempt=10 => 10s * 2^9 = 5120s, capped to 60s, so [0..60s]
	rng := rand.New(rand.NewSource(42))
	next := NextRetryAt(now, 10, cfg, rng)

	if next.Before(now) || next.After(now.Add(60*time.Second)) {
		t.Fatalf("capped out of range: %s", next.Sub(now))
	}
}

func TestNextRetryAt_AttemptLessThanOne(t *testing.T) {
	cfg := DefaultBackoff()
	now := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)

	rng := rand.New(rand.NewSource(7))
	next := NextRetryAt(now, 0, cfg, rng)

	if next.Before(now) || next.After(now.Add(cfg.BaseDelay)) {
		t.Fatalf("attempt 0 should behave like attempt 1: %s", next.Sub(now))
	}
}
