package onceo

import (
	"context"
	"net/http"
)

type Provider[T any] interface {
	Name() string
	VerifySignature(headers http.Header, body []byte) error
	Parse(body []byte) (T, error)
	Normalize(parsed T) (Event, error)
}

func Process[T any](ctx context.Context, p Provider[T], store Store, headers http.Header, body []byte) (Event, error) {
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

	seen, err := store.SeenBefore(ctx, event.Provider, event.ProviderEventID)
	if err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}
	if seen {
		return Event{}, WrapProviderError(p.Name(), ErrDuplicateEvent)
	}

	if err := store.Save(ctx, event); err != nil {
		return Event{}, WrapProviderError(p.Name(), err)
	}

	return event, nil
}
