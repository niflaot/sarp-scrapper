package presence

import (
	"context"
	"sync"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/presence"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Scheduler struct {
	config   presence.Config
	service  *Service
	notifier Notifier
	log      *zap.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewScheduler(config presence.Config, service *Service, notifier Notifier, log *zap.Logger) *Scheduler {
	return &Scheduler{config: config, service: service, notifier: notifier, log: log}
}

func RegisterScheduler(lifecycle fx.Lifecycle, scheduler *Scheduler) {
	lifecycle.Append(fx.Hook{OnStart: scheduler.start, OnStop: scheduler.stop})
}

func (s *Scheduler) start(context.Context) error {
	if !s.config.Enabled() {
		s.log.Info("presence watcher disabled: PRESENCE_WATCH_NAMES is empty")
		return nil
	}
	if !s.notifier.PresenceEnabled() {
		s.log.Warn("presence watcher disabled: set DISCORD_BOT_TOKEN and/or DISCORD_WEBHOOK_URL")
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.mu.Lock()
	s.cancel, s.done = cancel, done
	s.mu.Unlock()
	s.log.Info("presence watcher started", zap.Strings("characters", s.config.Names()), zap.Duration("interval", s.config.PollInterval))
	go func() { defer close(done); s.run(ctx) }()
	return nil
}

func (s *Scheduler) stop(ctx context.Context) error {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) run(ctx context.Context) {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()
	for {
		if err := s.service.Check(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("presence check failed", zap.Error(err))
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
