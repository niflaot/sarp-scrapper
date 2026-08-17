package review

import "go.uber.org/fx"

var Module = fx.Module("review", fx.Provide(NewStore))
