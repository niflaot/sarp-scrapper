package businesses

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/niflaot/sarp-scrapper/internal/module/listings"
	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"go.uber.org/zap"
)

type sourceStub struct {
	body []byte
	err  error
}

func (s *sourceStub) FetchBusinesses(context.Context) ([]byte, error) { return s.body, s.err }

func TestServiceRefresh(t *testing.T) {
	store, err := review.NewStore(review.Config{StateFile: filepath.Join(t.TempDir(), "state.json")}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	listingService := listings.NewService(store, disabledNotifier{}, zap.NewNop())
	service := NewService(&sourceStub{body: []byte(`{"businesses":[{"id":4,"name":"Market","price":5000}]}`)}, NewMemoryRepository(), listingService, zap.NewNop())
	if err := service.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if snapshot := service.Snapshot(); snapshot.Count != 1 || snapshot.Businesses[0].ID != "4" {
		t.Fatalf("snapshot = %#v", snapshot)
	}
}

type disabledNotifier struct{}

func (disabledNotifier) Enabled() bool                                              { return false }
func (disabledNotifier) NotifyListing(context.Context, discord.ListingNotice) error { return nil }
