package main

import (
	"github.com/niflaot/sarp-scrapper/internal/module/auth"
	"github.com/niflaot/sarp-scrapper/internal/module/businesses"
	"github.com/niflaot/sarp-scrapper/internal/module/houses"
	"github.com/niflaot/sarp-scrapper/internal/module/listings"
	"github.com/niflaot/sarp-scrapper/internal/module/status"
	"github.com/niflaot/sarp-scrapper/internal/module/warehouses"
	"github.com/niflaot/sarp-scrapper/pkg/config"
	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/logger"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		config.Module,
		logger.Module,
		discord.Module,
		review.Module,
		server.Module,
		sarp.Module,
		listings.Module,
		auth.Module,
		houses.Module,
		businesses.Module,
		warehouses.Module,
		status.Module,
		fx.WithLogger(newFxEventLogger),
	).Run()
}

func newFxEventLogger(log *zap.Logger) fxevent.Logger {
	fxLog := &fxevent.ZapLogger{Logger: log}
	fxLog.UseLogLevel(zap.DebugLevel)
	fxLog.UseErrorLevel(zap.DebugLevel)
	return fxLog
}
