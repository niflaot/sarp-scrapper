package businesses

import (
	"context"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
)

type Source interface {
	FetchBusinesses(context.Context) ([]byte, error)
}

func NewSource(api *sarp.APIClient) Source { return api }
