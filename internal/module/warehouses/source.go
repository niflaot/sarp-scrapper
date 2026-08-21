package warehouses

import (
	"context"
	"errors"
	"fmt"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"go.uber.org/zap"
)

type Source interface {
	FetchMap(context.Context) ([]byte, error)
}

type FallbackSource struct {
	api     Source
	browser Source
	log     *zap.Logger
}

func NewSource(config sarp.Config, api *sarp.APIClient, browser *sarp.Browser, log *zap.Logger) Source {
	if config.APIToken == "" {
		return browser
	}
	return &FallbackSource{api: api, browser: browser, log: log}
}

func (s *FallbackSource) FetchMap(ctx context.Context) ([]byte, error) {
	body, apiErr := s.api.FetchMap(ctx)
	if apiErr == nil {
		return body, nil
	}
	if ctx.Err() != nil {
		return nil, apiErr
	}
	s.log.Debug("SARP API source unavailable for warehouses; trying authenticated Chrome session", zap.Error(apiErr))
	body, browserErr := s.browser.FetchMap(ctx)
	if browserErr == nil {
		return body, nil
	}
	return nil, fmt.Errorf("SARP API and Chrome sources failed: %w", errors.Join(apiErr, browserErr))
}
