package onceo

import (
	"fmt"
	"time"
)

type Status int

const (
	StatusPending Status = iota
	StatusSuccess
	StatusFailed
	StatusReversed
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	case StatusReversed:
		return "reversed"
	default:
		return "unknown"
	}
}

func StatusFromString(s string) (Status, error) {
	switch s {
	case "pending":
		return StatusPending, nil
	case "success":
		return StatusSuccess, nil
	case "failed":
		return StatusFailed, nil
	case "reversed":
		return StatusReversed, nil
	default:
		return StatusPending, fmt.Errorf("unknown status: %s", s)
	}
}

type Event struct {
	ID              string    `json:"id"`
	Provider        string    `json:"provider"`
	ProviderEventID string    `json:"provider_event_id"`
	Type            string    `json:"type"`
	Status          Status    `json:"status"`
	AmountMinor     int64     `json:"amount_minor"`
	Currency        string    `json:"currency"`
	Reference       string    `json:"reference"`
	RawPayload      []byte    `json:"raw_payload"`
	ReceivedAt      time.Time `json:"received_at"`
	ProcessedAt     time.Time `json:"processed_at"`
}
