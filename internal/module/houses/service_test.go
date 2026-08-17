package houses

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

type sourceStub struct {
	body  []byte
	err   error
	wait  chan struct{}
	calls int
}

func (s *sourceStub) FetchMap(ctx context.Context) ([]byte, error) {
	s.calls++
	if s.wait != nil {
		select {
		case <-s.wait:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return s.body, s.err
}

func TestServiceRefresh(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(&sourceStub{body: []byte(`[{"key":749,"address":"Rodeo","price":100,"available":true}]`)}, repository, nil, zap.NewNop())
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	snapshot := service.Snapshot()
	if snapshot.Count != 1 || snapshot.LastAttempt == nil || snapshot.LastSuccess == nil {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
}

func TestServiceRefreshErrorsAndConcurrency(t *testing.T) {
	t.Run("source", func(t *testing.T) {
		service := NewService(&sourceStub{err: errors.New("offline")}, NewMemoryRepository(), nil, zap.NewNop())
		err := service.Refresh(context.Background())
		if err == nil || !strings.Contains(err.Error(), "fetch available houses") || service.Snapshot().LastError == "" {
			t.Fatalf("Refresh() error = %v", err)
		}
	})
	t.Run("parse", func(t *testing.T) {
		service := NewService(&sourceStub{body: []byte(`{}`)}, NewMemoryRepository(), nil, zap.NewNop())
		err := service.Refresh(context.Background())
		if err == nil || !strings.Contains(err.Error(), "parse available houses") {
			t.Fatalf("Refresh() error = %v", err)
		}
	})
	t.Run("already running", func(t *testing.T) {
		wait := make(chan struct{})
		service := NewService(&sourceStub{body: []byte(`[]`), wait: wait}, NewMemoryRepository(), nil, zap.NewNop())
		done := make(chan error, 1)
		go func() { done <- service.Refresh(context.Background()) }()
		deadline := time.Now().Add(time.Second)
		for service.Snapshot().LastAttempt == nil && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if err := service.Refresh(context.Background()); !errors.Is(err, ErrRefreshInProgress) {
			t.Fatalf("Refresh() error = %v", err)
		}
		close(wait)
		<-done
	})
}
