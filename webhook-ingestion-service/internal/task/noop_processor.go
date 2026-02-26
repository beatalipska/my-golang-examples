package task

import (
	"context"
	"encoding/json"
)

type NoopProcessor struct{}

func (NoopProcessor) Process(ctx context.Context, eventID, eventType string, payload json.RawMessage) error {
	// In real life: call downstream service, update payment table, publish to Kafka/SNS, etc.
	return nil
}
