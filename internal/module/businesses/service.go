package businesses

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

var ErrRefreshInProgress = errors.New("business refresh already in progress")

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
	body, err := s.source.FetchBusinesses(ctx)
	if err != nil {
		err = fmt.Errorf("fetch available businesses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	businesses, err := ParseAvailableBusinesses(body)
	if err != nil {
		err = fmt.Errorf("parse available businesses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	items := make([]review.Item, 0, len(businesses))
	for _, business := range businesses {
		items = append(items, review.Item{Key: business.ID, Name: business.Name, Value: business.Value, Icon: business.Icon})
	}
	visible, err := s.listings.Process(ctx, listings.KindBusiness, items)
	if err != nil {
		err = fmt.Errorf("persist available businesses: %w", err)
		s.repository.RecordFailure(err)
		return err
	}
	allowed := make(map[string]struct{}, len(visible))
	for _, record := range visible {
		allowed[record.Key] = struct{}{}
	}
	filtered := businesses[:0]
	for _, business := range businesses {
		if _, ok := allowed[business.ID]; ok {
			filtered = append(filtered, business)
		}
	}
	s.repository.Replace(filtered, s.now().UTC())
	s.log.Info("available businesses refreshed", zap.Int("count", len(filtered)))
	return nil
}

func (s *Service) Snapshot() Snapshot {
	snapshot := s.repository.Snapshot()
	allowed := make(map[string]struct{})
	for _, record := range s.listings.Available() {
		if record.Kind == listings.KindBusiness {
			allowed[record.Key] = struct{}{}
		}
	}
	filtered := snapshot.Businesses[:0]
	for _, business := range snapshot.Businesses {
		if _, ok := allowed[business.ID]; ok {
			filtered = append(filtered, business)
		}
	}
	snapshot.Businesses = filtered
	snapshot.Count = len(filtered)
	return snapshot
}

func (s *Service) SetNextRun(at *time.Time) { s.repository.SetNextRun(at) }
