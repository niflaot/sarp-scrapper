package businesses

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/niflaot/sarp-scrapper/internal/module/listings"
	"github.com/niflaot/sarp-scrapper/pkg/discord"
	"github.com/niflaot/sarp-scrapper/pkg/review"
	"github.com/niflaot/sarp-scrapper/pkg/server"
	"go.uber.org/zap"
)

func TestBusinessRoutes(t *testing.T) {
	store, err := review.NewStore(review.Config{StateFile: filepath.Join(t.TempDir(), "state.json")}, zap.NewNop())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	listingService := listings.NewService(store, businessDisabledNotifier{}, zap.NewNop())
	service := NewService(&sourceStub{body: []byte(`{"businesses":[{"id":4,"name":"Market","price":5000}]}`)}, NewMemoryRepository(), listingService, zap.NewNop())
	httpServer := server.New(server.Config{Environment: server.EnvironmentDevelopment, BindHost: "127.0.0.1", BindPort: 8080}, zap.NewNop())
	Register(httpServer, NewHandler(service))
	refresh, err := httpServer.App().Test(httptest.NewRequest(http.MethodPost, "/businesses/refresh", nil))
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	_ = refresh.Body.Close()
	response, err := httpServer.App().Test(httptest.NewRequest(http.MethodGet, "/businesses", nil))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer response.Body.Close()
	var snapshot Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil || snapshot.Count != 1 {
		t.Fatalf("snapshot = %#v, %v", snapshot, err)
	}
}

type businessDisabledNotifier struct{}

func (businessDisabledNotifier) Enabled() bool { return false }
func (businessDisabledNotifier) NotifyListing(context.Context, discord.ListingNotice) error {
	return nil
}
