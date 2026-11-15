package presence

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

// tracker notifies when a watched entry shows up in a list it was absent from
// in the previous successful check. It keeps its state in memory and must be
// driven by a single goroutine.
//
// The first successful check only records what is already present. A failed
// fetch keeps the previous state, and a failed notification is retried on the
// next check.
type tracker[T any] struct {
	kind   string
	fetch  func(context.Context) ([]T, error)
	key    func(T) (key string, watched bool)
	notify func(context.Context, T) error
	log    *zap.Logger

	present map[string]struct{}
	primed  bool
}

func (t *tracker[T]) check(ctx context.Context) error {
	items, err := t.fetch(ctx)
	if err != nil {
		return err
	}
	current := make(map[string]T)
	for _, item := range items {
		if key, watched := t.key(item); watched {
			current[key] = item
		}
	}
	if !t.primed {
		t.present = make(map[string]struct{}, len(current))
		for key := range current {
			t.present[key] = struct{}{}
		}
		t.primed = true
		t.log.Info("presence baseline recorded", zap.String("kind", t.kind), zap.Int("watched_present", len(current)))
		return nil
	}

	next := make(map[string]struct{}, len(current))
	var failures []error
	for key, item := range current {
		if _, was := t.present[key]; was {
			next[key] = struct{}{}
			continue
		}
		if err := t.notify(ctx, item); err != nil {
			t.log.Warn("presence notification failed; it will be retried", zap.String("kind", t.kind), zap.String("key", key), zap.Error(err))
			failures = append(failures, err)
			continue
		}
		t.log.Info("watched entry appeared online", zap.String("kind", t.kind), zap.String("key", key))
		next[key] = struct{}{}
	}
	t.present = next
	return errors.Join(failures...)
}
