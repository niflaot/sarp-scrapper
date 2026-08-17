package discord

import "go.uber.org/fx"

var Module = fx.Module("discord", fx.Provide(NewClient))
