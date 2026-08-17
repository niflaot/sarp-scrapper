package sarp

import (
	"github.com/niflaot/sarp-scrapper/pkg/session"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"sarp",
	fx.Provide(
		NewBrowser,
		NewAPIClient,
		NewStatusProvider,
		func(provider *StatusProvider) session.Provider { return provider },
	),
	fx.Invoke(RegisterLifecycle),
)
