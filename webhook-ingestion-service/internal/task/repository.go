package task

import (
	"context"
	"encoding/json"

	"webhook-ingestion-service/internal/model"
)

type EventRepository interface {
	InsertReceived(ctx context.Context, id string, eventType string, payload json.RawMessage) (bool, error)
	GetByID(ctx context.Context, id string) (model.Event, error)
}
