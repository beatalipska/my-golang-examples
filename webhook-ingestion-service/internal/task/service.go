package task

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"webhook-ingestion-service/internal/model"
)

type Service struct {
	events EventRepository
}

func NewService(events EventRepository) *Service {
	return &Service{events: events}
}

var ErrInvalidEvent = errors.New("invalid event payload")

type IncomingEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func (s *Service) IngestWebhook(ctx context.Context, id string, rawBody []byte) (created bool, err error) {
	var in IncomingEvent
	if err := json.Unmarshal(rawBody, &in); err != nil {
		return false, ErrInvalidEvent
	}
	in.Type = strings.TrimSpace(in.Type)
	if in.Type == "" {
		return false, ErrInvalidEvent
	}

	// store full payload (rawBody) or structured payload; here: full body as JSONB
	return s.events.InsertReceived(ctx, id, in.Type, json.RawMessage(rawBody))
}

func (s *Service) GetEvent(ctx context.Context, id string) (model.Event, error) {
	return s.events.GetByID(ctx, id)
}
