package houses

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/sarp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type lifecycleRecorder struct{ hook fx.Hook }

func (r *lifecycleRecorder) Append(hook fx.Hook) { r.hook = hook }

func TestSchedulerNextDelay(t *testing.T) {
	scheduler := NewScheduler(sarp.Config{Interval: 30 * time.Minute, Jitter: 5 * time.Minute}, nil, zap.NewNop())
	scheduler.random = rand.New(rand.NewSource(42))
	seen := map[time.Duration]bool{}
	for range 1000 {
		delay := scheduler.nextDelay()
		if delay < 25*time.Minute || delay > 35*time.Minute {
			t.Fatalf("nextDelay() = %s", delay)
		}
		seen[delay] = true
	}
	if len(seen) < 2 {
		t.Fatal("nextDelay() did not vary")
	}

	scheduler.config.Jitter = 0
	if delay := scheduler.nextDelay(); delay != 30*time.Minute {
		t.Fatalf("nextDelay() = %s", delay)
	}
}

func TestSchedulerLifecycle(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(&sourceStub{body: []byte(`[{"key":749,"address":"Rodeo","price":100,"available":true}]`)}, repository, zap.NewNop())
	scheduler := NewScheduler(sarp.Config{Enabled: true, Interval: time.Hour}, service, zap.NewNop())
	recorder := &lifecycleRecorder{}
	RegisterScheduler(recorder, scheduler)
	if recorder.hook.OnStart == nil || recorder.hook.OnStop == nil {
		t.Fatal("scheduler lifecycle hooks were not registered")
	}
	if err := recorder.hook.OnStart(context.Background()); err != nil {
		t.Fatalf("OnStart() error = %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for repository.Snapshot().LastSuccess == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if repository.Snapshot().LastSuccess == nil {
		t.Fatal("scheduler did not run initial refresh")
	}
	stopContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := recorder.hook.OnStop(stopContext); err != nil {
		t.Fatalf("OnStop() error = %v", err)
	}
	if repository.Snapshot().NextRun != nil {
		t.Fatal("scheduler did not clear next run on stop")
	}
}

func TestDisabledSchedulerLifecycle(t *testing.T) {
	scheduler := NewScheduler(sarp.Config{}, nil, zap.NewNop())
	if err := scheduler.start(context.Background()); err != nil {
		t.Fatalf("start() error = %v", err)
	}
	if err := scheduler.stop(context.Background()); err != nil {
		t.Fatalf("stop() error = %v", err)
	}
}
