package presence

import (
	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"presence",
	fx.Provide(
		func(client *sarp.APIClient) APIQuerier { return client },
		func(source *Source) OnlineSource { return source },
		func(client *discord.Client) Notifier { return client },
		NewSource,
		NewService,
		NewAccountService,
		NewScheduler,
	),
	fx.Invoke(RegisterScheduler),
)
