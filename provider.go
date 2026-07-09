package onceo

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
)

type Provider[T any] interface {
	Name() string
	VerifySignature(headers http.Header, body []byte) error
	Parse(body []byte) (T, error)
	Normalize(parsed T) (Event, error)
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
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

	event.ID = generateID()
	event.RawPayload = body

	created, err := store.SaveIfNew(ctx, event)
	if err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}
	if !created {
		return Event{}, WrapProviderError(p.Name(), ErrDuplicateEvent)
	}

	return event, nil
}
