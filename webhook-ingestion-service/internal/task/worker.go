package task

import (
	"context"
	"errors"
	"log"
	"time"
)

type WorkerConfig struct {
	Interval  time.Duration // base polling interval (e.g. 200ms, 1s)
	Burst     int           // max events per tick (e.g. 5)
	IdleDelay time.Duration // sleep when no work (e.g. 500ms)
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		Interval:  500 * time.Millisecond,
		Burst:     5,
		IdleDelay: 800 * time.Millisecond,
	}
}

// RunWorker runs until ctx is canceled.
// It periodically calls ProcessOnce and supports burst processing.
func RunWorker(ctx context.Context, deps WorkerDeps, cfg WorkerConfig, logger *log.Logger) {
	if cfg.Interval <= 0 {
		cfg.Interval = 500 * time.Millisecond
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 1
	}
	if cfg.IdleDelay <= 0 {
		cfg.IdleDelay = 800 * time.Millisecond
	}
	if logger == nil {
		logger = log.Default()
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	logger.Printf("worker started: interval=%s burst=%d", cfg.Interval, cfg.Burst)

	for {
		select {
		case <-ctx.Done():
			logger.Printf("worker stopping: %v", ctx.Err())
			return

		case <-ticker.C:
			// burst: try up to cfg.Burst items per tick
			processedAny := false

			for i := 0; i < cfg.Burst; i++ {
				claimed, err := ProcessOnce(ctx, deps)

				if err != nil {
					if errors.Is(err, ErrNoWork) {
						// no more work right now
						break
					}
					// processing error (already marked failed inside ProcessOnce)
					logger.Printf("worker: process_once error: %v", err)
					// continue trying next item (or break; both are acceptable)
					continue
				}

				if claimed {
					processedAny = true
				}
			}

			// If nothing was processed, sleep a bit to avoid tight polling
			if !processedAny {
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.IdleDelay):
				}
			}
		}
	}
}
