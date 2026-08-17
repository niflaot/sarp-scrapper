package status

import "go.uber.org/fx"

var Module = fx.Module("status", fx.Invoke(Register))
