package apiquery

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

var ErrInvalidRequest = errors.New("API query request is invalid")

const (
	maxQueryKeys      = 100
	maxValuesPerKey   = 20
	maxQueryComponent = 4096
)

type Client interface {
	QueryProvider(context.Context, string, string, url.Values) ([]byte, error)
}

type Service struct{ client Client }

func NewService(client Client) *Service { return &Service{client: client} }

func (s *Service) Query(ctx context.Context, request Request) ([]byte, error) {
	if request.Path == "" {
		return nil, fmt.Errorf("%w: path is required", ErrInvalidRequest)
	}
	if len(request.Query) > maxQueryKeys {
		return nil, fmt.Errorf("%w: query exceeds %d keys", ErrInvalidRequest, maxQueryKeys)
	}
	values := make(url.Values, len(request.Query))
	for key, entries := range request.Query {
		if key == "" || len(key) > maxQueryComponent {
			return nil, fmt.Errorf("%w: query contains an invalid key", ErrInvalidRequest)
		}
		if len(entries) > maxValuesPerKey {
			return nil, fmt.Errorf("%w: query key %q exceeds %d values", ErrInvalidRequest, key, maxValuesPerKey)
		}
		for _, value := range entries {
			if len(value) > maxQueryComponent {
				return nil, fmt.Errorf("%w: query value for %q is too long", ErrInvalidRequest, key)
			}
			values.Add(key, value)
		}
	}
	return s.client.QueryProvider(ctx, request.Provider, request.Path, values)
}
