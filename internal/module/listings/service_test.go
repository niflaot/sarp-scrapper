package listings

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"go.uber.org/zap"
)

type notifierStub struct {
	enabled bool
	notices []discord.ListingNotice
	err     error
}

func (n *notifierStub) Enabled() bool { return n.enabled }
func (n *notifierStub) NotifyListing(_ context.Context, notice discord.ListingNotice) error {
	n.notices = append(n.notices, notice)
	return n.err
}

func TestServiceProcessesAndNotifiesNewListingsOnce(t *testing.T) {
	store, err := review.NewStore(review.Config{StateFile: filepath.Join(t.TempDir(), "state.json")}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	notifier := &notifierStub{enabled: true}
	service := NewService(store, notifier, zap.NewNop())
	service.now = func() time.Time { return time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC) }
	items := []review.Item{{Key: "7", Name: "24/7", Value: 5000}}
	for attempt := 0; attempt < 2; attempt++ {
		visible, err := service.Process(context.Background(), KindBusiness, items)
		if err != nil || len(visible) != 1 {
			t.Fatalf("Process() = %#v, %v", visible, err)
		}
	}
	if len(notifier.notices) != 1 || notifier.notices[0].Kind != "Negocio" {
		t.Fatalf("notices = %#v", notifier.notices)
	}
}

func TestServiceFiltersListingsBelowConfiguredMinimums(t *testing.T) {
	store, err := review.NewStore(review.Config{
		StateFile: filepath.Join(t.TempDir(), "state.json"), HouseMinValue: 500000, BusinessMinValue: 100000,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	notifier := &notifierStub{enabled: true}
	service := NewService(store, notifier, zap.NewNop())

	if _, err := service.Process(context.Background(), KindHouse, []review.Item{
		{Key: "low-house", Name: "Low", Value: 499999}, {Key: "house", Name: "House", Value: 500000},
	}); err != nil {
		t.Fatalf("Process(house) error = %v", err)
	}
	if _, err := service.Process(context.Background(), KindBusiness, []review.Item{
		{Key: "low-business", Name: "Low", Value: 99999}, {Key: "business", Name: "Business", Value: 100000},
	}); err != nil {
		t.Fatalf("Process(business) error = %v", err)
	}

	visible := service.Available()
	if len(visible) != 2 || len(notifier.notices) != 2 {
		t.Fatalf("visible = %#v, notices = %#v", visible, notifier.notices)
	}
	for _, record := range visible {
		if record.Key == "low-house" || record.Key == "low-business" {
			t.Fatalf("below-threshold record remained visible: %#v", record)
		}
	}
}
