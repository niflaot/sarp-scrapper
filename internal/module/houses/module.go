package houses

import "go.uber.org/fx"

var Module = fx.Module(
	"houses",
	fx.Provide(
		NewSource,
		func(repository *MemoryRepository) Repository { return repository },
		NewMemoryRepository,
		NewService,
		NewHandler,
		NewScheduler,
	),
	fx.Invoke(Register, RegisterScheduler),
)
