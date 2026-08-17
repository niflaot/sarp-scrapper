package server

import (
	"context"

	"go.uber.org/fx"
)

var Module = fx.Module("server", fx.Provide(New), fx.Invoke(registerLifecycle))

func registerLifecycle(lifecycle fx.Lifecycle, server *Server) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error { return server.start() },
		OnStop:  server.stop,
	})
}
