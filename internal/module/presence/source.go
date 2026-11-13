package presence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

const onlinePath = "/public/global-metrics/online"

var ErrMalformedOnlineList = errors.New("online players response has no payload list")

type APIQuerier interface {
	QueryProvider(context.Context, string, string, url.Values) ([]byte, error)
}

type Source struct{ client APIQuerier }

func NewSource(client APIQuerier) *Source { return &Source{client: client} }

// Online returns the characters currently connected according to SARP.
func (s *Source) Online(ctx context.Context) ([]Character, error) {
	body, err := s.client.QueryProvider(ctx, "sarp", onlinePath, nil)
	if err != nil {
		return nil, fmt.Errorf("query online players: %w", err)
	}
	var response onlineResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode online players: %w", err)
	}
	if response.Payload == nil {
		return nil, ErrMalformedOnlineList
	}
	return response.Payload, nil
}
