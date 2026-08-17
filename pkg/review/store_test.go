package review

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestStorePersistsReviewAndIgnoreState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	store, err := NewStore(Config{StateFile: path}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	now := time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC)
	visible, unnotified, err := store.Reconcile("house", []Item{
		{Key: "1", Name: "One", Value: 100}, {Key: "2", Name: "Two", Value: 200},
	}, now)
	if err != nil || len(visible) != 2 || len(unnotified) != 2 {
		t.Fatalf("Reconcile() = %#v, %#v, %v", visible, unnotified, err)
	}
	if _, err := store.SetStatus("house", "1", StatusIgnored, now); err != nil {
		t.Fatalf("ignore error = %v", err)
	}
	if _, err := store.SetStatus("house", "2", StatusReviewed, now); err != nil {
		t.Fatalf("review error = %v", err)
	}
	if err := store.MarkNotified("house", "2", now); err != nil {
		t.Fatalf("MarkNotified() error = %v", err)
	}

	reloaded, err := NewStore(Config{StateFile: path}, zap.NewNop())
	if err != nil {
		t.Fatalf("reload error = %v", err)
	}
	visible, unnotified, err = reloaded.Reconcile("house", []Item{
		{Key: "1", Name: "One", Value: 100}, {Key: "2", Name: "Two", Value: 200},
	}, now.Add(time.Hour))
	if err != nil || len(visible) != 1 || visible[0].Status != StatusReviewed || len(unnotified) != 0 {
		t.Fatalf("reloaded Reconcile() = %#v, %#v, %v", visible, unnotified, err)
	}
}

func TestStoreRejectsUnknownListing(t *testing.T) {
	store, err := NewStore(Config{StateFile: filepath.Join(t.TempDir(), "state.json")}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.SetStatus("business", "missing", StatusIgnored, time.Now()); !errors.Is(err, ErrListingNotFound) {
		t.Fatalf("SetStatus() error = %v", err)
	}
}
