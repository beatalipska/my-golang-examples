package model

import "time"

type EventStatus string

const (
	StatusReceived   EventStatus = "received"
	StatusProcessing EventStatus = "processing"
	StatusProcessed  EventStatus = "processed"
	StatusFailed     EventStatus = "failed"
)

type Event struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Payload     []byte      `json:"-"` // raw JSON bytes; not returned directly
	Status      EventStatus `json:"status"`
	Attempts    int         `json:"attempts"`
	NextRetryAt time.Time   `json:"next_retry_at"`
	LastError   *string     `json:"last_error,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	ProcessedAt *time.Time  `json:"processed_at,omitempty"`
}
