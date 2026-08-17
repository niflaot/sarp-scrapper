package listings

import (
	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"listings",
	fx.Provide(
		func(client *discord.Client) Notifier { return client },
		NewService,
		NewHandler,
	),
	fx.Invoke(Register),
)
