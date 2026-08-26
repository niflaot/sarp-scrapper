package apiquery

import (
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"apiquery",
	fx.Provide(
		func(client *sarp.APIClient) Client { return client },
		NewService,
		NewHandler,
	),
	fx.Invoke(Register),
)
