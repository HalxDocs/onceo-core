package onceo

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Provider[T any] interface {
	Name() string
	VerifySignature(headers http.Header, body []byte) error
	Parse(body []byte) (T, error)
	Normalize(parsed T) (Event, error)
}

func generateID(r io.Reader) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("generating event id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func Process[T any](ctx context.Context, p Provider[T], store Store, headers http.Header, body []byte) (Event, error) {
	if len(body) > MaxBodySize {
		return Event{}, WrapProviderError(p.Name(), fmt.Errorf("body exceeds maximum size of %d bytes", MaxBodySize))
	}

	if err := p.VerifySignature(headers, body); err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}

	parsed, err := p.Parse(body)
	if err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}

	event, err := p.Normalize(parsed)
	if err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}

	if event.ProviderEventID == "" {
		return Event{}, WrapProviderError(p.Name(), ErrEventParseFailed)
	}

	id, err := generateID(rand.Reader)
	if err != nil {
		return Event{}, fmt.Errorf("onceo: %w", err)
	}
	event.ID = id
	event.Provider = p.Name()
	event.RawPayload = body
	event.ReceivedAt = time.Now().UTC()

	created, err := store.SaveIfNew(ctx, event)
	if err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}
	if !created {
		return Event{}, WrapProviderError(p.Name(), ErrDuplicateEvent)
	}

	return event, nil
}
