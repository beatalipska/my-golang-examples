package task

import (
	"context"
	"encoding/json"
)

type Processor interface {
	Process(ctx context.Context, eventID string, eventType string, payload json.RawMessage) error
}
