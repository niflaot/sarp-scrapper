package houses

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

var ErrRefreshInProgress = errors.New("house refresh already in progress")

type Source interface {
	FetchMap(context.Context) ([]byte, error)
}

type Service struct {
	source     Source
	repository Repository
	log        *zap.Logger
	now        func() time.Time
	refreshMu  sync.Mutex
}

func NewService(source Source, repository Repository, log *zap.Logger) *Service {
	return &Service{source: source, repository: repository, log: log, now: time.Now}
}

func (s *Service) Refresh(ctx context.Context) error {
	if !s.refreshMu.TryLock() {
		return ErrRefreshInProgress
	}
	defer s.refreshMu.Unlock()

	attemptedAt := s.now().UTC()
	s.repository.RecordAttempt(attemptedAt)
	body, err := s.source.FetchMap(ctx)
	if err != nil {
		err = fmt.Errorf("fetch available houses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	houses, err := ParseAvailableHouses(body)
	if err != nil {
		err = fmt.Errorf("parse available houses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	s.repository.Replace(houses, s.now().UTC())
	s.log.Info("available houses refreshed", zap.Int("count", len(houses)))
	return nil
}

func (s *Service) Snapshot() Snapshot { return s.repository.Snapshot() }

func (s *Service) SetNextRun(at *time.Time) { s.repository.SetNextRun(at) }
