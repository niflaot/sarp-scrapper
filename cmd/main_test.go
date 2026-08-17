package main

import (
	"testing"

	"github.com/niflaot/sarp-scrapper/internal/module/auth"
	"github.com/niflaot/sarp-scrapper/internal/module/businesses"
	"github.com/niflaot/sarp-scrapper/internal/module/houses"
	"github.com/niflaot/sarp-scrapper/internal/module/listings"
	"github.com/niflaot/sarp-scrapper/internal/module/status"
	"github.com/niflaot/sarp-scrapper/pkg/config"
	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/logger"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestApplicationGraph(t *testing.T) {
	t.Setenv("SARP_SCRAPER_ENABLED", "false")
	if err := fx.ValidateApp(config.Module, logger.Module, discord.Module, review.Module, server.Module, sarp.Module, listings.Module, auth.Module, houses.Module, businesses.Module, status.Module); err != nil {
		t.Fatalf("ValidateApp() error = %v", err)
	}
}

func TestFxEventsUseDebugLevel(t *testing.T) {
	core, observed := observer.New(zap.DebugLevel)
	logger := newFxEventLogger(zap.New(core))
	logger.LogEvent(&fxevent.Started{})
	entries := observed.All()
	if len(entries) != 1 || entries[0].Level != zap.DebugLevel {
		t.Fatalf("unexpected Fx logs: %#v", entries)
	}
}
