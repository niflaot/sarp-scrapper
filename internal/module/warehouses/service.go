package warehouses

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

var ErrRefreshInProgress = errors.New("warehouse refresh already in progress")

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
	s.repository.RecordAttempt(s.now().UTC())
	body, err := s.source.FetchMap(ctx)
	if err != nil {
		err = fmt.Errorf("fetch available warehouses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	warehouses, err := ParseAvailableWarehouses(body)
	if err != nil {
		err = fmt.Errorf("parse available warehouses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	items := make([]review.Item, 0, len(warehouses))
	for _, warehouse := range warehouses {
		items = append(items, review.Item{Key: warehouse.ID, Name: warehouse.Address, Value: warehouse.Value, Icon: warehouse.Icon})
	}
	visible, err := s.listings.Process(ctx, listings.KindWarehouse, items)
	if err != nil {
		err = fmt.Errorf("persist available warehouses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	allowed := make(map[string]struct{}, len(visible))
	for _, record := range visible {
		allowed[record.Key] = struct{}{}
	}
	filtered := warehouses[:0]
	for _, warehouse := range warehouses {
		if _, ok := allowed[warehouse.ID]; ok {
			filtered = append(filtered, warehouse)
		}
	}
	s.repository.Replace(filtered, s.now().UTC())
	s.log.Info("available warehouses refreshed", zap.Int("count", len(filtered)))
	return nil
}

func (s *Service) Snapshot() Snapshot {
	snapshot := s.repository.Snapshot()
	allowed := make(map[string]struct{})
	for _, record := range s.listings.Available() {
		if record.Kind == listings.KindWarehouse {
			allowed[record.Key] = struct{}{}
		}
	}
	filtered := snapshot.Warehouses[:0]
	for _, warehouse := range snapshot.Warehouses {
		if _, ok := allowed[warehouse.ID]; ok {
			filtered = append(filtered, warehouse)
		}
	}
	snapshot.Warehouses = filtered
	snapshot.Count = len(filtered)
	return snapshot
}

func (s *Service) SetNextRun(at *time.Time) { s.repository.SetNextRun(at) }
