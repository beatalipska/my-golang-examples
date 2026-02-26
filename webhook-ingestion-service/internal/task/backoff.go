package task

import (
	"math/rand"
	"time"
)

type BackoffConfig struct {
	BaseDelay time.Duration // e.g. 1s
	MaxDelay  time.Duration // e.g. 60s
}

func DefaultBackoff() BackoffConfig {
	return BackoffConfig{
		BaseDelay: 1 * time.Second,
		MaxDelay:  60 * time.Second,
	}
}

// NextRetryAt computes the next retry time using exponential backoff with full jitter.
// attempt is 1-based (1 => BaseDelay).
func NextRetryAt(now time.Time, attempt int, cfg BackoffConfig, rng *rand.Rand) time.Time {
	if attempt < 1 {
		attempt = 1
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 1 * time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 60 * time.Second
	}

	// exponential: base * 2^(attempt-1)
	delay := cfg.BaseDelay << (attempt - 1)

	// cap
	if delay > cfg.MaxDelay {
		delay = cfg.MaxDelay
	}

	// full jitter: random in [0, delay]
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	jitter := time.Duration(rng.Int63n(int64(delay) + 1))

	return now.Add(jitter).UTC()
}
