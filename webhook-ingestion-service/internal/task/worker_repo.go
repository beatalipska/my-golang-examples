package task

import (
	"context"
	"encoding/json"
	"time"
)

type ClaimedEvent struct {
	ID       string
	Type     string
	Payload  json.RawMessage
	Attempts int
}

type WorkerRepository interface {
	ClaimNextDue(ctx context.Context) (ClaimedEvent, bool, error)
	MarkProcessed(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, lastErr string, nextRetryAt time.Time) error
}
