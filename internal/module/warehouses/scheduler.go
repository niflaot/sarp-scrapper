package warehouses

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Scheduler struct {
	config  sarp.Config
	service *Service
	log     *zap.Logger
	random  *rand.Rand
	now     func() time.Time
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
}

func NewScheduler(config sarp.Config, service *Service, log *zap.Logger) *Scheduler {
	return &Scheduler{config: config, service: service, log: log, random: rand.New(rand.NewSource(time.Now().UnixNano())), now: time.Now}
}

func RegisterScheduler(lifecycle fx.Lifecycle, scheduler *Scheduler) {
	lifecycle.Append(fx.Hook{OnStart: scheduler.start, OnStop: scheduler.stop})
}

func (s *Scheduler) start(context.Context) error {
	if !s.config.Enabled {
		s.log.Info("SARP warehouse scraper disabled")
		return nil
	}
	s.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel, s.done = cancel, make(chan struct{})
	done := s.done
	s.mu.Unlock()
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
	for {
		if err := s.service.Refresh(ctx); err != nil && ctx.Err() == nil {
			s.log.Warn("available warehouses refresh failed", zap.Error(err))
		}
		delay := s.nextDelay()
		nextRun := s.now().UTC().Add(delay)
		s.service.SetNextRun(&nextRun)
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			s.service.SetNextRun(nil)
			return
		}
	}
}

func (s *Scheduler) nextDelay() time.Duration {
	if s.config.Jitter == 0 {
		return s.config.Interval
	}
	span := int64(2*s.config.Jitter) + 1
	return s.config.Interval + time.Duration(s.random.Int63n(span)-int64(s.config.Jitter))
}
