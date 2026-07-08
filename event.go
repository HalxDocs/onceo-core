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
	ID              string
	Provider        string
	ProviderEventID string
	Type            string
	Status          Status
	AmountMinor     int64
	Currency        string
	Reference       string
	RawPayload      []byte
	ReceivedAt      time.Time
	ProcessedAt     time.Time
}
