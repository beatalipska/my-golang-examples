package task

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

var ErrNoWork = errors.New("no due events")

type WorkerDeps struct {
	Repo      WorkerRepository
	Processor Processor
	Backoff   BackoffConfig
	RNG       *rand.Rand
	Now       func() time.Time
}

func ProcessOnce(ctx context.Context, deps WorkerDeps) (bool, error) {
	if deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	if deps.Backoff.BaseDelay == 0 && deps.Backoff.MaxDelay == 0 {
		deps.Backoff = DefaultBackoff()
	}

	ev, ok, err := deps.Repo.ClaimNextDue(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, ErrNoWork
	}

	if err := deps.Processor.Process(ctx, ev.ID, ev.Type, ev.Payload); err != nil {
		next := NextRetryAt(deps.Now(), ev.Attempts, deps.Backoff, deps.RNG)
		_ = deps.Repo.MarkFailed(ctx, ev.ID, err.Error(), next)
		return true, err
	}

	if err := deps.Repo.MarkProcessed(ctx, ev.ID); err != nil {
		return true, err
	}
	return true, nil
}
