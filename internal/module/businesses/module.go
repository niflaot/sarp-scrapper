package businesses

import "go.uber.org/fx"

var Module = fx.Module(
	"businesses",
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
