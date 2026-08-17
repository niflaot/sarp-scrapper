package houses

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/niflaot/sarp-scrapper/internal/module/listings"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"go.uber.org/zap"
)

var ErrRefreshInProgress = errors.New("house refresh already in progress")

type Source interface {
	FetchMap(context.Context) ([]byte, error)
}

type Service struct {
	source     Source
	repository Repository
	listings   *listings.Service
	log        *zap.Logger
	now        func() time.Time
	refreshMu  sync.Mutex
}

func NewService(source Source, repository Repository, listingService *listings.Service, log *zap.Logger) *Service {
	return &Service{source: source, repository: repository, listings: listingService, log: log, now: time.Now}
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
	if s.listings != nil {
		items := make([]review.Item, 0, len(houses))
		for _, house := range houses {
			name := house.Address
			if name == "" {
				name = "Propiedad #" + house.Key
			}
			items = append(items, review.Item{Key: house.Key, Name: name, Value: house.Price, Icon: house.Icon})
		}
		visible, processErr := s.listings.Process(ctx, listings.KindHouse, items)
		if processErr != nil {
			err = fmt.Errorf("persist available houses: %w", processErr)
			s.repository.RecordFailure(err)
			return err
		}
		allowed := make(map[string]struct{}, len(visible))
		for _, record := range visible {
			allowed[record.Key] = struct{}{}
		}
		filtered := houses[:0]
		for _, house := range houses {
			if _, ok := allowed[house.Key]; ok {
				filtered = append(filtered, house)
			}
		}
		houses = filtered
	}
	s.repository.Replace(houses, s.now().UTC())
	s.log.Info("available houses refreshed", zap.Int("count", len(houses)))
	return nil
}

func (s *Service) Snapshot() Snapshot {
	snapshot := s.repository.Snapshot()
	if s.listings == nil {
		return snapshot
	}
	allowed := make(map[string]struct{})
	for _, record := range s.listings.Available() {
		if record.Kind == listings.KindHouse {
			allowed[record.Key] = struct{}{}
		}
	}
	filtered := snapshot.Houses[:0]
	for _, house := range snapshot.Houses {
		if _, ok := allowed[house.Key]; ok {
			filtered = append(filtered, house)
		}
	}
	snapshot.Houses = filtered
	snapshot.Count = len(filtered)
	return snapshot
}

func (s *Service) SetNextRun(at *time.Time) { s.repository.SetNextRun(at) }
