package presence

import (
	"context"
	"sync"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/presence"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type job struct {
	name     string
	interval time.Duration
	check    func(context.Context) error
}

// Scheduler runs one polling loop per enabled watcher.
type Scheduler struct {
	jobs     []job
	notifier Notifier
	log      *zap.Logger

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewScheduler(config presence.Config, characters *Service, accounts *AccountService, notifier Notifier, log *zap.Logger) *Scheduler {
	scheduler := &Scheduler{notifier: notifier, log: log}
	if config.Enabled() {
		scheduler.jobs = append(scheduler.jobs, job{name: "characters", interval: config.PollInterval, check: characters.Check})
	}
	if config.AccountsEnabled() {
		scheduler.jobs = append(scheduler.jobs, job{name: "accounts", interval: config.AccountPollInterval, check: accounts.Check})
	}
	return scheduler
}

func RegisterScheduler(lifecycle fx.Lifecycle, scheduler *Scheduler) {
	lifecycle.Append(fx.Hook{OnStart: scheduler.start, OnStop: scheduler.stop})
}

func (s *Scheduler) start(context.Context) error {
	if len(s.jobs) == 0 {
		s.log.Info("presence watcher disabled: PRESENCE_WATCH_NAMES and PRESENCE_WATCH_ACCOUNT_IDS are empty")
		return nil
	}
	if !s.notifier.PresenceEnabled() {
		s.log.Warn("presence watcher disabled: set DISCORD_BOT_TOKEN and/or DISCORD_WEBHOOK_URL")
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	for _, j := range s.jobs {
		s.log.Info("presence watcher started", zap.String("watcher", j.name), zap.Duration("interval", j.interval))
		s.wg.Add(1)
		go func(j job) { defer s.wg.Done(); s.run(ctx, j) }(j)
	}
	return nil
}

func (s *Scheduler) stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) run(ctx context.Context, j job) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		if err := j.check(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("presence check failed", zap.String("watcher", j.name), zap.Error(err))
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}
