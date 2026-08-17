package logger

import (
	"context"
	"errors"
	"os"
	"syscall"

	"go.uber.org/fx"
	uberzap "go.uber.org/zap"
)

var Module = fx.Module("logger", fx.Provide(New), fx.Invoke(registerLifecycle))

func registerLifecycle(lifecycle fx.Lifecycle, log *uberzap.Logger) {
	lifecycle.Append(fx.Hook{OnStop: func(context.Context) error {
		err := log.Sync()
		if errors.Is(err, os.ErrInvalid) || errors.Is(err, syscall.ENOTTY) || errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EBADF) {
			return nil
		}
		return err
	}})
}
