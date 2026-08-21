package listings

import (
	"context"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"go.uber.org/zap"
)

const (
	KindHouse     = "house"
	KindBusiness  = "business"
	KindWarehouse = "warehouse"
)

type Notifier interface {
	Enabled() bool
	NotifyListing(context.Context, discord.ListingNotice) error
}

type Service struct {
	store    *review.Store
	notifier Notifier
	log      *zap.Logger
	now      func() time.Time
}

func NewService(store *review.Store, notifier Notifier, log *zap.Logger) *Service {
	return &Service{store: store, notifier: notifier, log: log, now: time.Now}
}

func (s *Service) Process(ctx context.Context, kind string, items []review.Item) ([]review.Record, error) {
	items = s.filterItems(kind, items)
	now := s.now().UTC()
	visible, unnotified, err := s.store.Reconcile(kind, items, now)
	if err != nil {
		return nil, err
	}
	if !s.notifier.Enabled() {
		return visible, nil
	}
	for _, record := range unnotified {
		notice := discord.ListingNotice{
			Kind: kindLabel(record.Kind), Key: record.Key, Name: record.Name,
			Value: record.Value, IconURL: record.Icon,
		}
		if err := s.notifier.NotifyListing(ctx, notice); err != nil {
			s.log.Warn("Discord listing notification failed", zap.String("kind", record.Kind), zap.String("key", record.Key), zap.Error(err))
			continue
		}
		if err := s.store.MarkNotified(record.Kind, record.Key, s.now().UTC()); err != nil {
			s.log.Warn("could not persist Discord notification state", zap.String("kind", record.Kind), zap.String("key", record.Key), zap.Error(err))
		}
	}
	return visible, nil
}

func (s *Service) SetStatus(kind, key string, status review.Status) (review.Record, error) {
	return s.store.SetStatus(kind, key, status, s.now().UTC())
}

func (s *Service) Available() []review.Record {
	records := s.store.Available()
	filtered := records[:0]
	for _, record := range records {
		if record.Value >= s.store.MinimumValue(record.Kind) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func (s *Service) filterItems(kind string, items []review.Item) []review.Item {
	minimum := s.store.MinimumValue(kind)
	filtered := make([]review.Item, 0, len(items))
	for _, item := range items {
		if item.Value >= minimum {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func kindLabel(kind string) string {
	switch kind {
	case KindBusiness:
		return "Negocio"
	case KindWarehouse:
		return "Almacén"
	default:
		return "Propiedad"
	}
}
